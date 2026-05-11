package usecase

import (
	"context"
	"time"

	"vnp-memory/services/graphiti-search/internal/domain"
)

type StoreSearchClient interface {
	CosineSimilaritySearch(ctx context.Context, queryVector []float32, limit int) ([]domain.SearchResult, error)
	FulltextSearch(ctx context.Context, query string, limit int) ([]domain.SearchResult, error)
	BFSSearch(ctx context.Context, startNodeID string, maxDepth int) ([]domain.SearchResult, error)
	NodeSearch(ctx context.Context, labels []string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error)
	EdgeSearch(ctx context.Context, edgeType string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error)
	CommunitySearch(ctx context.Context, query string) ([]domain.SearchResult, error)
}

type EmbedderClient interface {
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
}

type CacheRepo interface {
	Get(ctx context.Context, key string) ([]domain.RankedResult, error)
	Set(ctx context.Context, key string, results []domain.RankedResult, ttl time.Duration) error
	InvalidateGroup(ctx context.Context, groupID string) error
}

type Reranker interface {
	Rerank(ctx context.Context, query string, results []domain.SearchResult) ([]domain.RankedResult, error)
	Type() domain.RerankerType
}
