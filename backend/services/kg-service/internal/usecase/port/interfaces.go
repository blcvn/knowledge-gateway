// Package port defines output interfaces for kg-service usecases.
package port

import (
	"context"

	"vnp-memory/services/kg-service/internal/domain/cognee"
	"vnp-memory/services/kg-service/internal/domain/graphiti"
)

// EpisodeRepository persists episodes (pgvector).
type EpisodeRepository interface {
	Create(ctx context.Context, ep *graphiti.Episode) error
	FindByUUID(ctx context.Context, tenantID, uuid string) (*graphiti.Episode, error)
	SemanticSearch(ctx context.Context, tenantID string, embedding []float32, limit int) ([]*graphiti.Episode, error)
	TextSearch(ctx context.Context, tenantID, query string, limit int) ([]*graphiti.Episode, error)
}

// GraphRepository persists nodes and edges in Neo4j.
type GraphRepository interface {
	UpsertNode(ctx context.Context, node *graphiti.Node) error
	UpsertEdge(ctx context.Context, edge *graphiti.Edge) error
	GetNode(ctx context.Context, tenantID, uuid string) (*graphiti.Node, error)
	GetEdge(ctx context.Context, tenantID, uuid string) (*graphiti.Edge, error)
	GetNeighbors(ctx context.Context, tenantID, nodeUUID string, depth int) ([]*graphiti.Node, []*graphiti.Edge, error)
	GetOntology(ctx context.Context, tenantID string) (*graphiti.Ontology, error)
	UpdateOntology(ctx context.Context, ontology *graphiti.Ontology) error
	QuerySubgraph(ctx context.Context, tenantID, query string) ([]*graphiti.Node, []*graphiti.Edge, error)
}

// EmbeddingService generates text embeddings.
type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// EventPublisher publishes domain events.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

// DatasetRepository persists Cognee dataset metadata (PostgreSQL).
type DatasetRepository interface {
	Save(ctx context.Context, ds *cognee.Dataset) error
	FindByID(ctx context.Context, id string) (*cognee.Dataset, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*cognee.Dataset, error)
	UpdateStatus(ctx context.Context, id, status string) error
}

// CogneeClient is the interface to the Cognee Python service.
type CogneeClient interface {
	CreateDataset(ctx context.Context, name string) (*cognee.DatasetResponse, error)
	UploadData(ctx context.Context, datasetID string, item *cognee.DataItem) error
	Cognify(ctx context.Context, datasetID string) (*cognee.CognifyJob, error)
	Search(ctx context.Context, query string) ([]*cognee.SearchResult, error)
}
