# Change Request: CR-GR-002 — Temporal Knowledge Graph Store (Multi-Backend)

**CR ID:** CR-GR-002  
**Component:** `services/graphiti-store` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** graphiti PRD §5.3, §6.2, SRS §3.2, specs/services/05-store-service.md  
**Maps to Python:** `graphiti_core/driver/` + `utils/maintenance/` + `namespaces/`

---

## 1. Mô tả

Xây dựng **graphiti-store** service — abstraction layer duy nhất cho tất cả graph database operations, với **bi-temporal data model** (valid_at/invalid_at/expired_at) và hỗ trợ 4 graph backends: **Neo4j**, **FalkorDB**, **Kuzu**, **Amazon Neptune**.

Đây là service quan trọng nhất, nắm giữ toàn bộ graph data và là nền tảng để các services khác vận hành.

---

## 2. Vấn đề hiện tại

`services/graph-service` hiện tại:
- ✅ Kết nối được với Neo4j.
- ❌ Không có **bi-temporal model** đầy đủ (`valid_at`, `invalid_at`, `expired_at`).
- ❌ Không có **4-type node model** (EntityNode/EpisodicNode/CommunityNode/SagaNode).
- ❌ Không có **5-type edge model** (EntityEdge/EpisodicEdge/CommunityEdge/HasEpisodeEdge/NextEpisodeEdge).
- ❌ Không có **FalkorDB/Kuzu/Neptune driver** — chỉ có Neo4j.
- ❌ Không có **InvalidateEntityEdge** (temporal invalidation) — core feature của graphiti.
- ❌ Không có **unified gRPC API** cho 50+ graph operations.
- ❌ Không có **BFS traversal** + **node distance reranker**.
- ❌ Không có **community cluster** queries.
- ❌ Không có **SaveBulk** (atomic episode+nodes+edges).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/graphiti-store/`

**Port:** `9004` (gRPC internal)

### 3.2. Core Domain Models (Bi-Temporal)

```go
// pkg/graph/node.go (shared package)

type EntityNode struct {
    UUID            string
    Name            string
    Labels          []string          // e.g. ["Person", "Developer"]
    Summary         string            // LLM-generated description
    Attributes      map[string]any    // custom key-value pairs
    NameEmbedding   []float32         // vector for name similarity search
    GroupID         string            // multi-tenant partition key
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type EpisodicNode struct {
    UUID             string
    Name             string
    Content          string            // raw episode content
    Source           EpisodeType       // text|json|message|fact_triple
    SourceDescription string
    ValidAt          time.Time         // when the event occurred
    EntityEdges      []string          // UUIDs of related EntityEdges
    EpisodeMetadata  map[string]any
    GroupID          string
    CreatedAt        time.Time
}

type CommunityNode struct {
    UUID          string
    Name          string
    Summary       string            // LLM-generated cluster summary
    NameEmbedding []float32
    GroupID       string
    CreatedAt     time.Time
}

type SagaNode struct {
    UUID              string
    Name              string
    GroupID           string
    Summary           string
    FirstEpisodeUUID  string
    LastEpisodeUUID   string
    LastSummarizedAt  *time.Time
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

// pkg/graph/edge.go

// EntityEdge — primary fact carrier with full bi-temporal model
type EntityEdge struct {
    UUID            string
    SourceNodeUUID  string
    TargetNodeUUID  string
    Name            string            // relationship label e.g. "WORKS_AT"
    Fact            string            // natural language: "Alice works at Acme Corp"
    FactEmbedding   []float32
    Episodes        []string          // source episode UUIDs (provenance)
    ValidAt         *time.Time        // when fact became true in real world
    InvalidAt       *time.Time        // when fact ceased to be true
    ExpiredAt       *time.Time        // when edge was superseded by newer info
    GroupID         string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// EpisodicEdge (MENTIONS): Episode → Entity
type EpisodicEdge struct {
    UUID       string
    SourceUUID string  // EpisodicNode
    TargetUUID string  // EntityNode
    GroupID    string
    CreatedAt  time.Time
}

// CommunityEdge (HAS_MEMBER): Community → Entity
type CommunityEdge struct {
    UUID       string
    SourceUUID string  // CommunityNode
    TargetUUID string  // EntityNode
    GroupID    string
    CreatedAt  time.Time
}

// HasEpisodeEdge (HAS_EPISODE): Saga → Episode
type HasEpisodeEdge struct {
    UUID       string
    SourceUUID string  // SagaNode
    TargetUUID string  // EpisodicNode
    GroupID    string
    CreatedAt  time.Time
}

// NextEpisodeEdge (NEXT_EPISODE): Episode → Episode (chronological order)
type NextEpisodeEdge struct {
    UUID       string
    SourceUUID string  // EpisodicNode (earlier)
    TargetUUID string  // EpisodicNode (later)
    GroupID    string
    CreatedAt  time.Time
}
```

### 3.3. Bi-Temporal Semantics

| Field | Meaning |
|---|---|
| `valid_at` | Khi fact trở nên đúng trong thế giới thực |
| `invalid_at` | Khi fact ngừng đúng (set khi bị mâu thuẫn) |
| `expired_at` | System time khi edge bị supersede bởi edge mới hơn |
| `created_at` | System time khi edge được tạo trong graph |

**Invalidation logic:**
```
New episode contradicts old fact →
  1. old_edge.invalid_at = reference_time (mark as invalid)
  2. old_edge.expired_at = now (mark as superseded)
  3. Create new_edge with updated valid_at
  4. Both edges PERSIST for point-in-time queries
```

### 3.4. GraphDriver Interface

```go
// internal/usecase/port/output.go

type GraphDriver interface {
    Close(ctx) error
    Ping(ctx) error
    Provider() GraphProvider  // Neo4j | FalkorDB | Kuzu | Neptune
    ExecuteQuery(ctx, query, params) ([]Record, error)
    BeginTransaction(ctx) (Transaction, error)
    EntityNodeOps()     EntityNodeRepository
    EpisodeNodeOps()    EpisodeNodeRepository
    CommunityNodeOps()  CommunityNodeRepository
    SagaNodeOps()       SagaNodeRepository
    EntityEdgeOps()     EntityEdgeRepository
    EpisodicEdgeOps()   EpisodicEdgeRepository
    CommunityEdgeOps()  CommunityEdgeRepository
    HasEpisodeEdgeOps() HasEpisodeEdgeRepository
    NextEpisodeEdgeOps() NextEpisodeEdgeRepository
    SearchOps()         SearchRepository
    MaintenanceOps()    MaintenanceRepository
}

type Transaction interface {
    Run(ctx, query, params) ([]Record, error)
    Commit(ctx) error
    Rollback(ctx) error
}
```

### 3.5. Repository Interfaces

```go
type EntityNodeRepository interface {
    Save(ctx, node, tx) error
    SaveBulk(ctx, nodes, tx, batchSize) error
    GetByUUID(ctx, uuid) (*EntityNode, error)
    GetByUUIDs(ctx, uuids) ([]*EntityNode, error)
    GetByGroupIDs(ctx, groupIDs, limit, cursor) ([]*EntityNode, error)
    Delete(ctx, uuid, tx) error
    DeleteByGroupID(ctx, groupID, tx, batchSize) error
    LoadEmbeddings(ctx, node) error
    LoadEmbeddingsBulk(ctx, nodes, batchSize) error
}

type EntityEdgeRepository interface {
    Save(ctx, edge, tx) error
    SaveBulk(ctx, edges, tx, batchSize) error
    GetByUUID(ctx, uuid) (*EntityEdge, error)
    GetBetweenNodes(ctx, srcUUID, tgtUUID) ([]*EntityEdge, error)
    GetByNodeUUID(ctx, nodeUUID) ([]*EntityEdge, error)
    Invalidate(ctx, uuid, invalidAt, tx) error  // ← core temporal operation
    Delete(ctx, uuid, tx) error
    LoadEmbeddings(ctx, edge) error
}

type EpisodeNodeRepository interface {
    Save(ctx, node, tx) error
    GetByUUID(ctx, uuid) (*EpisodicNode, error)
    GetByEntityNodeUUID(ctx, entityNodeUUID) ([]*EpisodicNode, error)
    RetrieveEpisodes(ctx, referenceTime, lastN, groupIDs, source, saga) ([]*EpisodicNode, error)
    Delete(ctx, uuid, tx) error
}

type SearchRepository interface {
    NodeFulltextSearch(ctx, query, groupIDs, limit, filters) ([]*EntityNode, error)
    NodeSimilaritySearch(ctx, vector, groupIDs, limit, minScore) ([]*EntityNode, error)
    NodeBFSSearch(ctx, originUUIDs, maxDepth, groupIDs, limit) ([]*EntityNode, error)
    EdgeFulltextSearch(ctx, query, groupIDs, limit, filters) ([]*EntityEdge, error)
    EdgeSimilaritySearch(ctx, vector, srcUUID, tgtUUID, groupIDs, limit, minScore) ([]*EntityEdge, error)
    EdgeBFSSearch(ctx, originUUIDs, maxDepth, groupIDs, limit) ([]*EntityEdge, error)
    EpisodeFulltextSearch(ctx, query, groupIDs, limit) ([]*EpisodicNode, error)
    CommunityFulltextSearch(ctx, query, groupIDs, limit) ([]*CommunityNode, error)
    CommunitySimilaritySearch(ctx, vector, groupIDs, limit, minScore) ([]*CommunityNode, error)
    NodeDistanceReranker(ctx, nodeUUIDs, centerUUID) (map[string]float64, error)
    EpisodeMentionsReranker(ctx, nodeUUIDs) (map[string]float64, error)
}

type MaintenanceRepository interface {
    ClearData(ctx, groupIDs) error
    BuildIndicesAndConstraints(ctx, deleteExisting) error
    DeleteAllIndexes(ctx) error
    GetCommunityClusters(ctx, groupIDs) ([][]string, error)
    RemoveCommunities(ctx) error
    DetermineEntityCommunity(ctx, entityUUID) (string, error)
    GetMentionedNodes(ctx, episodeUUIDs) ([]*EntityNode, error)
}
```

### 3.6. Atomic Bulk Save

```go
// SaveBulk: single gRPC call for atomic episode+nodes+edges
// type SaveBulkRequest:
//   - episode: EpisodicNode
//   - entity_nodes: []EntityNode
//   - entity_edges: []EntityEdge
//   - episodic_edges: []EpisodicEdge
//   - saga (optional): SagaNode
//   - has_episode_edges: []HasEpisodeEdge
//   - next_episode_edges: []NextEpisodeEdge
//   - group_id: string

// Neo4j: wrapped in real transaction
// FalkorDB/Kuzu/Neptune: best-effort (no-op transaction wrapper)
```

### 3.7. Driver Implementations

| Driver | Notes |
|---|---|
| **Neo4j** (`neo4j-go-driver`) | Primary. Real transactions. Lucene fulltext. Native vector index. |
| **FalkorDB** (`redisgraph-go`) | In-memory. `FT_` prefix fulltext. Built-in vector. Separate graph per group_id. |
| **Kuzu** (`kuzu-go`) | Embedded file-based. `CONTAINS` fulltext. Separate database per group_id. Intermediate `RelatesToNode_` for edges. |
| **Neptune** (`HTTP client`) | AWS managed. OpenSearch for fulltext. CSV serialized embeddings. |

### 3.8. Index Management (Neo4j migrations)

```cypher
-- 001_initial_schema.cypher
CREATE CONSTRAINT entity_node_uuid IF NOT EXISTS ON (n:Entity) ASSERT n.uuid IS UNIQUE;
CREATE CONSTRAINT episodic_node_uuid IF NOT EXISTS ON (n:Episodic) ASSERT n.uuid IS UNIQUE;
CREATE CONSTRAINT community_node_uuid IF NOT EXISTS ON (n:Community) ASSERT n.uuid IS UNIQUE;

-- 002_vector_indices.cypher
CREATE VECTOR INDEX entity_name_embedding IF NOT EXISTS
    FOR (n:Entity) ON (n.name_embedding) OPTIONS {indexConfig: {`vector.dimensions`: 1536, `vector.similarity_function`: 'cosine'}};
CREATE VECTOR INDEX entity_edge_fact_embedding IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.fact_embedding) OPTIONS {...};

-- 003_fulltext_indices.cypher
CREATE FULLTEXT INDEX entity_fulltext IF NOT EXISTS FOR (n:Entity) ON EACH [n.name, n.summary];
CREATE FULLTEXT INDEX episode_fulltext IF NOT EXISTS FOR (n:Episodic) ON EACH [n.content, n.source_description];
```

### 3.9. Multi-Tenancy Enforcement

```go
// group_id enforcement at Store level — last line of defense
// ALL queries MUST include group_id filter

// Neo4j/Neptune: WHERE n.group_id IN $group_ids
// FalkorDB: separate graph per group_id → db.SelectGraph(groupID)
// Kuzu: separate database per group_id
```

### 3.10. gRPC API Surface (50+ RPCs)

```protobuf
service StoreService {
    // Entity Nodes (8 RPCs)
    rpc SaveEntityNode, SaveEntityNodeBulk, GetEntityNode, GetEntityNodeBulk,
        GetEntityNodesByGroupIDs, DeleteEntityNode, DeleteEntityNodesByGroupID,
        LoadEntityNodeEmbeddings

    // Episode Nodes (5 RPCs)
    rpc SaveEpisodeNode, SaveEpisodeNodeBulk, GetEpisodeNode,
        GetEpisodesByEntityNode, RetrieveEpisodes, DeleteEpisodeNode

    // Community Nodes (3 RPCs)
    rpc SaveCommunityNode, GetCommunityNode, DeleteCommunityNode

    // Saga Nodes (3 RPCs)
    rpc SaveSagaNode, GetSagaNode, GetSagasByGroupIDs

    // Entity Edges (7 RPCs)
    rpc SaveEntityEdge, SaveEntityEdgeBulk, GetEntityEdge, GetEdgesBetweenNodes,
        GetEdgesByNodeUUID, InvalidateEntityEdge, DeleteEntityEdge, LoadEntityEdgeEmbeddings

    // Episodic/Community/Has/Next Edges (6 RPCs)
    rpc SaveEpisodicEdge, SaveEpisodicEdgeBulk, SaveCommunityEdge, DeleteCommunityEdge,
        SaveHasEpisodeEdge, SaveNextEpisodeEdge

    // Bulk (1 RPC)
    rpc SaveBulk

    // Search (10 RPCs)
    rpc NodeFulltextSearch, NodeSimilaritySearch, NodeBFSSearch,
        EdgeFulltextSearch, EdgeSimilaritySearch, EdgeBFSSearch,
        EpisodeFulltextSearch, CommunityFulltextSearch, CommunitySimilaritySearch,
        NodeDistanceReranker, EpisodeMentionsReranker

    // Maintenance (7 RPCs)
    rpc ClearData, BuildIndicesAndConstraints, DeleteAllIndexes,
        GetCommunityClusters, RemoveCommunities, DetermineEntityCommunity,
        GetMentionedNodes

    // Transactions (3 RPCs)
    rpc BeginTransaction, CommitTransaction, RollbackTransaction
}
```

---

## 4. Configuration

```yaml
server:
  grpc_port: 9004

driver:
  provider: "neo4j"  # neo4j | falkordb | kuzu | neptune
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
    endpoint: "https://neptune:8182"
    region: "us-east-1"
    opensearch_endpoint: "https://opensearch"

indices:
  entity_index_name: "entities"
  episode_index_name: "episodes"
  community_index_name: "communities"
  entity_edge_index_name: "entity_edges"

bulk:
  default_batch_size: 100
  max_batch_size: 1000
```

---

## 5. Acceptance Criteria

- [ ] `SaveEntityNode` → entity readable via `GetEntityNode` với đúng UUID, name, summary.
- [ ] `SaveEntityEdge` với fact "Alice works at Acme" → `GetEntityEdge` trả về `fact: "Alice works at Acme"`.
- [ ] `InvalidateEntityEdge` với `invalid_at = now` → edge có `invalid_at` set; subsequent `EdgeSimilaritySearch` không trả về edge đó trong temporal filter `valid_at < invalid_at`.
- [ ] `SaveBulk` (episode + 3 entities + 2 edges) → tất cả entities và edges tồn tại trong graph (atomic).
- [ ] Neo4j transaction rollback: nếu một bước trong `SaveBulk` fail → không có partial data.
- [ ] `NodeSimilaritySearch` với embedding vector → trả về nodes sorted by cosine similarity.
- [ ] `NodeBFSSearch` với `max_depth=2` → trả về nodes trong 2 hops từ origin nodes.
- [ ] `GetCommunityClusters` → trả về cluster groups dưới dạng `[][]string` (lists of node UUIDs).
- [ ] Switch driver từ Neo4j → FalkorDB (chỉ thay config) → tất cả operations hoạt động đúng.
- [ ] `ClearData(group_id="project-alpha")` → chỉ xóa data của group-alpha, không ảnh hưởng group khác.
- [ ] `BuildIndicesAndConstraints` → Neo4j vector index và fulltext index được tạo thành công.
