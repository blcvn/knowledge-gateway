---
id: TDD-graphiti-store
title: Technical Design — graphiti-store
service: graphiti-store
version: 2.0.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Graphiti
linked_sol: SOL-001
---

# Technical Design — graphiti-store

> **Group**: Graphiti | **gRPC Port**: 9024 | **Health Port**: 9097

## 1. Service Overview

Graph database abstraction layer with pluggable backends. All graph CRUD operations, bi-temporal edge management, transactions, index management, and search primitives are routed through this service. Neo4j is the primary backend; FalkorDB, Kuzu, Neptune are pluggable alternatives.

**Key Characteristics:**
- GraphDriver interface: Strategy pattern for backend selection
- Bi-temporal edge model (valid_at, invalid_at, expired_at, created_at)
- Atomic bulk operations (SaveBulk with rollback support)
- Search primitives: cosine similarity, fulltext BM25, BFS traversal
- Index management abstracted across backends
- Transaction support via unit-of-work pattern

## 2. Clean Architecture Layers

### 2.1 Domain Layer

```
internal/domain/
├── entity.go          # EntityNode, EpisodicNode, CommunityNode, SagaNode
├── edge.go            # EntityEdge (bi-temporal), EpisodicEdge
├── value_object.go    # NodeLabel, EdgeType, GroupID, UUID, EmbeddingVector
├── index.go           # IndexDefinition, IndexType (vector, fulltext, composite)
├── driver.go          # GraphDriver interface (composition of all repositories)
├── search.go          # SearchParams, SearchResult, SimilarityMetric
└── errors.go          # ErrNodeNotFound, ErrDriverNotSupported, ErrTransactionFailed
```

**GraphDriver Interface:**
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

type NodeRepository interface {
    SaveNode(ctx context.Context, node EntityNode) error
    GetNode(ctx context.Context, id string) (*EntityNode, error)
    GetNodeByName(ctx context.Context, groupID, name string) (*EntityNode, error)
    DeleteNode(ctx context.Context, id string) error
    ListNodes(ctx context.Context, groupID string, opts PaginationOpts) ([]EntityNode, error)
}

type EdgeRepository interface {
    SaveEdge(ctx context.Context, edge EntityEdge) error
    GetEdge(ctx context.Context, id string) (*EntityEdge, error)
    DeleteEdge(ctx context.Context, id string) error
    InvalidateEdge(ctx context.Context, id string, invalidAt time.Time) error
    GetEdgesInTimeRange(ctx context.Context, groupID string, from, to time.Time) ([]EntityEdge, error)
}

type SearchRepository interface {
    CosineSimilaritySearch(ctx context.Context, groupID string, embedding []float32, limit int) ([]SearchResult, error)
    FulltextSearch(ctx context.Context, groupID, query string, limit int) ([]SearchResult, error)
    BFSSearch(ctx context.Context, startNodeID string, depth int) ([]SearchResult, error)
}

type BulkRepository interface {
    SaveBulk(ctx context.Context, nodes []EntityNode, edges []EntityEdge, episode EpisodicNode) error
    RollbackBulk(ctx context.Context, episodeID string) error
    DeleteByGroupID(ctx context.Context, groupID string) error
}

type IndexRepository interface {
    BuildIndices(ctx context.Context, groupID string) error
    DropIndices(ctx context.Context, groupID string) error
    ListIndices(ctx context.Context) ([]IndexDefinition, error)
}

type TransactionManager interface {
    WithTransaction(ctx context.Context, fn func(tx Transaction) error) error
}
```

### 2.2 Usecase Layer

```
internal/usecase/
├── node_ops.go          # SaveNode, GetNode, DeleteNode, ListNodes
├── edge_ops.go          # SaveEdge, GetEdge, DeleteEdge, InvalidateEdge
├── community_ops.go     # SaveCommunity, GetCommunity, DeleteCommunity
├── bulk_ops.go          # SaveBulk, RollbackBulk, DeleteByGroupID
├── search_ops.go        # CosineSimilarity, Fulltext, BFS (delegation)
├── index_ops.go         # BuildIndices, DropIndices, ListIndices
├── port/
│   ├── input.go         # Use case interfaces for each operation group
│   └── output.go        # GraphDriver (single output port wrapping all repos)
└── dto/
    ├── request.go
    └── response.go
```

### 2.3 Adapter Layer

```
internal/adapter/
├── grpc/
│   ├── handler.go       # GraphitiStoreService gRPC handlers
│   └── mapper.go        # Proto ↔ Domain bidirectional mapping
├── driver/
│   ├── neo4j/
│   │   ├── driver.go         # GraphDriver impl (composes all repos)
│   │   ├── node_repo.go      # Cypher queries for nodes
│   │   ├── edge_repo.go      # Cypher queries for edges (bi-temporal)
│   │   ├── community_repo.go # Cypher queries for communities
│   │   ├── search_repo.go    # Vector index + fulltext + BFS queries
│   │   ├── bulk_repo.go      # Bulk operations in single transaction
│   │   ├── index_repo.go     # CREATE INDEX queries
│   │   └── transaction.go    # Neo4j session/transaction wrapper
│   ├── falkordb/              # FalkorDB driver (future)
│   ├── kuzu/                  # Kuzu driver (future)
│   └── neptune/               # Neptune driver (future)
└── factory/
    └── driver_factory.go      # DRIVER_PROVIDER → GraphDriver instantiation
```

### 2.4 Infrastructure Layer

```
internal/infra/
├── config/config.go     # Driver selection + connection config
├── server/grpc.go
├── telemetry/
└── wire/wire.go         # Wire providers for driver factory
```

## 3. gRPC API

```protobuf
service GraphitiStoreService {
  // Node operations
  rpc SaveNode(SaveNodeRequest) returns (SaveNodeResponse);
  rpc GetNode(GetNodeRequest) returns (GetNodeResponse);
  rpc DeleteNode(DeleteNodeRequest) returns (DeleteNodeResponse);
  // Edge operations (bi-temporal)
  rpc SaveEdge(SaveEdgeRequest) returns (SaveEdgeResponse);
  rpc GetEdge(GetEdgeRequest) returns (GetEdgeResponse);
  rpc DeleteEdge(DeleteEdgeRequest) returns (DeleteEdgeResponse);
  rpc InvalidateEdge(InvalidateEdgeRequest) returns (InvalidateEdgeResponse);
  // Bulk operations
  rpc SaveBulk(SaveBulkRequest) returns (SaveBulkResponse);
  rpc RollbackBulk(RollbackBulkRequest) returns (RollbackBulkResponse);
  rpc DeleteByGroupID(DeleteByGroupIDRequest) returns (DeleteByGroupIDResponse);
  // Search primitives
  rpc CosineSimilaritySearch(CosineSimilarityRequest) returns (SearchResponse);
  rpc FulltextSearch(FulltextSearchRequest) returns (SearchResponse);
  rpc BFSSearch(BFSSearchRequest) returns (SearchResponse);
  // Index management
  rpc BuildIndices(BuildIndicesRequest) returns (BuildIndicesResponse);
  rpc DropIndices(DropIndicesRequest) returns (DropIndicesResponse);
}
```

## 4. Neo4j Cypher Patterns

### Cosine Similarity
```cypher
CALL db.index.vector.queryNodes('entity_name_embedding', $limit, $embedding)
YIELD node, score
WHERE node.group_id = $group_id
RETURN node, score
ORDER BY score DESC
```

### BFS Traversal
```cypher
MATCH path = (start:Entity {uuid: $start_id})-[*1..$depth]-(connected)
WHERE ALL(r IN relationships(path) WHERE r.group_id = $group_id)
RETURN connected, length(path) AS distance
ORDER BY distance ASC
```

## 5. Observability

- **Metrics**: `graphiti_store_operation_duration_seconds{operation,driver}`, `graphiti_store_neo4j_pool_active`, `graphiti_store_bulk_size`, `graphiti_store_search_results_count{method}`
- **Traces**: OTel spans per Cypher query
- **Health**: gRPC health + HTTP /healthz on :9097

## Feature Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| FEAT-STO-001 | Domain layer | ⏳ Draft | P0 |
| FEAT-STO-002 | Usecase + ports | ⏳ Draft | P0 |
| FEAT-STO-003 | Neo4j node repo | ⏳ Draft | P0 |
| FEAT-STO-004 | Neo4j edge repo | ⏳ Draft | P0 |
| FEAT-STO-005 | Neo4j search | ⏳ Draft | P0 |
| FEAT-STO-006 | Neo4j bulk + txn | ⏳ Draft | P0 |
| FEAT-STO-007 | Neo4j indexes | ⏳ Draft | P1 |
| FEAT-STO-008 | gRPC handlers | ⏳ Draft | P0 |
| FEAT-STO-009 | Infrastructure | ⏳ Draft | P0 |

---

> **Next Steps**: Implement FEAT specs from SOL-001 in dependency order.
