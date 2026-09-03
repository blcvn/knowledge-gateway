# TASK-GR-015 — Custom Ontology Storage & Validation

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-015 |
| **Wave** | 3 |
| **Component** | `services/graphiti-knowledge/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-005 §4, §5 |
| **Priority** | High |
| **Depends On** | TASK-GR-001, TASK-GR-008 |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-knowledge custom ontology support  
---

## Context

Implement `OntologyRepository` (PostgreSQL + Redis cache) và `ManageOntologyUseCase`. Extend `ExtractEntitiesUseCase` và `ExtractEdgesUseCase` để tự động lookup + validate ontology khi extraction.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-knowledge/internal/adapter/repository/postgres/ontology_repo.go` |
| CREATE | `services/graphiti-knowledge/internal/usecase/manage_ontology.go` |
| CREATE | `db/migrations/0023_graphiti_ontology.up.sql` |
| MODIFY | `services/graphiti-knowledge/internal/usecase/extract_entities.go` |
| MODIFY | `services/graphiti-knowledge/internal/usecase/extract_edges.go` |

---

## Implementation

### File 1: `db/migrations/0023_graphiti_ontology.up.sql`

```sql
CREATE TABLE IF NOT EXISTS graphiti_ontology_registries (
    group_id     TEXT PRIMARY KEY,
    entity_types JSONB NOT NULL DEFAULT '{}',
    edge_types   JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_graphiti_ontology_updated
    ON graphiti_ontology_registries(updated_at DESC);
```

### File 2: `services/graphiti-knowledge/internal/adapter/repository/postgres/ontology_repo.go`

```go
package postgres

import (
    "context"
    "encoding/json"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
    "github.com/vnp-memory/pkg/graph"
)

type OntologyRepository struct {
    db    *pgxpool.Pool
    cache redis.UniversalClient
}

func NewOntologyRepository(db *pgxpool.Pool, cache redis.UniversalClient) *OntologyRepository {
    return &OntologyRepository{db: db, cache: cache}
}

const cacheTTL = 30 * time.Minute

func (r *OntologyRepository) Save(ctx context.Context, registry graph.OntologyRegistry) error {
    entityJSON, _ := json.Marshal(registry.EntityTypes)
    edgeJSON, _   := json.Marshal(registry.EdgeTypes)

    _, err := r.db.Exec(ctx, `
        INSERT INTO graphiti_ontology_registries (group_id, entity_types, edge_types, updated_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (group_id) DO UPDATE
            SET entity_types = $2, edge_types = $3, updated_at = NOW()
    `, registry.GroupID, entityJSON, edgeJSON)
    if err != nil { return err }

    r.cache.Del(ctx, "ontology:"+registry.GroupID)
    return nil
}

func (r *OntologyRepository) Get(ctx context.Context, groupID string) (*graph.OntologyRegistry, error) {
    // Cache check
    if data, err := r.cache.Get(ctx, "ontology:"+groupID).Bytes(); err == nil {
        var reg graph.OntologyRegistry
        if err := json.Unmarshal(data, &reg); err == nil { return &reg, nil }
    }

    var entityJSON, edgeJSON []byte
    var createdAt, updatedAt time.Time
    err := r.db.QueryRow(ctx, `
        SELECT entity_types, edge_types, created_at, updated_at
        FROM graphiti_ontology_registries WHERE group_id = $1
    `, groupID).Scan(&entityJSON, &edgeJSON, &createdAt, &updatedAt)

    if err == pgx.ErrNoRows { return nil, nil }
    if err != nil { return nil, err }

    registry := &graph.OntologyRegistry{GroupID: groupID, CreatedAt: createdAt, UpdatedAt: updatedAt}
    json.Unmarshal(entityJSON, &registry.EntityTypes)
    json.Unmarshal(edgeJSON,   &registry.EdgeTypes)

    // Store in cache
    if data, err := json.Marshal(registry); err == nil {
        r.cache.Set(ctx, "ontology:"+groupID, data, cacheTTL)
    }
    return registry, nil
}

func (r *OntologyRepository) Delete(ctx context.Context, groupID string) error {
    _, err := r.db.Exec(ctx, `DELETE FROM graphiti_ontology_registries WHERE group_id = $1`, groupID)
    r.cache.Del(ctx, "ontology:"+groupID)
    return err
}
```

### File 3: `services/graphiti-knowledge/internal/usecase/manage_ontology.go`

```go
package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/pkg/graph/presets"
)

type OntologyRepository interface {
    Save(ctx context.Context, registry graph.OntologyRegistry) error
    Get(ctx context.Context, groupID string) (*graph.OntologyRegistry, error)
    Delete(ctx context.Context, groupID string) error
}

type ManageOntologyUseCase struct {
    repo OntologyRepository
}

func NewManageOntologyUseCase(repo OntologyRepository) *ManageOntologyUseCase {
    return &ManageOntologyUseCase{repo: repo}
}

func (uc *ManageOntologyUseCase) Save(ctx context.Context, registry graph.OntologyRegistry) error {
    registry.UpdatedAt = time.Now()
    if registry.CreatedAt.IsZero() { registry.CreatedAt = time.Now() }
    return uc.repo.Save(ctx, registry)
}

func (uc *ManageOntologyUseCase) Get(ctx context.Context, groupID string) (*graph.OntologyRegistry, error) {
    return uc.repo.Get(ctx, groupID)
}

func (uc *ManageOntologyUseCase) Delete(ctx context.Context, groupID string) error {
    return uc.repo.Delete(ctx, groupID)
}

func (uc *ManageOntologyUseCase) ApplyPreset(ctx context.Context, groupID, presetName string) error {
    preset, ok := presets.PresetByName[presetName]
    if !ok {
        available := make([]string, 0)
        for k := range presets.PresetByName { available = append(available, k) }
        return fmt.Errorf("unknown preset %q (available: %v)", presetName, available)
    }
    registry := *preset
    registry.GroupID   = groupID
    registry.UpdatedAt = time.Now()
    return uc.repo.Save(ctx, registry)
}
```

### MODIFY `extract_entities.go` — add ontology lookup

Add at the beginning of `Execute`:
```go
// Resolve ontology (priority: inline > registered > learned)
func (uc *ExtractEntitiesUseCase) resolveOntology(ctx context.Context, req port.ExtractEntitiesReq) map[string]graph.EntityTypeSchema {
    if len(req.EntityTypes) > 0 { return req.EntityTypes }   // inline override

    if req.GroupID != "" && uc.ontologyRepo != nil {
        if reg, err := uc.ontologyRepo.Get(ctx, req.GroupID); err == nil && reg != nil {
            if len(reg.EntityTypes) > 0 { return reg.EntityTypes }
        }
    }
    return nil  // learned mode
}
```

---

## Verification

```bash
cd services/graphiti-knowledge
go build ./...

# Test migration
psql $DB_URL -f db/migrations/0023_graphiti_ontology.up.sql
```

**Acceptance tests:**
1. POST ontology for group-A → extraction restricted to defined types
2. Extract episode for group-A → only prescribed entity types extracted
3. GET ontology → returns saved schema
4. DELETE ontology → group-A reverts to learned mode
5. Apply preset "hr" → HR entity types active
