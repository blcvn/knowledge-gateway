package port

import (
	"context"
	"time"

	"vnp-memory/pkg/graph"
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
	Episode            graph.EpisodicNode
	EntityNodes        []graph.EntityNode
	EntityEdges        []graph.EntityEdge
	EpisodicEdges      []graph.EpisodicEdge
	SagaNode           *graph.SagaNode
	HasEpisodeEdges    []graph.HasEpisodeEdge
	NextEpisodeEdges   []graph.NextEpisodeEdge
	InvalidatedEdgeIDs []string // mark invalid BEFORE saving new edges
	GroupID            string
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
