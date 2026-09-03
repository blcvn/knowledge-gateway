package domain

import (
	"context"
	"io"
	"time"
)

// GraphDriver is the composite interface that all graph database backends must implement.
// It follows the Strategy pattern — a single implementation covers all repository contracts.
//
// Backend selection is done at startup via the DRIVER_PROVIDER config.
// Neo4j is the primary implementation; FalkorDB, Kuzu, Neptune are pluggable alternatives.
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

// NodeRepository handles CRUD operations for all node types.
type NodeRepository interface {
	// SaveNode creates or updates an EntityNode (MERGE by UUID).
	SaveNode(ctx context.Context, node EntityNode) error

	// GetNode retrieves an EntityNode by UUID, scoped to group_id.
	GetNode(ctx context.Context, groupID, uuid string) (*EntityNode, error)

	// GetNodeByName retrieves an EntityNode by exact name within a group.
	GetNodeByName(ctx context.Context, groupID, name string) (*EntityNode, error)

	// DeleteNode removes a node and all its connected relationships.
	DeleteNode(ctx context.Context, groupID, uuid string) error

	// ListNodes returns paginated EntityNodes for a group.
	ListNodes(ctx context.Context, groupID string, opts PaginationOpts) ([]EntityNode, string, error)

	// SaveEpisodicNode creates or updates an EpisodicNode.
	SaveEpisodicNode(ctx context.Context, node EpisodicNode) error

	// SaveCommunityNode creates or updates a CommunityNode.
	SaveCommunityNode(ctx context.Context, node CommunityNode) error

	// SaveSagaNode creates or updates a SagaNode.
	SaveSagaNode(ctx context.Context, node SagaNode) error
}

// EdgeRepository handles CRUD operations for edges with bi-temporal support.
type EdgeRepository interface {
	// SaveEdge creates a RELATES_TO relationship between two Entity nodes.
	SaveEdge(ctx context.Context, edge EntityEdge) error

	// GetEdge retrieves an EntityEdge by UUID.
	GetEdge(ctx context.Context, uuid string) (*EntityEdge, error)

	// DeleteEdge removes an edge.
	DeleteEdge(ctx context.Context, uuid string) error

	// InvalidateEdge sets invalid_at on an edge without deleting it.
	InvalidateEdge(ctx context.Context, uuid string, invalidAt time.Time) error

	// GetEdgesInTimeRange returns edges whose validity window intersects [from, to].
	GetEdgesInTimeRange(ctx context.Context, groupID string, from, to time.Time) ([]EntityEdge, error)

	// SaveEpisodicEdge creates a MENTIONS relationship between an episode and entity.
	SaveEpisodicEdge(ctx context.Context, edge EpisodicEdge) error
}

// CommunityRepository handles community-related operations.
type CommunityRepository interface {
	// GetCommunity retrieves a CommunityNode by UUID.
	GetCommunity(ctx context.Context, groupID, uuid string) (*CommunityNode, error)

	// ListCommunities returns all communities for a group.
	ListCommunities(ctx context.Context, groupID string) ([]CommunityNode, error)

	// DeleteCommunity removes a community and its HAS_MEMBER edges.
	DeleteCommunity(ctx context.Context, groupID, uuid string) error
}

// SearchRepository provides search primitives over the graph.
type SearchRepository interface {
	// CosineSimilaritySearch returns top-K nodes ordered by embedding similarity.
	CosineSimilaritySearch(ctx context.Context, groupID string, embedding EmbeddingVector, limit int) ([]SearchResult, error)

	// FulltextSearch returns BM25-ranked results from name/summary/fact.
	FulltextSearch(ctx context.Context, groupID, query string, limit int) ([]SearchResult, error)

	// BFSSearch traverses the graph from a start node to a configurable depth.
	BFSSearch(ctx context.Context, startNodeID string, depth, limit int) ([]SearchResult, error)
}

// IndexRepository manages graph database indexes.
type IndexRepository interface {
	// BuildIndices creates all required indexes (idempotent — uses IF NOT EXISTS).
	BuildIndices(ctx context.Context, groupID string, defs []IndexDefinition) error

	// DropIndices removes all indexes for a group.
	DropIndices(ctx context.Context, groupID string) error

	// ListIndices returns current index definitions.
	ListIndices(ctx context.Context) ([]IndexDefinition, error)
}

// BulkRepository handles batch operations with atomic transaction guarantees.
type BulkRepository interface {
	// SaveBulk persists nodes + edges + episode atomically in a single transaction.
	SaveBulk(ctx context.Context, nodes []EntityNode, edges []EntityEdge, episode EpisodicNode) error

	// RollbackBulk removes all nodes/edges created by a specific episode.
	RollbackBulk(ctx context.Context, episodeID string) error

	// DeleteByGroupID removes ALL data for a tenant (purge operation).
	DeleteByGroupID(ctx context.Context, groupID string) error
}

// Transaction represents a unit of work within the graph database.
type Transaction interface {
	// Commit finalises the transaction.
	Commit(ctx context.Context) error

	// Rollback aborts the transaction.
	Rollback(ctx context.Context) error
}

// TransactionManager creates and manages transactions.
type TransactionManager interface {
	// WithTransaction executes fn within a transaction. Auto-commits on success, rolls back on error.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
