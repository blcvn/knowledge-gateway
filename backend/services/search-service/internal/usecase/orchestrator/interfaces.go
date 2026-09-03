// Package orchestrator — interface re-exports for use by main.go wiring.
package orchestrator

import (
	"context"

	"vnp-memory/services/search-service/internal/domain/search"
)

// KGClientInterface is the interface to kg-service.
type KGClientInterface interface {
	GraphitiSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
	CogneeSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
}

// MemoryClientInterface is the interface to memory-service.
type MemoryClientInterface interface {
	MemobaseSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
	SMSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
}

// StorageClientInterface is the interface to storage-service.
type StorageClientInterface interface {
	FileSearch(ctx context.Context, q *search.Query) ([]*search.Item, error)
}
