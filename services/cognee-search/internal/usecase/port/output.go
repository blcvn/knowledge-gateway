package port

import (
	"context"
	"vnp-memory/services/cognee-search/internal/domain"
)

type Retriever interface {
	Retrieve(ctx context.Context, query string, topK int, filters domain.SearchFilters) ([]domain.SearchResult, error)
	Strategy() domain.SearchStrategy
	RequiresLLM() bool
}

type VectorSearcher interface {
	SearchSimilar(ctx context.Context, embedding []float32, topK int, tenantID, datasetID string) ([]domain.SearchResult, error)
	SearchChunks(ctx context.Context, query string, topK int, tenantID, datasetID string) ([]domain.SearchResult, error)
}

type GraphSearcher interface {
	ExecuteCypher(ctx context.Context, cypher string, params map[string]interface{}) ([]domain.SearchResult, error)
	SearchNodes(ctx context.Context, query string, topK int, tenantID string) ([]domain.SearchResult, error)
}

type Reranker interface {
	Rerank(ctx context.Context, query string, results []domain.SearchResult, topK int) []domain.SearchResult
}

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

type CacheStore interface {
	Get(ctx context.Context, key string) ([]domain.SearchResult, error)
	Set(ctx context.Context, key string, results []domain.SearchResult) error
	Invalidate(ctx context.Context, pattern string) error
}
