// Package port defines interfaces for search-service.
package port

import (
	"context"

	"vnp-memory/services/search-service/internal/domain/connector"
	"vnp-memory/services/search-service/internal/domain/search"
)

// ── Backend Engine Clients ─────────────────────────────────────────────────

// KGClient is the interface to kg-service.
type KGClient interface {
	GraphitiSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
	CogneeSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
}

// MemoryClient is the interface to memory-service.
type MemoryClient interface {
	MemobaseSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
	SMSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
}

// StorageClient is the interface to storage-service.
type StorageClient interface {
	FileSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
}

// Reranker applies score-based reranking algorithms.
type Reranker interface {
	RRF(results [][]*search.Item, k int) []*search.Item
}

// ConnectorRepository persists connector configurations.
type ConnectorRepository interface {
	Create(ctx context.Context, c *connector.Connector) error
	GetByID(ctx context.Context, id string) (*connector.Connector, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*connector.Connector, error)
	CreateJob(ctx context.Context, job *connector.SyncJob) error
}

// EventPublisher publishes domain events.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}
