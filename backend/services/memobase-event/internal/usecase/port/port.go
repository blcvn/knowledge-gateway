// Package port defines outbound ports for memobase-event service.
// SOL-MB-004: Event Timeline & Semantic Search
package port

import (
	"context"

	"vnp-memory/services/memobase-event/internal/domain"
)

// EventRepository is the storage port for user events.
type EventRepository interface {
	Save(ctx context.Context, e domain.Event) (*domain.Event, error)
	GetByUser(ctx context.Context, userID, projectID string, limit int) ([]domain.Event, error)
	UpdateEvent(ctx context.Context, id, userID, projectID string, data domain.EventData) error
	DeleteEvent(ctx context.Context, id, userID, projectID string) error
	DeleteByUser(ctx context.Context, userID, projectID string) error
	SearchByEmbedding(ctx context.Context, q SearchByEmbeddingQuery) ([]domain.SearchResult, error)
	FilterByTags(ctx context.Context, req FilterByTagsRequest) ([]domain.Event, error)
}

// GistRepository is the storage port for event gists.
type GistRepository interface {
	SaveBulk(ctx context.Context, gists []domain.EventGist) error
	SearchByEmbedding(ctx context.Context, q GistSearchQuery) ([]domain.GistSearchResult, error)
	GetRecentByUser(ctx context.Context, userID, projectID string, limit int) ([]domain.EventGist, error)
	DeleteByEvent(ctx context.Context, eventID string) error
	DeleteByUser(ctx context.Context, userID, projectID string) error
}

// Embedder embeds text queries for semantic search.
type Embedder interface {
	IsEnabled() bool
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
}

// SearchByEmbeddingQuery parameterizes a pgvector cosine-similarity search.
type SearchByEmbeddingQuery struct {
	UserID     string
	ProjectID  string
	Vector     []float32
	Threshold  float64
	TimeRange  int // days, 0 = no limit
	Limit      int
}

// GistSearchQuery parameterizes a pgvector cosine-similarity search on gists.
type GistSearchQuery struct {
	UserID    string
	ProjectID string
	Vector    []float32
	Threshold float64
	Limit     int
}

// FilterByTagsRequest parameterizes a JSONB tag-based filter.
type FilterByTagsRequest struct {
	UserID        string
	ProjectID     string
	TimeRangeDays int
	HasEventTag   []string          // tag key must exist
	EventTagEqual map[string]string // tag=value pairs
}
