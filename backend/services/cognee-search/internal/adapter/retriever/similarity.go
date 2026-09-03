package retriever

import (
	"context"

	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type similarityRetriever struct {
	vectorSearcher port.VectorSearcher
	llmClient      port.LLMClient
}

func NewSimilarityRetriever(vectorSearcher port.VectorSearcher, llmClient port.LLMClient) port.Retriever {
	return &similarityRetriever{
		vectorSearcher: vectorSearcher,
		llmClient:      llmClient,
	}
}

func (r *similarityRetriever) Retrieve(ctx context.Context, query string, topK int, filters domain.SearchFilters) ([]domain.SearchResult, error) {
	// 1. Embed query
	emb, err := r.llmClient.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// 2. Vector search
	results, err := r.vectorSearcher.SearchSimilar(ctx, emb, topK, filters.TenantID, filters.DatasetID)
	if err != nil {
		return nil, err
	}
	
	for i := range results {
		results[i].Strategy = domain.Similarity
	}
	
	return results, nil
}

func (r *similarityRetriever) Strategy() domain.SearchStrategy { return domain.Similarity }
func (r *similarityRetriever) RequiresLLM() bool { return false } // Requires LLM for embedding only, not generation
