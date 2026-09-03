---
id: FEAT-STO-001
title: Domain Layer — Graph Entity Types
service: graphiti-store
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement domain layer cho graphiti-store: graph entity types (EntityNode, EpisodicNode, CommunityNode, SagaNode), edge types (EntityEdge — bi-temporal), value objects, GraphDriver interface, và domain errors.

## Bối Cảnh Nghiệp Vụ

graphiti-store là abstraction layer cho graph database. Domain layer định nghĩa tất cả graph types mà các service khác (pipeline, search) sử dụng. GraphDriver interface là Strategy pattern cho phép swap backend (Neo4j → FalkorDB → Kuzu).

## Scope

### In Scope
- `internal/domain/entity.go` — EntityNode, EpisodicNode, CommunityNode, SagaNode
- `internal/domain/edge.go` — EntityEdge (bi-temporal), EpisodicEdge
- `internal/domain/value_object.go` — NodeLabel, EdgeType, GroupID, UUID, EmbeddingVector
- `internal/domain/index.go` — IndexDefinition, IndexType
- `internal/domain/driver.go` — GraphDriver composite interface
- `internal/domain/search.go` — SearchParams, SearchResult, SimilarityMetric
- `internal/domain/errors.go` — ErrNodeNotFound, ErrEdgeNotFound, ErrDriverNotSupported, ErrTransactionFailed

### Out of Scope
- Repository implementations (FEAT-STO-003..007)
- gRPC handlers (FEAT-STO-008)

## Thiết Kế Kỹ Thuật

### Entity Types

```go
type EntityNode struct {
    UUID          string            `json:"uuid"`
    Name          string            `json:"name"`
    GroupID       string            `json:"group_id"`
    Summary       string            `json:"summary"`
    NameEmbedding []float32         `json:"name_embedding"`
    Labels        []string          `json:"labels"`
    Attributes    map[string]string `json:"attributes"`
    CreatedAt     time.Time         `json:"created_at"`
    UpdatedAt     time.Time         `json:"updated_at"`
}

type EpisodicNode struct {
    UUID        string    `json:"uuid"`
    Name        string    `json:"name"`
    GroupID     string    `json:"group_id"`
    Content     string    `json:"content"`
    Source      string    `json:"source"`
    ValidAt     time.Time `json:"valid_at"`
    EntityEdges []string  `json:"entity_edges"` // UUIDs of related entity edges
    CreatedAt   time.Time `json:"created_at"`
}

type EntityEdge struct {
    UUID          string            `json:"uuid"`
    SourceNodeID  string            `json:"source_node_id"`
    TargetNodeID  string            `json:"target_node_id"`
    Name          string            `json:"name"`
    GroupID       string            `json:"group_id"`
    Fact          string            `json:"fact"`
    FactEmbedding []float32         `json:"fact_embedding"`
    ValidAt       time.Time         `json:"valid_at"`
    InvalidAt     *time.Time        `json:"invalid_at,omitempty"` // NULL = still valid
    ExpiredAt     *time.Time        `json:"expired_at,omitempty"` // Superseded by newer edge
    Attributes    map[string]string `json:"attributes"`
    EpisodeID     string            `json:"episode_id"`
    CreatedAt     time.Time         `json:"created_at"`
}
```

### GraphDriver Interface

```go
type GraphDriver interface {
    NodeRepository
    EdgeRepository
    CommunityRepository
    SearchRepository
    IndexRepository
    BulkRepository
    TransactionManager
    io.Closer
}
```

### Bi-Temporal Validation Rules

```go
func (e *EntityEdge) Validate() error {
    if e.ValidAt.IsZero() { return ErrInvalidTemporalData }
    if e.InvalidAt != nil && e.InvalidAt.Before(e.ValidAt) { return ErrInvalidTemporalRange }
    if e.ExpiredAt != nil && e.ExpiredAt.Before(e.CreatedAt) { return ErrInvalidExpiration }
    return nil
}
```

## Acceptance Criteria

- [ ] AC-1: Domain layer compiles with ZERO external imports (only Go stdlib)
- [ ] AC-2: All entity types have JSON tags for serialization
- [ ] AC-3: EntityEdge.Validate() enforces: valid_at required, invalid_at > valid_at
- [ ] AC-4: GraphDriver interface composes all 7 repository interfaces
- [ ] AC-5: Domain errors are sentinel errors with `errors.Is()` support
- [ ] AC-6: EmbeddingVector type supports cosine distance calculation

## Test Requirements
- **Unit tests**: Entity validation, value object constructors, bi-temporal rules
- **Minimum coverage**: 90%
