# Solution: SOL-002 — Temporal Knowledge Graph Store (Multi-Backend)

**CR ID:** CR-GR-002  
**Solution ID:** SOL-002  
**Priority:** Critical (Wave 1)  
**Architecture:** REBUILD `services/graphiti-store/` — Foundation cho tất cả graphiti services

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §2.2`:
- `graphiti-store` đã có trong monolith (service #7 trong nhóm Graphiti).
- **Neo4j** đã configured và dùng cho cả cognee + graphiti.
- **PostgreSQL + pgvector** đã có — nhưng graphiti nên dùng Neo4j làm primary.
- `InProcessRegistry` bufconn — giao tiếp zero-network.

**Vấn đề cốt lõi:** Service hiện tại thiếu:
1. Bi-temporal model đầy đủ (valid_at/invalid_at/expired_at trên EntityEdge).
2. 4-type node, 5-type edge model.
3. `InvalidateEntityEdge` operation — core của temporal graph.
4. `SaveBulk` atomic (episode+nodes+edges trong 1 Neo4j transaction).
5. BFS traversal, community cluster queries.
6. FalkorDB/Kuzu/Neptune drivers (ngoài Neo4j).

---

## 2. GraphDriver Interface — `internal/usecase/port/output.go`

```go
// services/graphiti-store/internal/usecase/port/output.go

package port

import (
    "context"
    "github.com/vnp-memory/pkg/graph"
)

// GraphDriver — unified interface cho tất cả graph backends
type GraphDriver interface {
    Close(ctx context.Context) error
    Ping(ctx context.Context) error
    Provider() GraphProvider

    // Direct Cypher/Query execution
    ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]Record, error)
    BeginTransaction(ctx context.Context) (Transaction, error)

    // Repository accessors
    EntityNodes()      EntityNodeRepository
    EpisodeNodes()     EpisodeNodeRepository
    CommunityNodes()   CommunityNodeRepository
    SagaNodes()        SagaNodeRepository
    EntityEdges()      EntityEdgeRepository
    EpisodicEdges()    EpisodicEdgeRepository
    CommunityEdges()   CommunityEdgeRepository
    HasEpisodeEdges()  HasEpisodeEdgeRepository
    NextEpisodeEdges() NextEpisodeEdgeRepository
    Search()           SearchRepository
    Maintenance()      MaintenanceRepository
    Bulk()             BulkRepository
}

type GraphProvider string
const (
    ProviderNeo4j    GraphProvider = "neo4j"
    ProviderFalkorDB GraphProvider = "falkordb"
    ProviderKuzu     GraphProvider = "kuzu"
    ProviderNeptune  GraphProvider = "neptune"
)

type Transaction interface {
    Run(ctx context.Context, query string, params map[string]any) ([]Record, error)
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}

type Record struct {
    Keys   []string
    Values []any
}
```

---

## 3. Repository Interfaces

```go
// services/graphiti-store/internal/usecase/port/repositories.go

type EntityNodeRepository interface {
    Save(ctx context.Context, node graph.EntityNode, tx Transaction) error
    SaveBulk(ctx context.Context, nodes []graph.EntityNode, tx Transaction, batchSize int) error
    GetByUUID(ctx context.Context, uuid string) (*graph.EntityNode, error)
    GetByUUIDs(ctx context.Context, uuids []string) ([]*graph.EntityNode, error)
    GetByGroupIDs(ctx context.Context, groupIDs []string, limit, offset int) ([]*graph.EntityNode, error)
    Delete(ctx context.Context, uuid string, tx Transaction) error
    DeleteByGroupID(ctx context.Context, groupID string, tx Transaction, batchSize int) error
    LoadEmbeddings(ctx context.Context, node *graph.EntityNode) error
    LoadEmbeddingsBulk(ctx context.Context, nodes []*graph.EntityNode, batchSize int) error
}

type EntityEdgeRepository interface {
    Save(ctx context.Context, edge graph.EntityEdge, tx Transaction) error
    SaveBulk(ctx context.Context, edges []graph.EntityEdge, tx Transaction, batchSize int) error
    GetByUUID(ctx context.Context, uuid string) (*graph.EntityEdge, error)
    GetBetweenNodes(ctx context.Context, srcUUID, tgtUUID string) ([]*graph.EntityEdge, error)
    GetByNodeUUID(ctx context.Context, nodeUUID string) ([]*graph.EntityEdge, error)
    // KEY TEMPORAL OPERATION: marks edge as invalid (NEVER deletes)
    Invalidate(ctx context.Context, uuid string, invalidAt time.Time, tx Transaction) error
    Delete(ctx context.Context, uuid string, tx Transaction) error
    LoadEmbeddings(ctx context.Context, edge *graph.EntityEdge) error
}

type EpisodeNodeRepository interface {
    Save(ctx context.Context, node graph.EpisodicNode, tx Transaction) error
    GetByUUID(ctx context.Context, uuid string) (*graph.EpisodicNode, error)
    GetByEntityNodeUUID(ctx context.Context, entityNodeUUID string) ([]*graph.EpisodicNode, error)
    RetrieveEpisodes(ctx context.Context, req RetrieveEpisodesReq) ([]*graph.EpisodicNode, error)
    Delete(ctx context.Context, uuid string, tx Transaction) error
}

type RetrieveEpisodesReq struct {
    ReferenceTime *time.Time
    LastN         int
    GroupIDs      []string
    Source        *graph.EpisodeType  // optional filter
    SagaID        string              // optional filter
}

type SearchRepository interface {
    NodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, labels []string) ([]*graph.EntityNode, error)
    NodeSimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.EntityNode, error)
    NodeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityNode, error)
    EdgeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, filters EdgeSearchFilters) ([]*graph.EntityEdge, error)
    EdgeSimilaritySearch(ctx context.Context, req EdgeSimilarityReq) ([]*graph.EntityEdge, error)
    EdgeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityEdge, error)
    EpisodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.EpisodicNode, error)
    CommunityFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.CommunityNode, error)
    CommunitySimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.CommunityNode, error)
    NodeDistanceReranker(ctx context.Context, nodeUUIDs []string, centerUUID string) (map[string]float64, error)
    EpisodeMentionsReranker(ctx context.Context, nodeUUIDs []string) (map[string]int, error)
}

type EdgeSimilarityReq struct {
    Vector     []float32
    SourceUUID string
    TargetUUID string
    GroupIDs   []string
    Limit      int
    MinScore   float64
    Filters    EdgeSearchFilters
}

type EdgeSearchFilters struct {
    ValidAt        *time.Time
    InvalidAt      *time.Time
    CreatedAtStart *time.Time
    CreatedAtEnd   *time.Time
    EntityLabels   []string
}

type BulkRepository interface {
    SaveBulk(ctx context.Context, req SaveBulkReq) error
}

type SaveBulkReq struct {
    Episode            graph.EpisodicNode
    EntityNodes        []graph.EntityNode
    EntityEdges        []graph.EntityEdge
    EpisodicEdges      []graph.EpisodicEdge
    SagaNode           *graph.SagaNode         // optional
    HasEpisodeEdges    []graph.HasEpisodeEdge
    NextEpisodeEdges   []graph.NextEpisodeEdge
    InvalidatedEdgeIDs []string                // edges to invalidate before save
    GroupID            string
}

type MaintenanceRepository interface {
    ClearData(ctx context.Context, groupIDs []string) error
    BuildIndicesAndConstraints(ctx context.Context, deleteExisting bool) error
    DeleteAllIndexes(ctx context.Context) error
    GetCommunityClusters(ctx context.Context, groupIDs []string) ([][]string, error)
    RemoveCommunities(ctx context.Context, groupID string) error
    DetermineEntityCommunity(ctx context.Context, entityUUID string) (string, error)
    GetMentionedNodes(ctx context.Context, episodeUUIDs []string) ([]*graph.EntityNode, error)
}
```

---

## 4. Neo4j Driver Implementation — `internal/adapter/driver/neo4j/`

### 4.1. EntityEdge — Bi-Temporal Save + Invalidate

```go
// services/graphiti-store/internal/adapter/driver/neo4j/entity_edge_repo.go

package neo4j

import (
    "context"
    "time"
    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type entityEdgeRepo struct {
    driver neo4jDriver
}

// Save creates EntityEdge as Neo4j relationship with bi-temporal fields
func (r *entityEdgeRepo) Save(ctx context.Context, edge graph.EntityEdge, tx port.Transaction) error {
    cypher := `
        MATCH (src {uuid: $src_uuid}), (tgt {uuid: $tgt_uuid})
        MERGE (src)-[e:RELATES_TO {uuid: $uuid}]->(tgt)
        SET e.name           = $name,
            e.fact           = $fact,
            e.episodes       = $episodes,
            e.group_id       = $group_id,
            e.valid_at       = $valid_at,
            e.invalid_at     = null,
            e.expired_at     = null,
            e.created_at     = datetime(),
            e.updated_at     = datetime()
    `
    params := map[string]any{
        "uuid":      edge.UUID,
        "src_uuid":  edge.SourceNodeUUID,
        "tgt_uuid":  edge.TargetNodeUUID,
        "name":      edge.Name,
        "fact":      edge.Fact,
        "episodes":  edge.Episodes,
        "group_id":  edge.GroupID,
        "valid_at":  edge.ValidAt,
    }
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

// Invalidate marks an EntityEdge as invalid (temporal invalidation — NEVER deletes)
// This is the KEY invariant of the temporal graph
func (r *entityEdgeRepo) Invalidate(ctx context.Context, uuid string, invalidAt time.Time, tx port.Transaction) error {
    cypher := `
        MATCH ()-[e:RELATES_TO {uuid: $uuid}]->()
        SET e.invalid_at = $invalid_at,
            e.expired_at = datetime(),
            e.updated_at = datetime()
    `
    params := map[string]any{
        "uuid":       uuid,
        "invalid_at": invalidAt,
    }
    if tx != nil {
        _, err := tx.Run(ctx, cypher, params)
        return err
    }
    _, err := r.driver.ExecuteQuery(ctx, cypher, params)
    return err
}

// GetBetweenNodes returns ALL edges (including invalidated) between two nodes
// Caller must filter based on temporal requirements
func (r *entityEdgeRepo) GetBetweenNodes(ctx context.Context, srcUUID, tgtUUID string) ([]*graph.EntityEdge, error) {
    cypher := `
        MATCH (src {uuid: $src_uuid})-[e:RELATES_TO]->(tgt {uuid: $tgt_uuid})
        RETURN e.uuid, e.name, e.fact, e.fact_embedding, e.episodes,
               e.valid_at, e.invalid_at, e.expired_at, e.created_at, e.group_id
        ORDER BY e.created_at DESC
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "src_uuid": srcUUID, "tgt_uuid": tgtUUID,
    })
    if err != nil { return nil, err }
    return mapRecordsToEntityEdges(records), nil
}
```

### 4.2. Atomic SaveBulk

```go
// services/graphiti-store/internal/adapter/driver/neo4j/bulk_repo.go

func (r *bulkRepo) SaveBulk(ctx context.Context, req port.SaveBulkReq) error {
    // Neo4j: execute everything in a single explicit transaction
    tx, err := r.driver.BeginTransaction(ctx)
    if err != nil { return fmt.Errorf("begin tx: %w", err) }

    defer func() {
        if err != nil { tx.Rollback(ctx) }
    }()

    // 1. Invalidate old edges first (before adding new ones)
    for _, edgeID := range req.InvalidatedEdgeIDs {
        if err = r.entityEdges.Invalidate(ctx, edgeID, time.Now(), tx); err != nil {
            return fmt.Errorf("invalidate edge %s: %w", edgeID, err)
        }
    }

    // 2. Save new entity nodes
    if err = r.entityNodes.SaveBulk(ctx, req.EntityNodes, tx, 100); err != nil {
        return fmt.Errorf("save entity nodes: %w", err)
    }

    // 3. Save new entity edges
    if err = r.entityEdges.SaveBulk(ctx, req.EntityEdges, tx, 100); err != nil {
        return fmt.Errorf("save entity edges: %w", err)
    }

    // 4. Save episode node
    if err = r.episodeNodes.Save(ctx, req.Episode, tx); err != nil {
        return fmt.Errorf("save episode node: %w", err)
    }

    // 5. Save episodic edges (MENTIONS)
    if err = r.episodicEdges.SaveBulk(ctx, req.EpisodicEdges, tx); err != nil {
        return fmt.Errorf("save episodic edges: %w", err)
    }

    // 6. Save saga (optional)
    if req.SagaNode != nil {
        if err = r.sagaNodes.Save(ctx, *req.SagaNode, tx); err != nil {
            return fmt.Errorf("save saga: %w", err)
        }
        for _, e := range req.HasEpisodeEdges {
            if err = r.hasEpisodeEdges.Save(ctx, e, tx); err != nil {
                return fmt.Errorf("save has_episode edge: %w", err)
            }
        }
        for _, e := range req.NextEpisodeEdges {
            if err = r.nextEpisodeEdges.Save(ctx, e, tx); err != nil {
                return fmt.Errorf("save next_episode edge: %w", err)
            }
        }
    }

    return tx.Commit(ctx)
}
```

### 4.3. BFS Traversal

```go
// services/graphiti-store/internal/adapter/driver/neo4j/search_repo.go

func (r *searchRepo) NodeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityNode, error) {
    // BFS via Cypher variable-length paths
    cypher := fmt.Sprintf(`
        MATCH path = (origin:Entity)-[:RELATES_TO*1..%d]-(n:Entity)
        WHERE origin.uuid IN $origin_uuids
          AND n.group_id IN $group_ids
          AND (n.invalid_at IS NULL OR n.invalid_at > datetime())
        WITH DISTINCT n, min(length(path)) as distance
        ORDER BY distance ASC
        LIMIT $limit
        RETURN n
    `, maxDepth)

    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "origin_uuids": originUUIDs,
        "group_ids":    groupIDs,
        "limit":        limit,
    })
    if err != nil { return nil, err }
    return mapRecordsToEntityNodes(records), nil
}

// NodeDistanceReranker returns hop distance from center node
func (r *searchRepo) NodeDistanceReranker(ctx context.Context, nodeUUIDs []string, centerUUID string) (map[string]float64, error) {
    cypher := `
        MATCH (center {uuid: $center_uuid})
        UNWIND $node_uuids AS targetUUID
        MATCH path = shortestPath((center)-[:RELATES_TO*1..5]-(target {uuid: targetUUID}))
        RETURN targetUUID, length(path) as distance
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
        "center_uuid": centerUUID,
        "node_uuids":  nodeUUIDs,
    })
    if err != nil { return nil, err }

    scores := make(map[string]float64)
    for _, rec := range records {
        uuid := rec.Values[0].(string)
        dist := rec.Values[1].(int64)
        // Score inversely proportional to distance (closer = higher score)
        scores[uuid] = 1.0 / (float64(dist) + 1.0)
    }
    return scores, nil
}

// EpisodeMentionsReranker boosts nodes mentioned in more episodes
func (r *searchRepo) EpisodeMentionsReranker(ctx context.Context, nodeUUIDs []string) (map[string]int, error) {
    cypher := `
        MATCH (ep:Episodic)-[:MENTIONS]->(n:Entity)
        WHERE n.uuid IN $node_uuids
        RETURN n.uuid as node_uuid, count(ep) as mention_count
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"node_uuids": nodeUUIDs})
    if err != nil { return nil, err }

    counts := make(map[string]int)
    for _, rec := range records {
        counts[rec.Values[0].(string)] = int(rec.Values[1].(int64))
    }
    return counts, nil
}
```

---

## 5. Neo4j Index Migrations — `db/migrations/graphiti/`

```cypher
-- 001_initial_schema.cypher

CREATE CONSTRAINT entity_node_uuid IF NOT EXISTS
    FOR (n:Entity) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT episodic_node_uuid IF NOT EXISTS
    FOR (n:Episodic) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT community_node_uuid IF NOT EXISTS
    FOR (n:Community) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT saga_node_uuid IF NOT EXISTS
    FOR (n:Saga) REQUIRE n.uuid IS UNIQUE;
```

```cypher
-- 002_vector_indices.cypher

CREATE VECTOR INDEX entity_name_embedding IF NOT EXISTS
    FOR (n:Entity) ON (n.name_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};

CREATE VECTOR INDEX entity_edge_fact_embedding IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.fact_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};

CREATE VECTOR INDEX community_name_embedding IF NOT EXISTS
    FOR (n:Community) ON (n.name_embedding)
    OPTIONS {indexConfig: {`vector.dimensions`: 1536, `vector.similarity_function`: 'cosine'}};
```

```cypher
-- 003_fulltext_indices.cypher

CREATE FULLTEXT INDEX entity_fulltext IF NOT EXISTS
    FOR (n:Entity) ON EACH [n.name, n.summary];

CREATE FULLTEXT INDEX episode_fulltext IF NOT EXISTS
    FOR (n:Episodic) ON EACH [n.content, n.source_description];

CREATE FULLTEXT INDEX community_fulltext IF NOT EXISTS
    FOR (n:Community) ON EACH [n.name, n.summary];

CREATE FULLTEXT INDEX entity_edge_fulltext IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON EACH [r.fact, r.name];
```

---

## 6. Community Cluster Queries

```go
// GetCommunityClusters returns adjacency-based clusters using Neo4j GDS
func (r *maintenanceRepo) GetCommunityClusters(ctx context.Context, groupIDs []string) ([][]string, error) {
    // Option A: Use Neo4j GDS Label Propagation (if GDS plugin available)
    // Option B: In-memory BFS clustering (pure Cypher, no plugin required)
    cypher := `
        MATCH (n:Entity)-[:RELATES_TO]->(m:Entity)
        WHERE n.group_id IN $group_ids AND m.group_id IN $group_ids
          AND n.invalid_at IS NULL AND m.invalid_at IS NULL
        RETURN n.uuid as source, collect(DISTINCT m.uuid) as neighbors
    `
    records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"group_ids": groupIDs})
    if err != nil { return nil, err }

    // Build adjacency map
    adj := make(map[string][]string)
    allNodes := make(map[string]bool)
    for _, rec := range records {
        src := rec.Values[0].(string)
        neighbors := toStringSlice(rec.Values[1])
        adj[src] = neighbors
        allNodes[src] = true
        for _, n := range neighbors { allNodes[n] = true }
    }

    // BFS to find connected components (clusters)
    return bfsComponents(adj, allNodes), nil
}

func bfsComponents(adj map[string][]string, nodes map[string]bool) [][]string {
    visited := make(map[string]bool)
    var clusters [][]string
    for node := range nodes {
        if visited[node] { continue }
        cluster := bfsFrom(node, adj, visited)
        if len(cluster) > 1 { clusters = append(clusters, cluster) }
    }
    return clusters
}
```

---

## 7. Multi-Tenant Group Enforcement

```go
// Multi-tenancy is enforced at the DRIVER level — every query includes group_id filter

// Neo4j driver middleware: wrap all ExecuteQuery calls
func (d *neo4jDriver) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]Record, error) {
    // Validate group_id is present in params for all read queries
    // (write queries must explicitly set group_id on nodes)
    return d.session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        result, err := tx.Run(ctx, query, params)
        // ...
    })
}
```

---

## 8. FalkorDB Driver (Stub for Wave 1, Full in Wave 2)

```go
// services/graphiti-store/internal/adapter/driver/falkordb/driver.go
// FalkorDB uses separate graph per group_id for isolation

type falkorDBDriver struct {
    client *redisgraph.Client
    graphs map[string]*redisgraph.Graph  // key: group_id
}

func (d *falkorDBDriver) getGraph(groupID string) *redisgraph.Graph {
    if g, ok := d.graphs[groupID]; ok { return g }
    // Create new graph for this group
    g := redisgraph.GraphNew(groupID, d.client)
    d.graphs[groupID] = g
    return g
}

// Note: FalkorDB does NOT support real transactions
// SaveBulk on FalkorDB is best-effort (no rollback)
func (d *falkorDBDriver) BeginTransaction(ctx context.Context) (Transaction, error) {
    return &noopTransaction{driver: d}, nil  // Noop wrapper
}
```

---

## 9. gRPC Service — `internal/adapter/grpc/handler.go`

```go
// Full gRPC handlers wrap repository operations
// 50+ RPCs mapped to driver operations

func (h *StoreHandler) InvalidateEntityEdge(ctx context.Context, req *pb.InvalidateEntityEdgeRequest) (*pb.InvalidateEntityEdgeResponse, error) {
    invalidAt := req.InvalidAt.AsTime()
    err := h.driver.EntityEdges().Invalidate(ctx, req.EdgeUuid, invalidAt, nil)
    if err != nil { return nil, status.Errorf(codes.Internal, "invalidate edge: %v", err) }
    return &pb.InvalidateEntityEdgeResponse{Success: true}, nil
}

func (h *StoreHandler) SaveBulk(ctx context.Context, req *pb.SaveBulkRequest) (*pb.SaveBulkResponse, error) {
    err := h.driver.Bulk().SaveBulk(ctx, mapPBToBulkReq(req))
    if err != nil { return nil, status.Errorf(codes.Internal, "save bulk: %v", err) }
    return &pb.SaveBulkResponse{Success: true}, nil
}

func (h *StoreHandler) NodeBFSSearch(ctx context.Context, req *pb.NodeBFSSearchRequest) (*pb.NodeBFSSearchResponse, error) {
    nodes, err := h.driver.Search().NodeBFSSearch(ctx, req.OriginUuids, int(req.MaxDepth), req.GroupIds, int(req.Limit))
    if err != nil { return nil, status.Errorf(codes.Internal, "bfs search: %v", err) }
    return &pb.NodeBFSSearchResponse{Nodes: mapNodesToProto(nodes)}, nil
}
```

---

## 10. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `pkg/graph/ontology.go` | EntityTypeSchema, EdgeTypeSchema, OntologyRegistry |
| `services/graphiti-store/internal/usecase/port/output.go` | GraphDriver + all repository interfaces |
| `services/graphiti-store/internal/adapter/driver/neo4j/entity_node_repo.go` | EntityNode Neo4j CRUD |
| `services/graphiti-store/internal/adapter/driver/neo4j/entity_edge_repo.go` | EntityEdge + Invalidate |
| `services/graphiti-store/internal/adapter/driver/neo4j/episode_node_repo.go` | EpisodicNode CRUD |
| `services/graphiti-store/internal/adapter/driver/neo4j/community_repo.go` | CommunityNode CRUD |
| `services/graphiti-store/internal/adapter/driver/neo4j/saga_repo.go` | SagaNode CRUD |
| `services/graphiti-store/internal/adapter/driver/neo4j/edge_repos.go` | EpisodicEdge, CommunityEdge, HasEpisode, NextEpisode |
| `services/graphiti-store/internal/adapter/driver/neo4j/bulk_repo.go` | Atomic SaveBulk in transaction |
| `services/graphiti-store/internal/adapter/driver/neo4j/search_repo.go` | BFS, similarity, fulltext, rerankers |
| `services/graphiti-store/internal/adapter/driver/neo4j/maintenance_repo.go` | ClearData, BuildIndices, GetCommunityClusters |
| `services/graphiti-store/internal/adapter/driver/falkordb/driver.go` | FalkorDB stub |
| `services/graphiti-store/internal/adapter/driver/kuzu/driver.go` | Kuzu stub |
| `db/migrations/graphiti/001_initial_schema.cypher` | Node constraints |
| `db/migrations/graphiti/002_vector_indices.cypher` | Neo4j vector indices |
| `db/migrations/graphiti/003_fulltext_indices.cypher` | Fulltext indices |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/graphiti-store/internal/adapter/grpc/handler.go` | Implement all 50+ RPCs |
| `services/graphiti-store/api/proto/store/v1/store.proto` | Full 50+ RPC contract |
| `apps/memory/internal/bootstrap/graphiti.go` | Init neo4j driver + inject into store handler |

---

## 11. Acceptance Criteria Mapping

| AC từ CR-GR-002 | Covered by |
|----------------|-----------|
| SaveEntityNode → GetEntityNode trả về đúng | entityNodeRepo.Save() + GetByUUID() |
| SaveEntityEdge → GetEntityEdge trả về fact | entityEdgeRepo.Save() |
| InvalidateEntityEdge → invalid_at set, search không trả về | Invalidate() + temporal filter |
| SaveBulk atomic → nếu fail không có partial data | Neo4j explicit transaction |
| NodeSimilaritySearch → sorted by cosine | Neo4j vector index query |
| NodeBFSSearch max_depth=2 → 2 hops | NodeBFSSearch() Cypher |
| GetCommunityClusters → [][]string | bfsComponents() |
| Switch Neo4j → FalkorDB (config only) | GraphDriver interface |
| ClearData(group_alpha) → chỉ xóa group-alpha | group_id filter enforcement |
| BuildIndicesAndConstraints → Neo4j indices | maintenanceRepo.BuildIndicesAndConstraints() |
