# TASK-ZEP-011 — services/zep-graph: Domain Model & Neo4j Repository

**Task ID:** TASK-ZEP-011  
**Wave:** 4 (Graph Intelligence)  
**Solution:** [SOL-ZEP-003](../solutions/SOL-ZEP-003-Temporal-Knowledge-Graph.md)  
**Depends on:** TASK-ZEP-010 (Neo4j + Graphiti infrastructure)  
**Ước tính:** 4h  
**Priority:** Critical

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-graph: 6 .go - graph domain + Neo4j queries  
---

## Mục tiêu

Tạo domain layer và Neo4j repository cho `services/zep-graph/`:
- `EntityNode` (9 NodeTypes với priority hierarchy)
- `TemporalEdge` (Fact với `ValidAt`/`InvalidAt`/`ExpiredAt`)
- `Episode` (source message windows)
- `GraphOntology` (custom ontology definitions)
- Neo4j Cypher repository implementations

---

## Công việc cụ thể

### 1. Tạo `services/zep-graph/internal/domain/node.go`

```go
package domain

import "time"

type NodeType string

const (
    NodeTypeUser         NodeType = "User"          // Priority 1
    NodeTypeAssistant    NodeType = "Assistant"      // Priority 1
    NodeTypePreference   NodeType = "Preference"     // Priority 2
    NodeTypeOrganization NodeType = "Organization"   // Priority 3
    NodeTypeEvent        NodeType = "Event"          // Priority 3
    NodeTypeLocation     NodeType = "Location"       // Priority 4
    NodeTypeDocument     NodeType = "Document"       // Priority 4
    NodeTypeTopic        NodeType = "Topic"          // Priority 5
    NodeTypeObject       NodeType = "Object"         // Priority 6: catch-all
)

var NodeTypePriority = map[NodeType]int{
    NodeTypeUser: 1, NodeTypeAssistant: 1,
    NodeTypePreference: 2,
    NodeTypeOrganization: 3, NodeTypeEvent: 3,
    NodeTypeLocation: 4, NodeTypeDocument: 4,
    NodeTypeTopic: 5,
    NodeTypeObject: 6,
}

type EntityNode struct {
    UUID      string
    GroupID   string         // session_id hoặc user_id (scoping)
    Name      string         // entity name (e.g. "Alice", "Beta Inc")
    NodeType  NodeType
    Summary   string         // AI-generated summary
    Metadata  map[string]any
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2. Tạo `services/zep-graph/internal/domain/edge.go`

```go
// TemporalEdge là Fact với temporal annotations — core Zep differentiator
type TemporalEdge struct {
    UUID         string
    GroupID      string
    SourceNodeID string       // source EntityNode UUID
    TargetNodeID string       // target EntityNode UUID
    Name         string       // relationship label (WORKS_AT, LIVES_IN, etc.)
    Fact         string       // human-readable: "Alice works at Beta Inc"
    FactRating   float64      // 0.0-1.0 quality score from LLM

    // TEMPORAL ANNOTATIONS — the Zep differentiator
    ValidAt    *time.Time     // khi fact bắt đầu đúng (nil = unknown)
    InvalidAt  *time.Time     // khi fact không còn đúng (nil = still valid)
    ExpiredAt  *time.Time     // khi bị superseded bởi fact mới hơn (nil = current)

    Episodes  []string        // episode UUIDs mentioning this edge
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Invalidate marks fact as no longer valid (soft invalidate)
func (e *TemporalEdge) Invalidate() {
    now := time.Now()
    e.InvalidAt = &now
}

// IsCurrent returns true nếu fact vẫn còn hiệu lực
func (e *TemporalEdge) IsCurrent() bool {
    return e.InvalidAt == nil && e.ExpiredAt == nil
}
```

### 3. Tạo `services/zep-graph/internal/domain/episode.go` và `ontology.go`

```go
// Episode = source message window
type Episode struct {
    UUID      string
    GroupID   string
    Name      string    // "{groupID}-{messageUUID}"
    Content   string    // raw message text
    Source    string    // "message" | "graph_data"
    CreatedAt time.Time
}

// GraphOntology = custom entity/edge type definitions
type GraphOntology struct {
    GraphID  string
    Entities map[string]EntityDefinition  // custom node types
    Edges    map[string]EdgeDefinition    // custom edge types
    SetAt    time.Time
}

type EntityDefinition struct {
    Description string   `json:"description"`
    Fields      []string `json:"fields"`
}

type EdgeDefinition struct {
    Description string     `json:"description"`
    SourceTypes []NodeType `json:"source_types"`
    TargetTypes []NodeType `json:"target_types"`
}
```

### 4. Tạo Repository Interfaces

```go
// services/zep-graph/internal/domain/repository.go
type NodeRepository interface {
    Upsert(ctx context.Context, node *EntityNode) error
    GetByUUID(ctx context.Context, uuid string) (*EntityNode, error)
    ListByGroup(ctx context.Context, groupID string, limit int) ([]*EntityNode, error)
}

type EdgeRepository interface {
    Upsert(ctx context.Context, edge *TemporalEdge) error
    GetByUUID(ctx context.Context, uuid string) (*TemporalEdge, error)
    Update(ctx context.Context, edge *TemporalEdge) error
    ListByGroup(ctx context.Context, groupID string, limit int) ([]*TemporalEdge, error)
    // Invalidate: set invalid_at = now()
    Invalidate(ctx context.Context, uuid string) error
}

type EpisodeRepository interface {
    Upsert(ctx context.Context, episode *Episode) error
    GetByUUID(ctx context.Context, uuid string) (*Episode, error)
    ListByGroup(ctx context.Context, groupID string, lastN int) ([]*Episode, error)
    GetMentions(ctx context.Context, episodeUUID string) (nodes []*EntityNode, edges []*TemporalEdge, err error)
}

type OntologyRepository interface {
    Set(ctx context.Context, ontology *GraphOntology) error
    GetForGroup(ctx context.Context, groupID string) (*GraphOntology, error)
}
```

### 5. Implement Neo4j Repositories

**`services/zep-graph/internal/infra/neo4j/node_repo.go`**

```cypher
// Upsert node:
MERGE (n:Entity {uuid: $uuid})
ON CREATE SET n += {group_id: $group_id, name: $name, node_type: $node_type, 
                    summary: $summary, created_at: datetime(), updated_at: datetime()}
ON MATCH SET n.updated_at = datetime(), n.summary = $summary

// ListByGroup:
MATCH (n:Entity {group_id: $group_id})
RETURN n ORDER BY n.created_at DESC LIMIT $limit
```

**`services/zep-graph/internal/infra/neo4j/edge_repo.go`**

```cypher
// Upsert temporal edge (dynamic relationship type):
MATCH (src:Entity {uuid: $source_uuid})
MATCH (tgt:Entity {uuid: $target_uuid})
MERGE (src)-[r:RELATIONSHIP {uuid: $uuid}]->(tgt)
ON CREATE SET r += {name: $name, fact: $fact, fact_rating: $fact_rating,
                    valid_at: $valid_at, invalid_at: null, expired_at: null,
                    group_id: $group_id, created_at: datetime()}
ON MATCH SET r.updated_at = datetime(), r.fact_rating = $fact_rating

// Invalidate:
MATCH ()-[r {uuid: $uuid}]-()
SET r.invalid_at = datetime()
```

---

## Acceptance Criteria

- [ ] `go build ./services/zep-graph/...` không có lỗi
- [ ] `TemporalEdge.Invalidate()` sets `InvalidAt` to non-nil
- [ ] `IsCurrent()` returns false khi `InvalidAt != nil`
- [ ] Neo4j Upsert: cùng UUID → không duplicate, update ON MATCH
- [ ] EdgeRepository.Invalidate → Cypher set `r.invalid_at = datetime()`
- [ ] EpisodeRepository.GetMentions → trả về nodes + edges cho episode

---

## Files tạo ra

```
services/zep-graph/
├── internal/
│   ├── domain/
│   │   ├── node.go
│   │   ├── edge.go
│   │   ├── episode.go
│   │   ├── ontology.go
│   │   └── repository.go
│   └── infra/
│       └── neo4j/
│           ├── node_repo.go
│           ├── edge_repo.go
│           ├── episode_repo.go
│           └── ontology_repo.go
```

## Sau khi hoàn thành

Chạy: `go build ./services/zep-graph/...`
