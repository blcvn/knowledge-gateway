# 09 — Graphiti Store Service

> **gRPC**: 9024 | **Health**: 9098

---

## 1. Purpose

Graph database abstraction: CRUD operations trên Neo4j (primary) với pluggable drivers cho FalkorDB, Kuzu, Neptune. Quản lý bi-temporal model, indexes, và schema.

---

## 2. Clean Architecture

```
services/graphiti-store/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # EntityNode, EpisodicNode, CommunityNode
│   │   ├── value_object.go     # NodeLabel, EdgeLabel, BiTemporal
│   │   └── errors.go
│   ├── usecase/
│   │   ├── get_entity.go
│   │   ├── create_entity.go
│   │   ├── update_entity.go
│   │   ├── delete_entity.go    # Soft-delete with tombstone
│   │   ├── create_edge.go
│   │   ├── list_edges.go
│   │   ├── get_episode.go
│   │   ├── save_community.go
│   │   ├── graph_query.go      # Raw Cypher passthrough (admin only)
│   │   ├── port/
│   │   │   └── output.go       # GraphDriver interface (core abstraction)
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/handler.go     # graphiti.store.v1.StoreService impl
│   │   ├── driver/             # GraphDB driver implementations
│   │   │   ├── interface.go    # GraphDriver interface definition
│   │   │   ├── neo4j/
│   │   │   │   ├── driver.go   # Neo4j bolt driver
│   │   │   │   ├── queries.go  # Cypher query templates
│   │   │   │   └── mapper.go   # Neo4j Record → Domain mapping
│   │   │   ├── falkordb/
│   │   │   │   └── driver.go   # FalkorDB (Redis Graph) driver
│   │   │   └── kuzu/
│   │   │       └── driver.go   # Kuzu embedded driver
│   │   └── repository/
│   │       └── postgres/       # Metadata (indexes, schema versions)
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
```

---

## 3. GraphDriver Interface (Core Abstraction)

```go
type GraphDriver interface {
    // Node operations
    CreateNode(ctx context.Context, node *domain.EntityNode) error
    GetNode(ctx context.Context, id uuid.UUID) (*domain.EntityNode, error)
    UpdateNode(ctx context.Context, node *domain.EntityNode) error
    DeleteNode(ctx context.Context, id uuid.UUID) error    // Soft-delete
    
    // Edge operations
    CreateEdge(ctx context.Context, edge *domain.EntityEdge) error
    GetEdges(ctx context.Context, nodeID uuid.UUID, dir Direction) ([]*domain.EntityEdge, error)
    InvalidateEdge(ctx context.Context, id uuid.UUID, invalidAt time.Time) error
    
    // Episode operations
    SaveEpisode(ctx context.Context, ep *domain.EpisodicNode) error
    GetEpisode(ctx context.Context, id uuid.UUID) (*domain.EpisodicNode, error)
    
    // Community operations
    SaveCommunity(ctx context.Context, c *domain.CommunityNode) error
    GetCommunities(ctx context.Context, groupID string) ([]*domain.CommunityNode, error)
    
    // Search support
    FullTextSearch(ctx context.Context, query string, groupID string, limit int) ([]*domain.EntityNode, error)
    GetNeighborhood(ctx context.Context, nodeID uuid.UUID, depth int) ([]*domain.EntityNode, []*domain.EntityEdge, error)
    
    // Bulk operations
    BulkCreateNodes(ctx context.Context, nodes []*domain.EntityNode) error
    BulkCreateEdges(ctx context.Context, edges []*domain.EntityEdge) error
    
    // Schema management
    EnsureIndexes(ctx context.Context) error
    Migrate(ctx context.Context, version int) error
    
    // Health
    Ping(ctx context.Context) error
    Close() error
}
```

---

## 4. Bi-Temporal Model

```go
type BiTemporal struct {
    // Transaction time (system-managed)
    CreatedAt   time.Time   // When the record was created
    UpdatedAt   time.Time   // When the record was last modified
    
    // Valid time (domain-managed)
    ValidFrom   time.Time   // When the fact became true
    ValidTo     *time.Time  // When the fact stopped being true (nil = still valid)
    InvalidAt   *time.Time  // When this edge was invalidated by newer info
}

type EntityNode struct {
    UUID        uuid.UUID
    GroupID     string              // Graphiti's tenant isolation
    Name        string
    EntityType  string
    Summary     string
    Embedding   []float32
    BiTemporal
}

type EntityEdge struct {
    UUID        uuid.UUID
    GroupID     string
    SourceUUID  uuid.UUID
    TargetUUID  uuid.UUID
    Name        string
    Fact        string              // The factual statement this edge represents
    Weight      float64
    EpisodeUUIDs []uuid.UUID        // Provenance
    Embedding   []float32
    BiTemporal
}
```

---

## 5. Neo4j Cypher Templates

```cypher
// Create entity node
MERGE (n:Entity {uuid: $uuid, group_id: $group_id})
SET n += $properties, n.updated_at = datetime()

// Create edge with bi-temporal
MATCH (s:Entity {uuid: $source_uuid}), (t:Entity {uuid: $target_uuid})
CREATE (s)-[r:RELATES_TO {
  uuid: $uuid, name: $name, fact: $fact,
  valid_from: datetime($valid_from),
  created_at: datetime()
}]->(t)

// Invalidate edge
MATCH ()-[r:RELATES_TO {uuid: $uuid}]->()
SET r.invalid_at = datetime($invalid_at), r.valid_to = datetime($valid_to)

// Temporal query: valid at point-in-time
MATCH (n:Entity {group_id: $group_id})-[r:RELATES_TO]->(m:Entity)
WHERE r.valid_from <= datetime($at) 
  AND (r.valid_to IS NULL OR r.valid_to > datetime($at))
  AND r.invalid_at IS NULL
RETURN n, r, m
```
