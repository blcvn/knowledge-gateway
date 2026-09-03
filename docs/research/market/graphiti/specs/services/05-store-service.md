# graphiti-store — Graph Storage Service

**Version:** 2.0 | **Date:** 2026-05-09  
**Origin:** Python L2 (Data Access Layer — Operations + Namespaces) + L1 (Storage & Driver Layer)  
**Architecture:** Clean Architecture | **Protocol:** gRPC

---

## 1. Service Overview

Graph Storage Service là abstraction layer duy nhất cho tất cả graph database operations. Nó encapsulates driver-specific implementations (Neo4j, FalkorDB, Kuzu, Neptune), cung cấp unified CRUD, search, và transaction support qua single gRPC interface.

### Responsibilities

| Concern | Description |
|---------|-------------|
| **Node CRUD** | Save, get, delete, bulk operations cho Entity/Episode/Community/Saga nodes |
| **Edge CRUD** | Save, get, delete cho Entity/Episodic/Community/HasEpisode/NextEpisode edges |
| **Search Queries** | Fulltext, similarity, BFS graph traversal queries |
| **Transaction Management** | Atomic multi-operation transactions (real for Neo4j, no-op wrapper for others) |
| **Index Management** | Create/delete vector indices, fulltext indices, constraints |
| **Graph Maintenance** | Clear data, community clusters, adjacency queries |
| **Driver Abstraction** | Hide backend-specific query syntax, embedding storage format, edge models |
| **Multi-tenancy** | group_id partition enforcement across all operations |
| **Embedding Persistence** | Store and retrieve vector embeddings per node/edge |

---

## 2. Clean Architecture Layout

```
services/graphiti-store/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/                         # Layer 1: Entities
│   │   ├── node.go                     #   EntityNode, EpisodicNode, CommunityNode, SagaNode
│   │   ├── edge.go                     #   EntityEdge, EpisodicEdge, CommunityEdge, HasEpisodeEdge, NextEpisodeEdge
│   │   ├── temporal.go                 #   Bi-temporal model (valid_at, invalid_at, expired_at, created_at)
│   │   ├── group.go                    #   GroupID, partition semantics
│   │   ├── embedding.go                #   EmbeddingVector value object
│   │   ├── search.go                   #   SearchFilter, FulltextQuery
│   │   ├── transaction.go              #   Transaction domain model
│   │   └── errors.go
│   ├── usecase/                        # Layer 2: Use Cases
│   │   ├── entity_node.go              #   EntityNode CRUD use cases
│   │   ├── episode_node.go             #   EpisodicNode CRUD use cases
│   │   ├── community_node.go           #   CommunityNode CRUD use cases
│   │   ├── saga_node.go                #   SagaNode CRUD use cases
│   │   ├── entity_edge.go              #   EntityEdge CRUD + conflict
│   │   ├── episodic_edge.go            #   EpisodicEdge CRUD
│   │   ├── community_edge.go           #   CommunityEdge CRUD
│   │   ├── has_episode_edge.go         #   HasEpisodeEdge CRUD
│   │   ├── next_episode_edge.go        #   NextEpisodeEdge CRUD
│   │   ├── search_query.go             #   Fulltext, similarity, BFS queries
│   │   ├── save_bulk.go                #   Bulk save operations
│   │   ├── graph_maintenance.go        #   Index, clear, cluster operations
│   │   ├── port/
│   │   │   ├── input.go               #   StoreUseCase interfaces
│   │   │   └── output.go             #   Repository interfaces (per driver)
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── grpc/
│   │   │   ├── handler.go             #   gRPC service implementation
│   │   │   ├── node_handler.go        #   Node-specific handlers
│   │   │   ├── edge_handler.go        #   Edge-specific handlers
│   │   │   ├── search_handler.go      #   Search query handlers
│   │   │   ├── maintenance_handler.go #   Maintenance handlers
│   │   │   └── mapper.go             #   Proto ↔ Domain mapping
│   │   └── repository/               #   Driver implementations
│   │       ├── driver.go              #     GraphDriver interface
│   │       ├── neo4j/                 #     Neo4j driver
│   │       │   ├── driver.go          #       Neo4jDriver implementation
│   │       │   ├── entity_node.go     #       EntityNode operations
│   │       │   ├── episode_node.go    #       EpisodeNode operations
│   │       │   ├── community_node.go  #       CommunityNode operations
│   │       │   ├── saga_node.go       #       SagaNode operations
│   │       │   ├── entity_edge.go     #       EntityEdge operations
│   │       │   ├── episodic_edge.go   #       EpisodicEdge operations
│   │       │   ├── community_edge.go  #       CommunityEdge operations
│   │       │   ├── has_episode_edge.go
│   │       │   ├── next_episode_edge.go
│   │       │   ├── search.go          #       Search operations
│   │       │   ├── maintenance.go     #       Graph maintenance
│   │       │   ├── transaction.go     #       Neo4j transaction wrapper
│   │       │   ├── query_builder.go   #       Cypher query builder
│   │       │   └── record_parser.go   #       Record → domain mapping
│   │       ├── falkordb/              #     FalkorDB driver
│   │       │   ├── driver.go
│   │       │   ├── entity_node.go
│   │       │   ├── ...                #       Same structure as neo4j/
│   │       │   └── fulltext.go        #       Custom fulltext syntax
│   │       ├── kuzu/                  #     Kuzu driver
│   │       │   ├── driver.go
│   │       │   ├── ...
│   │       │   └── relates_to_node.go #       Intermediate node for edges
│   │       └── neptune/               #     Neptune driver
│   │           ├── driver.go
│   │           ├── ...
│   │           ├── opensearch.go      #       OpenSearch fulltext adapter
│   │           └── csv_embedding.go   #       CSV serialized embeddings
│   └── infra/
│       ├── config/
│       │   └── config.go
│       ├── server/
│       │   └── grpc.go
│       ├── telemetry/
│       └── wire/
├── api/
│   └── proto/
│       └── store/v1/
│           └── store.proto
├── migrations/
│   ├── neo4j/
│   │   ├── 001_initial_schema.cypher
│   │   ├── 002_vector_indices.cypher
│   │   └── 003_fulltext_indices.cypher
│   ├── falkordb/
│   └── kuzu/
├── Dockerfile
└── Makefile
```

---

## 3. gRPC API (Protobuf)

```protobuf
syntax = "proto3";
package graphiti.store.v1;

import "google/protobuf/timestamp.proto";
import "common/pagination.proto";

service StoreService {
  // === Entity Node Operations ===
  rpc SaveEntityNode(SaveEntityNodeRequest) returns (SaveEntityNodeResponse);
  rpc SaveEntityNodeBulk(SaveEntityNodeBulkRequest) returns (SaveEntityNodeBulkResponse);
  rpc GetEntityNode(GetEntityNodeRequest) returns (GetEntityNodeResponse);
  rpc GetEntityNodeBulk(GetEntityNodeBulkRequest) returns (GetEntityNodeBulkResponse);
  rpc GetEntityNodesByGroupIDs(GetByGroupIDsRequest) returns (GetEntityNodesResponse);
  rpc DeleteEntityNode(DeleteEntityNodeRequest) returns (DeleteResponse);
  rpc DeleteEntityNodesByGroupID(DeleteByGroupIDRequest) returns (DeleteResponse);
  rpc LoadEntityNodeEmbeddings(LoadEmbeddingsRequest) returns (LoadEmbeddingsResponse);
  
  // === Episode Node Operations ===
  rpc SaveEpisodeNode(SaveEpisodeNodeRequest) returns (SaveEpisodeNodeResponse);
  rpc SaveEpisodeNodeBulk(SaveEpisodeNodeBulkRequest) returns (SaveEpisodeNodeBulkResponse);
  rpc GetEpisodeNode(GetEpisodeNodeRequest) returns (GetEpisodeNodeResponse);
  rpc GetEpisodesByEntityNode(GetEpisodesByEntityNodeRequest) returns (GetEpisodesResponse);
  rpc RetrieveEpisodes(RetrieveEpisodesRequest) returns (GetEpisodesResponse);
  rpc DeleteEpisodeNode(DeleteEpisodeNodeRequest) returns (DeleteResponse);
  
  // === Community Node Operations ===
  rpc SaveCommunityNode(SaveCommunityNodeRequest) returns (SaveCommunityNodeResponse);
  rpc GetCommunityNode(GetCommunityNodeRequest) returns (GetCommunityNodeResponse);
  rpc DeleteCommunityNode(DeleteCommunityNodeRequest) returns (DeleteResponse);
  
  // === Saga Node Operations ===
  rpc SaveSagaNode(SaveSagaNodeRequest) returns (SaveSagaNodeResponse);
  rpc GetSagaNode(GetSagaNodeRequest) returns (GetSagaNodeResponse);
  rpc GetSagasByGroupIDs(GetByGroupIDsRequest) returns (GetSagasResponse);
  
  // === Entity Edge Operations ===
  rpc SaveEntityEdge(SaveEntityEdgeRequest) returns (SaveEntityEdgeResponse);
  rpc SaveEntityEdgeBulk(SaveEntityEdgeBulkRequest) returns (SaveEntityEdgeBulkResponse);
  rpc GetEntityEdge(GetEntityEdgeRequest) returns (GetEntityEdgeResponse);
  rpc GetEdgesBetweenNodes(GetEdgesBetweenNodesRequest) returns (GetEntityEdgesResponse);
  rpc GetEdgesByNodeUUID(GetEdgesByNodeUUIDRequest) returns (GetEntityEdgesResponse);
  rpc InvalidateEntityEdge(InvalidateEntityEdgeRequest) returns (InvalidateEntityEdgeResponse);
  rpc DeleteEntityEdge(DeleteEntityEdgeRequest) returns (DeleteResponse);
  rpc LoadEntityEdgeEmbeddings(LoadEmbeddingsRequest) returns (LoadEmbeddingsResponse);
  
  // === Episodic Edge Operations ===
  rpc SaveEpisodicEdge(SaveEpisodicEdgeRequest) returns (SaveEpisodicEdgeResponse);
  rpc SaveEpisodicEdgeBulk(SaveEpisodicEdgeBulkRequest) returns (SaveEpisodicEdgeBulkResponse);
  
  // === Community Edge Operations ===
  rpc SaveCommunityEdge(SaveCommunityEdgeRequest) returns (SaveCommunityEdgeResponse);
  rpc DeleteCommunityEdge(DeleteCommunityEdgeRequest) returns (DeleteResponse);
  
  // === HasEpisode/NextEpisode Edge Operations ===
  rpc SaveHasEpisodeEdge(SaveHasEpisodeEdgeRequest) returns (SaveHasEpisodeEdgeResponse);
  rpc SaveNextEpisodeEdge(SaveNextEpisodeEdgeRequest) returns (SaveNextEpisodeEdgeResponse);
  
  // === Bulk Operations ===
  rpc SaveBulk(SaveBulkRequest) returns (SaveBulkResponse);
  
  // === Search Operations ===
  rpc NodeFulltextSearch(FulltextSearchRequest) returns (NodeSearchResponse);
  rpc NodeSimilaritySearch(SimilaritySearchRequest) returns (NodeSearchResponse);
  rpc NodeBFSSearch(BFSSearchRequest) returns (NodeSearchResponse);
  rpc EdgeFulltextSearch(FulltextSearchRequest) returns (EdgeSearchResponse);
  rpc EdgeSimilaritySearch(EdgeSimilaritySearchRequest) returns (EdgeSearchResponse);
  rpc EdgeBFSSearch(BFSSearchRequest) returns (EdgeSearchResponse);
  rpc EpisodeFulltextSearch(FulltextSearchRequest) returns (EpisodeSearchResponse);
  rpc CommunityFulltextSearch(FulltextSearchRequest) returns (CommunitySearchResponse);
  rpc CommunitySimilaritySearch(SimilaritySearchRequest) returns (CommunitySearchResponse);
  rpc NodeDistanceReranker(NodeDistanceRerankerRequest) returns (NodeDistanceRerankerResponse);
  rpc EpisodeMentionsReranker(EpisodeMentionsRerankerRequest) returns (EpisodeMentionsRerankerResponse);
  
  // === Graph Maintenance ===
  rpc ClearData(ClearDataRequest) returns (ClearDataResponse);
  rpc BuildIndicesAndConstraints(BuildIndicesRequest) returns (BuildIndicesResponse);
  rpc DeleteAllIndexes(DeleteAllIndexesRequest) returns (DeleteAllIndexesResponse);
  rpc GetCommunityClusters(GetCommunityClustersRequest) returns (GetCommunityClustersResponse);
  rpc RemoveCommunities(RemoveCommunitiesRequest) returns (RemoveCommunitiesResponse);
  rpc DetermineEntityCommunity(DetermineEntityCommunityRequest) returns (DetermineEntityCommunityResponse);
  rpc GetMentionedNodes(GetMentionedNodesRequest) returns (GetMentionedNodesResponse);
  
  // === Transaction Operations ===
  rpc BeginTransaction(BeginTransactionRequest) returns (BeginTransactionResponse);
  rpc CommitTransaction(CommitTransactionRequest) returns (CommitTransactionResponse);
  rpc RollbackTransaction(RollbackTransactionRequest) returns (RollbackTransactionResponse);
}

// --- Common Messages ---

message SaveBulkRequest {
  // Atomic bulk save — episode + nodes + edges in single transaction
  EpisodeNodeProto episode = 1;
  repeated EntityNodeProto entity_nodes = 2;
  repeated EntityEdgeProto entity_edges = 3;
  repeated EpisodicEdgeProto episodic_edges = 4;
  optional SagaNodeProto saga = 5;
  repeated HasEpisodeEdgeProto has_episode_edges = 6;
  repeated NextEpisodeEdgeProto next_episode_edges = 7;
  string group_id = 8;
  optional string transaction_id = 9;
}

message FulltextSearchRequest {
  string query = 1;
  repeated string group_ids = 2;
  int32 limit = 3;
  SearchFilterProto filters = 4;
}

message SimilaritySearchRequest {
  repeated float search_vector = 1;
  repeated string group_ids = 2;
  int32 limit = 3;
  double min_score = 4;
  SearchFilterProto filters = 5;
}

message BFSSearchRequest {
  repeated string origin_uuids = 1;
  int32 max_depth = 2;
  repeated string group_ids = 3;
  int32 limit = 4;
  SearchFilterProto filters = 5;
}

message InvalidateEntityEdgeRequest {
  string edge_uuid = 1;
  google.protobuf.Timestamp invalid_at = 2;
}
```

---

## 4. Driver Interface (Output Port)

```go
// internal/usecase/port/output.go

// GraphDriver is the primary output port — all DB operations go through this
type GraphDriver interface {
    // Connection lifecycle
    Close(ctx context.Context) error
    Ping(ctx context.Context) error
    
    // Provider info
    Provider() GraphProvider
    
    // Query execution
    ExecuteQuery(ctx context.Context, query string, params map[string]interface{}) ([]Record, error)
    
    // Transaction support
    BeginTransaction(ctx context.Context) (Transaction, error)
    
    // Node operations
    EntityNodeOps() EntityNodeRepository
    EpisodeNodeOps() EpisodeNodeRepository
    CommunityNodeOps() CommunityNodeRepository
    SagaNodeOps() SagaNodeRepository
    
    // Edge operations
    EntityEdgeOps() EntityEdgeRepository
    EpisodicEdgeOps() EpisodicEdgeRepository
    CommunityEdgeOps() CommunityEdgeRepository
    HasEpisodeEdgeOps() HasEpisodeEdgeRepository
    NextEpisodeEdgeOps() NextEpisodeEdgeRepository
    
    // Search operations
    SearchOps() SearchRepository
    
    // Maintenance operations
    MaintenanceOps() MaintenanceRepository
}

// Transaction interface — real for Neo4j, no-op for others
type Transaction interface {
    Run(ctx context.Context, query string, params map[string]interface{}) ([]Record, error)
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}

type GraphProvider int
const (
    ProviderNeo4j GraphProvider = iota
    ProviderFalkorDB
    ProviderKuzu
    ProviderNeptune
)
```

---

## 5. Repository Interfaces (per object type)

```go
// internal/usecase/port/output.go

type EntityNodeRepository interface {
    Save(ctx context.Context, node *domain.EntityNode, tx Transaction) error
    SaveBulk(ctx context.Context, nodes []*domain.EntityNode, tx Transaction, batchSize int) error
    Delete(ctx context.Context, uuid string, tx Transaction) error
    DeleteByGroupID(ctx context.Context, groupID string, tx Transaction, batchSize int) error
    DeleteByUUIDs(ctx context.Context, uuids []string, tx Transaction, batchSize int) error
    GetByUUID(ctx context.Context, uuid string) (*domain.EntityNode, error)
    GetByUUIDs(ctx context.Context, uuids []string) ([]*domain.EntityNode, error)
    GetByGroupIDs(ctx context.Context, groupIDs []string, limit int, cursor string) ([]*domain.EntityNode, error)
    LoadEmbeddings(ctx context.Context, node *domain.EntityNode) error
    LoadEmbeddingsBulk(ctx context.Context, nodes []*domain.EntityNode, batchSize int) error
}

type EpisodeNodeRepository interface {
    Save(ctx context.Context, node *domain.EpisodicNode, tx Transaction) error
    SaveBulk(ctx context.Context, nodes []*domain.EpisodicNode, tx Transaction, batchSize int) error
    Delete(ctx context.Context, uuid string, tx Transaction) error
    GetByUUID(ctx context.Context, uuid string) (*domain.EpisodicNode, error)
    GetByEntityNodeUUID(ctx context.Context, entityNodeUUID string) ([]*domain.EpisodicNode, error)
    RetrieveEpisodes(ctx context.Context, referenceTime time.Time, lastN int, groupIDs []string, source string, saga string) ([]*domain.EpisodicNode, error)
}

type EntityEdgeRepository interface {
    Save(ctx context.Context, edge *domain.EntityEdge, tx Transaction) error
    SaveBulk(ctx context.Context, edges []*domain.EntityEdge, tx Transaction, batchSize int) error
    Delete(ctx context.Context, uuid string, tx Transaction) error
    GetByUUID(ctx context.Context, uuid string) (*domain.EntityEdge, error)
    GetBetweenNodes(ctx context.Context, sourceUUID, targetUUID string) ([]*domain.EntityEdge, error)
    GetByNodeUUID(ctx context.Context, nodeUUID string) ([]*domain.EntityEdge, error)
    Invalidate(ctx context.Context, uuid string, invalidAt time.Time, tx Transaction) error
    LoadEmbeddings(ctx context.Context, edge *domain.EntityEdge) error
}

type SearchRepository interface {
    // Node search
    NodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, filters *domain.SearchFilter) ([]*domain.EntityNode, error)
    NodeSimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*domain.EntityNode, error)
    NodeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*domain.EntityNode, error)
    
    // Edge search
    EdgeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, filters *domain.SearchFilter) ([]*domain.EntityEdge, error)
    EdgeSimilaritySearch(ctx context.Context, vector []float32, srcUUID, tgtUUID *string, groupIDs []string, limit int, minScore float64) ([]*domain.EntityEdge, error)
    EdgeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*domain.EntityEdge, error)
    
    // Episode search
    EpisodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*domain.EpisodicNode, error)
    
    // Community search
    CommunityFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*domain.CommunityNode, error)
    CommunitySimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*domain.CommunityNode, error)
    
    // Reranker queries
    NodeDistanceReranker(ctx context.Context, nodeUUIDs []string, centerUUID string) (map[string]float64, error)
    EpisodeMentionsReranker(ctx context.Context, nodeUUIDs []string) (map[string]float64, error)
    
    // Fulltext query builder (provider-specific Lucene syntax)
    BuildFulltextQuery(query string, groupIDs []string, maxLen int) string
}

type MaintenanceRepository interface {
    ClearData(ctx context.Context, groupIDs []string) error
    BuildIndicesAndConstraints(ctx context.Context, deleteExisting bool) error
    DeleteAllIndexes(ctx context.Context) error
    GetCommunityClusters(ctx context.Context, groupIDs []string) ([][]string, error)
    RemoveCommunities(ctx context.Context) error
    DetermineEntityCommunity(ctx context.Context, entityUUID string) (string, error)
    GetMentionedNodes(ctx context.Context, episodeUUIDs []string) ([]*domain.EntityNode, error)
    GetCommunitiesByNodes(ctx context.Context, nodeUUIDs []string) ([]*domain.CommunityNode, error)
}
```

---

## 6. Backend Differences (Implementation Details)

| Aspect | Neo4j | FalkorDB | Kuzu | Neptune |
|--------|-------|----------|------|---------|
| **Driver** | `neo4j-go-driver` | `redisgraph-go` | `kuzu-go` | `HTTP client` |
| **Transactions** | Real (commit/rollback) | No-op wrapper | No-op wrapper | No-op wrapper |
| **Fulltext Syntax** | Lucene native | Custom prefix `FT_` | Cypher `CONTAINS` | OpenSearch query |
| **Vector Index** | Native `db.index.vector` | Built-in | Property scan | OpenSearch kNN |
| **Edge Model** | Direct `RELATES_TO` | Direct relationship | Intermediate `RelatesToNode_` | Direct relationship |
| **Multi-tenant** | `group_id` property | Separate graph | Separate database | `group_id` property |
| **Embedding Storage** | `[]float64` property | `[]float64` property | `[]float64` property | CSV string |
| **Connection** | Bolt (7687) | Redis (6379) | Embedded | HTTP/HTTPS |
| **Record Parsing** | `neo4j.Record` | `redisgraph.Result` | `kuzu.QueryResult` | `JSON response` |

---

## 7. Configuration

```yaml
# config/store.yaml
server:
  grpc_port: 9004

driver:
  provider: "neo4j"                    # neo4j | falkordb | kuzu | neptune
  
  neo4j:
    uri: "bolt://neo4j:7687"
    user: "neo4j"
    password: "${NEO4J_PASSWORD}"
    database: "neo4j"
    max_connection_pool_size: 50
    connection_acquisition_timeout: 30s
    max_transaction_retry_time: 30s
  
  falkordb:
    host: "falkordb"
    port: 6379
    default_graph: "graphiti"
  
  kuzu:
    database_path: "/data/kuzu"
  
  neptune:
    endpoint: "https://your-neptune-endpoint:8182"
    region: "us-east-1"
    opensearch_endpoint: "https://your-opensearch-endpoint"

indices:
  entity_index_name: "entities"
  episode_index_name: "episodes"
  community_index_name: "communities"
  entity_edge_index_name: "entity_edges"

bulk:
  default_batch_size: 100
  max_batch_size: 1000

telemetry:
  otel_endpoint: "otel-collector:4317"
  service_name: "graphiti-store"
```

---

## 8. Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `store_queries_total` | Counter | operation, object_type, status | Total queries |
| `store_query_duration_seconds` | Histogram | operation, object_type | Query latency |
| `store_transactions_total` | Counter | status | Transaction count |
| `store_connection_pool_size` | Gauge | — | Active connections |
| `store_bulk_save_size` | Histogram | object_type | Bulk batch sizes |
| `store_search_results_count` | Histogram | search_type | Results per search |

---

## 9. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Single Store service for all drivers** | Driver is a compile-time choice, not runtime multi-backend; simplifies deployment |
| **Repository per object type** | Maps 1:1 to Python Operations ABCs; easy to implement and test |
| **Transaction as explicit interface** | Honest about guarantees — real for Neo4j, no-op for others |
| **Embedding persistence in Store** | Embeddings are data, not AI logic; Store owns all data persistence |
| **group_id enforcement at Store level** | Last line of defense for multi-tenant isolation |
| **No caching in Store** | Search Service caches results; Store is the source of truth |
| **Bulk save as single RPC** | Atomic episode+nodes+edges save reduces network roundtrips |
