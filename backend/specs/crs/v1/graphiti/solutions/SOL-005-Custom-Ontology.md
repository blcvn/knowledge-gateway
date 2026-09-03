# Solution: SOL-005 — Custom Ontology & Prescribed Entity/Edge Types

**CR ID:** CR-GR-005  
**Solution ID:** SOL-005  
**Priority:** High (Wave 3)  
**Architecture:** EXTEND `services/graphiti-knowledge/` + `services/graphiti-ingestion/` + PostgreSQL

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- `graphiti-knowledge` đã có `ExtractEntities` và prompt registry.
- **PostgreSQL** đã configured — lưu `OntologyRegistry` per group_id.
- **Redis** — có thể cache ontology lookups (hot data).
- Extraction prompt đã có logic cho entity types (từ SOL-003) — cần persist + serve.

---

## 2. Shared Package — `pkg/graph/ontology.go`

```go
// pkg/graph/ontology.go

package graph

// EntityTypeSchema — prescribed entity type definition
type EntityTypeSchema struct {
    Name        string              `json:"name"`
    Description string              `json:"description"`
    Properties  []OntologyProperty  `json:"properties,omitempty"`
    Examples    []string            `json:"examples,omitempty"`
}

// EdgeTypeSchema — prescribed relationship definition
type EdgeTypeSchema struct {
    Name        string              `json:"name"`
    Description string              `json:"description"`
    SourceTypes []string            `json:"source_types,omitempty"`
    TargetTypes []string            `json:"target_types,omitempty"`
    Properties  []OntologyProperty  `json:"properties,omitempty"`
    Examples    []string            `json:"examples,omitempty"`
}

type OntologyProperty struct {
    Name        string `json:"name"`
    Type        string `json:"type"`       // string | number | boolean | datetime
    Description string `json:"description"`
    Required    bool   `json:"required"`
}

// OntologyRegistry — per group_id ontology configuration
type OntologyRegistry struct {
    GroupID     string
    EntityTypes map[string]EntityTypeSchema
    EdgeTypes   map[string]EdgeTypeSchema
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

---

## 3. Domain Presets — `pkg/graph/presets/`

```go
// pkg/graph/presets/hr.go

package presets

import "github.com/vnp-memory/pkg/graph"

var HRPreset = graph.OntologyRegistry{
    EntityTypes: map[string]graph.EntityTypeSchema{
        "Employee": {
            Name:        "Employee",
            Description: "A person employed at the organization",
            Properties: []graph.OntologyProperty{
                {Name: "department", Type: "string"},
                {Name: "role", Type: "string"},
                {Name: "start_date", Type: "datetime"},
            },
            Examples: []string{"Alice the Software Engineer", "Bob in HR"},
        },
        "Department": {
            Name:        "Department",
            Description: "An organizational unit or division",
            Examples:    []string{"Engineering", "Human Resources", "Sales"},
        },
        "Role": {
            Name:        "Role",
            Description: "A job title or position",
            Examples:    []string{"Senior Software Engineer", "VP of Sales"},
        },
        "Project": {
            Name:        "Project",
            Description: "A work project or initiative",
            Properties:  []graph.OntologyProperty{{Name: "deadline", Type: "datetime"}},
        },
    },
    EdgeTypes: map[string]graph.EdgeTypeSchema{
        "REPORTS_TO": {
            Name:        "REPORTS_TO",
            Description: "Employee reports to another employee (management chain)",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Employee"},
        },
        "WORKS_IN": {
            Name:        "WORKS_IN",
            Description: "Employee works in a department",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Department"},
        },
        "HAS_ROLE": {
            Name:        "HAS_ROLE",
            Description: "Employee holds a role or job title",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Role"},
        },
        "WORKS_ON": {
            Name:        "WORKS_ON",
            Description: "Employee works on a project",
            SourceTypes: []string{"Employee"},
            TargetTypes: []string{"Project"},
        },
    },
}
```

```go
// pkg/graph/presets/crm.go

var CRMPreset = graph.OntologyRegistry{
    EntityTypes: map[string]graph.EntityTypeSchema{
        "Customer": {
            Name:        "Customer",
            Description: "A person or organization that is a customer or prospect",
            Properties: []graph.OntologyProperty{
                {Name: "company", Type: "string"},
                {Name: "email", Type: "string"},
                {Name: "tier", Type: "string"},
            },
        },
        "Deal": {
            Name:        "Deal",
            Description: "A sales opportunity or deal in the pipeline",
            Properties: []graph.OntologyProperty{
                {Name: "value", Type: "number"},
                {Name: "stage", Type: "string"},
                {Name: "close_date", Type: "datetime"},
            },
        },
        "Product":  {Name: "Product", Description: "A product or service offered"},
        "SalesRep": {Name: "SalesRep", Description: "A sales representative"},
    },
    EdgeTypes: map[string]graph.EdgeTypeSchema{
        "BOUGHT":      {SourceTypes: []string{"Customer"}, TargetTypes: []string{"Product"}},
        "INTERESTED_IN": {SourceTypes: []string{"Customer"}, TargetTypes: []string{"Product", "Deal"}},
        "ASSIGNED_TO": {SourceTypes: []string{"Deal"}, TargetTypes: []string{"SalesRep"}},
    },
}
```

```go
// pkg/graph/presets/software.go

var SoftwareProjectPreset = graph.OntologyRegistry{
    EntityTypes: map[string]graph.EntityTypeSchema{
        "Developer":  {Name: "Developer", Description: "A software developer or engineer"},
        "Service":    {Name: "Service", Description: "A microservice or application component"},
        "Repository": {Name: "Repository", Description: "A code repository"},
        "Feature":    {Name: "Feature", Description: "A software feature or user story"},
        "Bug":        {Name: "Bug", Description: "A software bug or defect"},
    },
    EdgeTypes: map[string]graph.EdgeTypeSchema{
        "OWNS":       {SourceTypes: []string{"Developer"}, TargetTypes: []string{"Service", "Repository"}},
        "DEPENDS_ON": {SourceTypes: []string{"Service"}, TargetTypes: []string{"Service"}},
        "IMPLEMENTS": {SourceTypes: []string{"Developer"}, TargetTypes: []string{"Feature"}},
        "FIXES":      {SourceTypes: []string{"Developer"}, TargetTypes: []string{"Bug"}},
        "REPORTED_IN": {SourceTypes: []string{"Bug"}, TargetTypes: []string{"Service"}},
    },
}

var PresetByName = map[string]*graph.OntologyRegistry{
    "hr":               &HRPreset,
    "crm":              &CRMPreset,
    "software_project": &SoftwareProjectPreset,
}
```

---

## 4. PostgreSQL Storage — `services/graphiti-knowledge/internal/adapter/repository/`

### 4.1. Database Schema

```sql
-- db/migrations/0023_graphiti_ontology.up.sql

CREATE TABLE graphiti_ontology_registries (
    group_id     TEXT PRIMARY KEY,
    entity_types JSONB NOT NULL DEFAULT '{}',
    edge_types   JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_graphiti_ontology_updated ON graphiti_ontology_registries(updated_at DESC);
```

### 4.2. Repository

```go
// services/graphiti-knowledge/internal/adapter/repository/postgres/ontology_repo.go

type OntologyRepository struct {
    db    *pgxpool.Pool
    cache *redis.Client  // TTL cache for hot ontology lookups
}

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

    // Invalidate cache
    r.cache.Del(ctx, "ontology:"+registry.GroupID)
    return nil
}

func (r *OntologyRepository) Get(ctx context.Context, groupID string) (*graph.OntologyRegistry, error) {
    // Check cache first
    if cached, ok := r.getFromCache(ctx, groupID); ok { return cached, nil }

    var entityJSON, edgeJSON []byte
    var createdAt, updatedAt time.Time
    err := r.db.QueryRow(ctx, `
        SELECT entity_types, edge_types, created_at, updated_at
        FROM graphiti_ontology_registries WHERE group_id = $1
    `, groupID).Scan(&entityJSON, &edgeJSON, &createdAt, &updatedAt)
    if err == pgx.ErrNoRows { return nil, nil }
    if err != nil { return nil, err }

    registry := &graph.OntologyRegistry{
        GroupID:   groupID,
        CreatedAt: createdAt,
        UpdatedAt: updatedAt,
    }
    json.Unmarshal(entityJSON, &registry.EntityTypes)
    json.Unmarshal(edgeJSON,   &registry.EdgeTypes)

    // Store in cache (30 min TTL)
    r.setInCache(ctx, groupID, registry, 30*time.Minute)
    return registry, nil
}

func (r *OntologyRepository) Delete(ctx context.Context, groupID string) error {
    _, err := r.db.Exec(ctx, `DELETE FROM graphiti_ontology_registries WHERE group_id = $1`, groupID)
    r.cache.Del(ctx, "ontology:"+groupID)
    return err
}
```

---

## 5. Ontology Use Cases — `services/graphiti-knowledge/internal/usecase/`

### 5.1. Manage Ontology

```go
// services/graphiti-knowledge/internal/usecase/manage_ontology.go

type ManageOntologyUseCase struct {
    repo port.OntologyRepository
}

func (uc *ManageOntologyUseCase) Save(ctx context.Context, registry graph.OntologyRegistry) error {
    registry.UpdatedAt = time.Now()
    return uc.repo.Save(ctx, registry)
}

func (uc *ManageOntologyUseCase) Get(ctx context.Context, groupID string) (*graph.OntologyRegistry, error) {
    return uc.repo.Get(ctx, groupID)
}

func (uc *ManageOntologyUseCase) Delete(ctx context.Context, groupID string) error {
    return uc.repo.Delete(ctx, groupID)
}

// ApplyPreset replaces ontology with a named preset
func (uc *ManageOntologyUseCase) ApplyPreset(ctx context.Context, groupID, presetName string) error {
    preset, ok := presets.PresetByName[presetName]
    if !ok { return fmt.Errorf("unknown preset: %s (available: hr, crm, software_project)", presetName) }

    registry := *preset
    registry.GroupID = groupID
    registry.UpdatedAt = time.Now()
    return uc.repo.Save(ctx, registry)
}
```

### 5.2. Ontology-Aware Extraction

```go
// services/graphiti-knowledge/internal/usecase/extract_entities.go
// (EXTEND existing ExtractEntitiesUseCase from SOL-003)

func (uc *ExtractEntitiesUseCase) resolveOntology(ctx context.Context, req ExtractEntitiesReq) map[string]graph.EntityTypeSchema {
    // Priority order for ontology:
    // 1. Inline (per-episode override) — from req.EntityTypes
    if len(req.EntityTypes) > 0 { return req.EntityTypes }

    // 2. Registered per group_id
    if req.GroupID != "" {
        if registry, err := uc.ontologyRepo.Get(ctx, req.GroupID); err == nil && registry != nil {
            if len(registry.EntityTypes) > 0 { return registry.EntityTypes }
        }
    }

    // 3. No ontology → learned mode (nil = any label allowed)
    return nil
}

// validateExtractedEntities filters entities against ontology if prescribed
func validateExtractedEntities(entities []ExtractedEntity, ontology map[string]graph.EntityTypeSchema) []ExtractedEntity {
    if len(ontology) == 0 { return entities }  // learned mode: no filtering

    var valid []ExtractedEntity
    for _, e := range entities {
        if _, ok := ontology[e.Label]; ok {
            valid = append(valid, e)
        }
        // else: silently discard entity with invalid label
    }
    return valid
}

// validateExtractedEdges filters edges against edge ontology
func validateExtractedEdges(edges []graph.EntityEdge, ontology map[string]graph.EdgeTypeSchema, entityMap map[string]string) []graph.EntityEdge {
    if len(ontology) == 0 { return edges }

    var valid []graph.EntityEdge
    for _, e := range edges {
        schema, ok := ontology[e.Name]
        if !ok { continue }  // edge type not in ontology

        // Validate source/target types
        srcLabel := entityMap[e.SourceNodeUUID]
        tgtLabel := entityMap[e.TargetNodeUUID]

        if len(schema.SourceTypes) > 0 && !contains(schema.SourceTypes, srcLabel) { continue }
        if len(schema.TargetTypes) > 0 && !contains(schema.TargetTypes, tgtLabel) { continue }

        valid = append(valid, e)
    }
    return valid
}
```

---

## 6. gRPC API Extension — Knowledge Service

```protobuf
// api/proto/graphiti/knowledge/v1/knowledge.proto (additions)

service KnowledgeService {
    // ... existing RPCs from SOL-003 ...

    // Ontology management (NEW)
    rpc SaveOntology(SaveOntologyRequest) returns (SaveOntologyResponse);
    rpc GetOntology(GetOntologyRequest) returns (GetOntologyResponse);
    rpc DeleteOntology(DeleteOntologyRequest) returns (DeleteOntologyResponse);
    rpc ApplyOntologyPreset(ApplyOntologyPresetRequest) returns (ApplyOntologyPresetResponse);
    rpc ListOntologyPresets(ListOntologyPresetsRequest) returns (ListOntologyPresetsResponse);
}

message SaveOntologyRequest {
    string group_id = 1;
    map<string, EntityTypeSchemaProto> entity_types = 2;
    map<string, EdgeTypeSchemaProto>   edge_types   = 3;
}

message EntityTypeSchemaProto {
    string name        = 1;
    string description = 2;
    repeated OntologyPropertyProto properties = 3;
    repeated string examples = 4;
}

message EdgeTypeSchemaProto {
    string name        = 1;
    string description = 2;
    repeated string source_types = 3;
    repeated string target_types = 4;
    repeated OntologyPropertyProto properties = 5;
}

message ApplyOntologyPresetRequest {
    string group_id    = 1;
    string preset_name = 2;  // "hr" | "crm" | "software_project"
}

message ListOntologyPresetsResponse {
    repeated string preset_names = 1;  // available preset names
}
```

---

## 7. Gateway Routes (NEW for CR-005)

```go
// gateway/internal/adapter/handler/router.go

// Ontology routes (via graphiti-knowledge gRPC)
r.Post("/v1/graphiti/ontology/{group_id}",         h.SaveOntology)
r.Get("/v1/graphiti/ontology/{group_id}",          h.GetOntology)
r.Delete("/v1/graphiti/ontology/{group_id}",       h.DeleteOntology)
r.Post("/v1/graphiti/ontology/{group_id}/preset",  h.ApplyOntologyPreset)
r.Get("/v1/graphiti/ontology/presets",             h.ListOntologyPresets)
```

---

## 8. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `pkg/graph/ontology.go` | EntityTypeSchema, EdgeTypeSchema, OntologyRegistry |
| `pkg/graph/presets/hr.go` | HR domain preset |
| `pkg/graph/presets/crm.go` | CRM domain preset |
| `pkg/graph/presets/software.go` | Software Project preset |
| `services/graphiti-knowledge/internal/usecase/manage_ontology.go` | Save/Get/Delete/ApplyPreset |
| `services/graphiti-knowledge/internal/adapter/repository/postgres/ontology_repo.go` | Postgres + Redis cache |
| `db/migrations/0023_graphiti_ontology.up.sql` | graphiti_ontology_registries table |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/graphiti-knowledge/internal/usecase/extract_entities.go` | + ontology lookup + validation |
| `services/graphiti-knowledge/internal/usecase/extract_edges.go` | + edge type validation |
| `services/graphiti-knowledge/internal/adapter/grpc/handler.go` | + 5 ontology RPCs |
| `api/proto/graphiti/knowledge/v1/knowledge.proto` | + ontology message types + RPCs |
| `gateway/internal/adapter/handler/router.go` | + 5 ontology routes |
| `gateway/internal/adapter/handler/graphiti_handler.go` | + ontology handlers |

---

## 9. Acceptance Criteria Mapping

| AC từ CR-GR-005 | Covered by |
|----------------|-----------|
| POST /v1/graphiti/ontology/project-alpha với CRM → ingest "Acme bought 10 licenses" → Customer + Product extracted | resolveOntology() + validateExtractedEntities() |
| Entity not matching prescribed types → NOT extracted | validateExtractedEntities() filter |
| Edge BOUGHT source=Company → rejected (type mismatch) | validateExtractedEdges() SourceTypes check |
| GET /v1/graphiti/ontology/{group_id} → registered schema | GetOntology() → ontologyRepo.Get() |
| POST /v1/graphiti/ontology/{group_id}/preset hr → HR ontology applied | ApplyPreset() → presets.PresetByName["hr"] |
| Learned mode (no ontology) → LLM free labels | resolveOntology() returns nil |
| Ontology cleared → back to learned | DeleteOntology() → repo.Delete() |
| Per-episode inline ontology overrides registered | Priority order in resolveOntology() |
