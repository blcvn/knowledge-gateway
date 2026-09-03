package retriever

import (
	"context"

	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type chunksRetriever struct {
	vectorSearcher port.VectorSearcher
}

func NewChunksRetriever(vectorSearcher port.VectorSearcher) port.Retriever {
	return &chunksRetriever{
		vectorSearcher: vectorSearcher,
	}
}

func (r *chunksRetriever) Retrieve(ctx context.Context, query string, topK int, filters domain.SearchFilters) ([]domain.SearchResult, error) {
	// Raw chunk retrieval (could be a simple match or keyword filter via Qdrant payloads)
	results, err := r.vectorSearcher.SearchChunks(ctx, query, topK, filters.TenantID, filters.DatasetID)
	if err != nil {
		return nil, err
	}
	for i := range results {
		results[i].Strategy = domain.Chunks
	}
	return results, nil
}

func (r *chunksRetriever) Strategy() domain.SearchStrategy { return domain.Chunks }
func (r *chunksRetriever) RequiresLLM() bool { return false }
