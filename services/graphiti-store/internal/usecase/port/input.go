// Package port defines the input and output boundaries for the graphiti-store usecase layer.
//
// Input ports define what the external world can ask of the service (handlers call these).
// Output ports define what the service needs from infrastructure (adapters implement these).
//
// The GraphDriver interface in the domain layer serves as the primary output port.
// This package adds additional output ports for cross-cutting concerns.
package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

// --- Input Ports ---

// NodeService provides operations for managing graph nodes.
type NodeService interface {
	// SaveNode creates or updates an entity node.
	SaveNode(ctx context.Context, req SaveNodeRequest) (*domain.EntityNode, error)

	// GetNode retrieves a node by UUID.
	GetNode(ctx context.Context, groupID, uuid string) (*domain.EntityNode, error)

	// DeleteNode removes a node and its relationships.
	DeleteNode(ctx context.Context, groupID, uuid string) error
}

// CommunityService provides operations for managing community nodes.
type CommunityService interface {
	// SaveCommunity creates or updates a community node.
	SaveCommunity(ctx context.Context, req SaveCommunityRequest) (*domain.CommunityNode, error)

	// GetCommunity retrieves a community node by UUID.
	GetCommunity(ctx context.Context, groupID, uuid string) (*domain.CommunityNode, error)

	// DeleteCommunity removes a community node and its relationships.
	DeleteCommunity(ctx context.Context, groupID, uuid string) error
}

// EdgeService provides operations for managing graph edges.
type EdgeService interface {
	// SaveEdge creates a bi-temporal relationship.
	SaveEdge(ctx context.Context, req SaveEdgeRequest) (*domain.EntityEdge, error)

	// GetEdge retrieves an edge by UUID.
	GetEdge(ctx context.Context, uuid string) (*domain.EntityEdge, error)

	// DeleteEdge removes an edge.
	DeleteEdge(ctx context.Context, uuid string) error

	// InvalidateEdge marks an edge as no longer valid.
	InvalidateEdge(ctx context.Context, req InvalidateEdgeRequest) error
}

// SearchService provides search primitives over the graph.
type SearchService interface {
	// CosineSimilaritySearch finds nodes by embedding similarity.
	CosineSimilaritySearch(ctx context.Context, req VectorSearchRequest) ([]domain.SearchResult, error)

	// FulltextSearch finds nodes by text matching (BM25).
	FulltextSearch(ctx context.Context, req TextSearchRequest) ([]domain.SearchResult, error)

	// BFSSearch traverses the graph breadth-first from a start node.
	BFSSearch(ctx context.Context, req BFSSearchRequest) ([]domain.SearchResult, error)
}

// BulkService provides batch operations.
type BulkService interface {
	// SaveBulk atomically persists nodes, edges, and episode.
	SaveBulk(ctx context.Context, req SaveBulkRequest) (*SaveBulkResponse, error)

	// RollbackBulk removes everything created by a specific episode.
	RollbackBulk(ctx context.Context, episodeID string) error

	// DeleteByGroupID purges all tenant data.
	DeleteByGroupID(ctx context.Context, groupID string) error
}

// IndexService manages graph database indexes.
type IndexService interface {
	// BuildIndices creates standard indexes for the graph.
	BuildIndices(ctx context.Context, groupID string) error

	// DropIndices removes standard indexes.
	DropIndices(ctx context.Context, groupID string) error

	// ListIndices returns standard index definitions.
	ListIndices(ctx context.Context) ([]domain.IndexDefinition, error)
}

// --- Output Ports ---

// EventPublisher publishes domain events to the message bus.
type EventPublisher interface {
	// PublishBulkSaved publishes an event when a bulk save completes.
	PublishBulkSaved(ctx context.Context, groupID, episodeID string, nodeCount, edgeCount int) error
}
