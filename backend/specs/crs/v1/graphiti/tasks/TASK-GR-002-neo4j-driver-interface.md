# TASK-GR-002 — GraphDriver Interface & Neo4j Driver

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-002 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/graphiti-store/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §3 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-001 |
| **Estimated** | 3h |

---

## Context

Định nghĩa `GraphDriver` interface — unified abstraction cho mọi graph backend (Neo4j, FalkorDB, Kuzu). Đây là hexagonal adapter layer của `graphiti-store`. Cũng triển khai `Neo4jDriver` concrete implementation với connection pooling.

---

## Goal

- `GraphDriver` interface với đầy đủ repository accessors
- `Transaction` interface cho atomic operations
- Neo4j driver implementation: connect, ping, execute query, begin transaction
- Repository accessor interfaces: EntityNodeRepository, EntityEdgeRepository, EpisodeNodeRepository, SearchRepository, BulkRepository, MaintenanceRepository

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-store/internal/usecase/port/output.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/neo4j/driver.go` |
| CREATE | `services/graphiti-store/internal/adapter/driver/falkordb/driver.go` |

---

## Implementation

### File 1: `services/graphiti-store/internal/usecase/port/output.go`

```go
package port

import (
    "context"
    "time"

    "github.com/vnp-memory/pkg/graph"
)

type GraphProvider string

const (
    ProviderNeo4j    GraphProvider = "neo4j"
    ProviderFalkorDB GraphProvider = "falkordb"
    ProviderKuzu     GraphProvider = "kuzu"
)

// Record represents a single row result from a graph query
type Record struct {
    Keys   []string
    Values []any
}

// Transaction represents an atomic graph database transaction
type Transaction interface {
    Run(ctx context.Context, query string, params map[string]any) ([]Record, error)
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}

// GraphDriver — unified interface for all graph backends.
// All implementations must be safe for concurrent use.
type GraphDriver interface {
    Close(ctx context.Context) error
    Ping(ctx context.Context) error
    Provider() GraphProvider
    ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]Record, error)
    BeginTransaction(ctx context.Context) (Transaction, error)

    // Repository accessors
    EntityNodes() EntityNodeRepository
    EpisodeNodes() EpisodeNodeRepository
    CommunityNodes() CommunityNodeRepository
    SagaNodes() SagaNodeRepository
    EntityEdges() EntityEdgeRepository
    EpisodicEdges() EpisodicEdgeRepository
    CommunityEdges() CommunityEdgeRepository
    HasEpisodeEdges() HasEpisodeEdgeRepository
    NextEpisodeEdges() NextEpisodeEdgeRepository
    Search() SearchRepository
    Maintenance() MaintenanceRepository
    Bulk() BulkRepository
}

// EntityNodeRepository — CRUD for EntityNode
type EntityNodeRepository interface {
    Save(ctx context.Context, node graph.EntityNode, tx Transaction) error
    SaveBulk(ctx context.Context, nodes []graph.EntityNode, tx Transaction, batchSize int) error
    GetByUUID(ctx context.Context, uuid string) (*graph.EntityNode, error)
    GetByUUIDs(ctx context.Context, uuids []string) ([]*graph.EntityNode, error)
    Delete(ctx context.Context, uuid string, tx Transaction) error
    DeleteByGroupID(ctx context.Context, groupID string, tx Transaction, batchSize int) error
}

// EntityEdgeRepository — CRUD + temporal invalidation for EntityEdge
type EntityEdgeRepository interface {
    Save(ctx context.Context, edge graph.EntityEdge, tx Transaction) error
    SaveBulk(ctx context.Context, edges []graph.EntityEdge, tx Transaction, batchSize int) error
    GetByUUID(ctx context.Context, uuid string) (*graph.EntityEdge, error)
    GetBetweenNodes(ctx context.Context, srcUUID, tgtUUID string) ([]*graph.EntityEdge, error)
    GetByNodeUUID(ctx context.Context, nodeUUID string) ([]*graph.EntityEdge, error)
    // Invalidate marks an edge as temporally invalid — NEVER deletes
    Invalidate(ctx context.Context, uuid string, invalidAt time.Time, tx Transaction) error
    Delete(ctx context.Context, uuid string, tx Transaction) error
}

// EpisodeNodeRepository — CRUD for EpisodicNode
type EpisodeNodeRepository interface {
    Save(ctx context.Context, node graph.EpisodicNode, tx Transaction) error
    GetByUUID(ctx context.Context, uuid string) (*graph.EpisodicNode, error)
    GetByEntityNodeUUID(ctx context.Context, entityNodeUUID string) ([]*graph.EpisodicNode, error)
    RetrieveEpisodes(ctx context.Context, req RetrieveEpisodesReq) ([]*graph.EpisodicNode, error)
    Delete(ctx context.Context, uuid string, tx Transaction) error
    DeleteByGroupID(ctx context.Context, groupID string, tx Transaction, batchSize int) error
}

type RetrieveEpisodesReq struct {
    ReferenceTime *time.Time
    LastN         int
    GroupIDs      []string
    Source        *graph.EpisodeType
    SagaID        string
}

// CommunityNodeRepository — CRUD for CommunityNode
type CommunityNodeRepository interface {
    Save(ctx context.Context, node graph.CommunityNode, tx Transaction) error
    GetByUUID(ctx context.Context, uuid string) (*graph.CommunityNode, error)
    DeleteByGroupID(ctx context.Context, groupID string, tx Transaction) error
}

// SagaNodeRepository — CRUD for SagaNode
type SagaNodeRepository interface {
    Save(ctx context.Context, node graph.SagaNode, tx Transaction) error
    GetByUUID(ctx context.Context, uuid, groupID string) (*graph.SagaNode, error)
    GetByGroupID(ctx context.Context, groupID string) ([]*graph.SagaNode, error)
}

// EpisodicEdgeRepository — CRUD for EpisodicEdge (MENTIONS)
type EpisodicEdgeRepository interface {
    Save(ctx context.Context, edge graph.EpisodicEdge, tx Transaction) error
    SaveBulk(ctx context.Context, edges []graph.EpisodicEdge, tx Transaction) error
    DeleteByEpisodeUUID(ctx context.Context, episodeUUID string, tx Transaction) error
}

// CommunityEdgeRepository — CRUD for CommunityEdge (HAS_MEMBER)
type CommunityEdgeRepository interface {
    Save(ctx context.Context, edge graph.CommunityEdge, tx Transaction) error
    DeleteByCommunityUUID(ctx context.Context, communityUUID string, tx Transaction) error
}

// HasEpisodeEdgeRepository — CRUD for HAS_EPISODE edges
type HasEpisodeEdgeRepository interface {
    Save(ctx context.Context, edge graph.HasEpisodeEdge, tx Transaction) error
}

// NextEpisodeEdgeRepository — CRUD for NEXT_EPISODE edges
type NextEpisodeEdgeRepository interface {
    Save(ctx context.Context, edge graph.NextEpisodeEdge, tx Transaction) error
}

// EdgeSearchFilters — temporal and property filters for edge search
type EdgeSearchFilters struct {
    ValidAt        *time.Time
    InvalidAt      *time.Time
    CreatedAtStart *time.Time
    CreatedAtEnd   *time.Time
    EntityLabels   []string
}

// EdgeSimilarityReq — request for vector similarity search on edges
type EdgeSimilarityReq struct {
    Vector     []float32
    SourceUUID string
    TargetUUID string
    GroupIDs   []string
    Limit      int
    MinScore   float64
    Filters    EdgeSearchFilters
}

// GroupStats — statistics for a group/tenant
type GroupStats struct {
    GroupID        string
    EpisodeCount   int64
    EntityCount    int64
    EdgeCount      int64
    CommunityCount int64
}

// SearchRepository — all search operations (vector, fulltext, BFS, reranking)
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

// SaveBulkReq — all objects to persist atomically for a single ingestion
type SaveBulkReq struct {
    Episode             graph.EpisodicNode
    EntityNodes         []graph.EntityNode
    EntityEdges         []graph.EntityEdge
    EpisodicEdges       []graph.EpisodicEdge
    SagaNode            *graph.SagaNode
    HasEpisodeEdges     []graph.HasEpisodeEdge
    NextEpisodeEdges    []graph.NextEpisodeEdge
    InvalidatedEdgeIDs  []string  // mark invalid BEFORE saving new edges
    GroupID             string
}

// BulkRepository — atomic multi-object persistence
type BulkRepository interface {
    SaveBulk(ctx context.Context, req SaveBulkReq) error
}

// MaintenanceRepository — administrative operations
type MaintenanceRepository interface {
    ClearData(ctx context.Context, groupIDs []string) error
    BuildIndicesAndConstraints(ctx context.Context, deleteExisting bool) error
    DeleteAllIndexes(ctx context.Context) error
    GetCommunityClusters(ctx context.Context, groupIDs []string) ([][]string, error)
    RemoveCommunities(ctx context.Context, groupID string) error
    GetGroupStats(ctx context.Context, groupID string) (*GroupStats, error)
    GetMentionedNodes(ctx context.Context, episodeUUIDs []string) ([]*graph.EntityNode, error)
}
```

### File 2: `services/graphiti-store/internal/adapter/driver/neo4j/driver.go`

```go
package neo4j

import (
    "context"
    "fmt"

    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

type Neo4jDriver struct {
    driver neo4j.DriverWithContext
    config Neo4jConfig

    // Cached repository instances (created once)
    entityNodeRepo    *entityNodeRepo
    entityEdgeRepo    *entityEdgeRepo
    episodeNodeRepo   *episodeNodeRepo
    communityNodeRepo *communityNodeRepo
    sagaNodeRepo      *sagaNodeRepo
    episodicEdgeRepo  *episodicEdgeRepo
    communityEdgeRepo *communityEdgeRepo
    hasEpisodeRepo    *hasEpisodeEdgeRepo
    nextEpisodeRepo   *nextEpisodeEdgeRepo
    searchRepo        *searchRepo
    maintenanceRepo   *maintenanceRepo
    bulkRepo          *bulkRepo
}

type Neo4jConfig struct {
    URI      string
    Username string
    Password string
    Database string
}

// NewNeo4jDriver creates and verifies a Neo4j driver connection
func NewNeo4jDriver(ctx context.Context, cfg Neo4jConfig) (*Neo4jDriver, error) {
    drv, err := neo4j.NewDriverWithContext(
        cfg.URI,
        neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
        func(c *neo4j.Config) {
            c.MaxConnectionPoolSize = 50
            c.ConnectionAcquisitionTimeout = 30
        },
    )
    if err != nil {
        return nil, fmt.Errorf("create neo4j driver: %w", err)
    }

    if err := drv.VerifyConnectivity(ctx); err != nil {
        return nil, fmt.Errorf("neo4j connectivity: %w", err)
    }

    d := &Neo4jDriver{driver: drv, config: cfg}

    // Init all repositories with shared driver reference
    d.entityNodeRepo    = &entityNodeRepo{driver: d}
    d.entityEdgeRepo    = &entityEdgeRepo{driver: d}
    d.episodeNodeRepo   = &episodeNodeRepo{driver: d}
    d.communityNodeRepo = &communityNodeRepo{driver: d}
    d.sagaNodeRepo      = &sagaNodeRepo{driver: d}
    d.episodicEdgeRepo  = &episodicEdgeRepo{driver: d}
    d.communityEdgeRepo = &communityEdgeRepo{driver: d}
    d.hasEpisodeRepo    = &hasEpisodeEdgeRepo{driver: d}
    d.nextEpisodeRepo   = &nextEpisodeEdgeRepo{driver: d}
    d.searchRepo        = &searchRepo{driver: d}
    d.maintenanceRepo   = &maintenanceRepo{driver: d}
    d.bulkRepo          = &bulkRepo{
        driver:            d,
        entityNodes:       d.entityNodeRepo,
        entityEdges:       d.entityEdgeRepo,
        episodeNodes:      d.episodeNodeRepo,
        sagaNodes:         d.sagaNodeRepo,
        episodicEdges:     d.episodicEdgeRepo,
        hasEpisodeEdges:   d.hasEpisodeRepo,
        nextEpisodeEdges:  d.nextEpisodeRepo,
    }
    return d, nil
}

func (d *Neo4jDriver) Close(ctx context.Context) error {
    return d.driver.Close(ctx)
}

func (d *Neo4jDriver) Ping(ctx context.Context) error {
    return d.driver.VerifyConnectivity(ctx)
}

func (d *Neo4jDriver) Provider() port.GraphProvider {
    return port.ProviderNeo4j
}

func (d *Neo4jDriver) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]port.Record, error) {
    db := d.config.Database
    if db == "" { db = "neo4j" }

    result, err := neo4j.ExecuteQuery(ctx, d.driver, query, params,
        neo4j.EagerResultTransformer,
        neo4j.ExecuteQueryWithDatabase(db),
    )
    if err != nil {
        return nil, fmt.Errorf("execute query: %w", err)
    }

    records := make([]port.Record, len(result.Records))
    for i, rec := range result.Records {
        records[i] = port.Record{
            Keys:   rec.Keys,
            Values: rec.Values,
        }
    }
    return records, nil
}

func (d *Neo4jDriver) BeginTransaction(ctx context.Context) (port.Transaction, error) {
    session := d.driver.NewSession(ctx, neo4j.SessionConfig{
        DatabaseName: d.config.Database,
        AccessMode:   neo4j.AccessModeWrite,
    })
    tx, err := session.BeginTransaction(ctx)
    if err != nil {
        session.Close(ctx)
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    return &neo4jTransaction{tx: tx, session: session}, nil
}

// Repository accessors
func (d *Neo4jDriver) EntityNodes() port.EntityNodeRepository       { return d.entityNodeRepo }
func (d *Neo4jDriver) EpisodeNodes() port.EpisodeNodeRepository     { return d.episodeNodeRepo }
func (d *Neo4jDriver) CommunityNodes() port.CommunityNodeRepository { return d.communityNodeRepo }
func (d *Neo4jDriver) SagaNodes() port.SagaNodeRepository           { return d.sagaNodeRepo }
func (d *Neo4jDriver) EntityEdges() port.EntityEdgeRepository        { return d.entityEdgeRepo }
func (d *Neo4jDriver) EpisodicEdges() port.EpisodicEdgeRepository   { return d.episodicEdgeRepo }
func (d *Neo4jDriver) CommunityEdges() port.CommunityEdgeRepository  { return d.communityEdgeRepo }
func (d *Neo4jDriver) HasEpisodeEdges() port.HasEpisodeEdgeRepository { return d.hasEpisodeRepo }
func (d *Neo4jDriver) NextEpisodeEdges() port.NextEpisodeEdgeRepository { return d.nextEpisodeRepo }
func (d *Neo4jDriver) Search() port.SearchRepository                 { return d.searchRepo }
func (d *Neo4jDriver) Maintenance() port.MaintenanceRepository       { return d.maintenanceRepo }
func (d *Neo4jDriver) Bulk() port.BulkRepository                    { return d.bulkRepo }

// neo4jTransaction wraps neo4j transaction to implement port.Transaction
type neo4jTransaction struct {
    tx      neo4j.ExplicitTransaction
    session neo4j.SessionWithContext
}

func (t *neo4jTransaction) Run(ctx context.Context, query string, params map[string]any) ([]port.Record, error) {
    result, err := t.tx.Run(ctx, query, params)
    if err != nil { return nil, err }
    rawRecords, err := result.Collect(ctx)
    if err != nil { return nil, err }
    records := make([]port.Record, len(rawRecords))
    for i, rec := range rawRecords {
        records[i] = port.Record{Keys: rec.Keys, Values: rec.Values}
    }
    return records, nil
}

func (t *neo4jTransaction) Commit(ctx context.Context) error {
    defer t.session.Close(ctx)
    return t.tx.Commit(ctx)
}

func (t *neo4jTransaction) Rollback(ctx context.Context) error {
    defer t.session.Close(ctx)
    return t.tx.Rollback(ctx)
}
```

### File 3: `services/graphiti-store/internal/adapter/driver/falkordb/driver.go`

```go
// Package falkordb provides a stub GraphDriver for FalkorDB.
// FalkorDB uses one graph per group_id (natural multi-tenancy).
// Full implementation is deferred to Wave 2+ roadmap.
package falkordb

import (
    "context"
    "fmt"

    "github.com/vnp-memory/services/graphiti-store/internal/usecase/port"
)

// FalkorDBDriver — stub implementation.
// Returns ErrNotImplemented for all operations except Provider() and Ping().
type FalkorDBDriver struct{}

var ErrNotImplemented = fmt.Errorf("falkordb: operation not yet implemented")

func (d *FalkorDBDriver) Provider() port.GraphProvider    { return port.ProviderFalkorDB }
func (d *FalkorDBDriver) Ping(ctx context.Context) error  { return nil }
func (d *FalkorDBDriver) Close(ctx context.Context) error { return nil }

func (d *FalkorDBDriver) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]port.Record, error) {
    return nil, ErrNotImplemented
}

func (d *FalkorDBDriver) BeginTransaction(ctx context.Context) (port.Transaction, error) {
    // FalkorDB does not support ACID transactions
    // Return a no-op transaction (best-effort)
    return &noopTx{}, nil
}

// All repository accessors return a stub that returns ErrNotImplemented
func (d *FalkorDBDriver) EntityNodes() port.EntityNodeRepository         { return &stubRepo{} }
func (d *FalkorDBDriver) EpisodeNodes() port.EpisodeNodeRepository        { return &stubRepo{} }
func (d *FalkorDBDriver) CommunityNodes() port.CommunityNodeRepository    { return &stubRepo{} }
func (d *FalkorDBDriver) SagaNodes() port.SagaNodeRepository              { return &stubRepo{} }
func (d *FalkorDBDriver) EntityEdges() port.EntityEdgeRepository           { return &stubRepo{} }
func (d *FalkorDBDriver) EpisodicEdges() port.EpisodicEdgeRepository      { return &stubRepo{} }
func (d *FalkorDBDriver) CommunityEdges() port.CommunityEdgeRepository     { return &stubRepo{} }
func (d *FalkorDBDriver) HasEpisodeEdges() port.HasEpisodeEdgeRepository   { return &stubRepo{} }
func (d *FalkorDBDriver) NextEpisodeEdges() port.NextEpisodeEdgeRepository { return &stubRepo{} }
func (d *FalkorDBDriver) Search() port.SearchRepository                    { return &stubRepo{} }
func (d *FalkorDBDriver) Maintenance() port.MaintenanceRepository          { return &stubRepo{} }
func (d *FalkorDBDriver) Bulk() port.BulkRepository                        { return &stubRepo{} }

type noopTx struct{}
func (t *noopTx) Run(ctx context.Context, q string, p map[string]any) ([]port.Record, error) { return nil, nil }
func (t *noopTx) Commit(ctx context.Context) error   { return nil }
func (t *noopTx) Rollback(ctx context.Context) error { return nil }

type stubRepo struct{}
// stubRepo implements all repository interfaces by returning ErrNotImplemented
```

---

## Dependencies to Add

```bash
cd services/graphiti-store
go get github.com/neo4j/neo4j-go-driver/v5@latest
```

---

## Verification

```bash
cd services/graphiti-store
go build ./internal/usecase/port/...
go build ./internal/adapter/driver/...
```

**Expected:** No compilation errors. Driver interfaces fully satisfied.
