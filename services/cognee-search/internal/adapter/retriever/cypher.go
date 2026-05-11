package retriever

import (
	"context"

	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type cypherRetriever struct {
	graphSearcher port.GraphSearcher
}

func NewCypherRetriever(graphSearcher port.GraphSearcher) port.Retriever {
	return &cypherRetriever{
		graphSearcher: graphSearcher,
	}
}

func (r *cypherRetriever) Retrieve(ctx context.Context, query string, topK int, filters domain.SearchFilters) ([]domain.SearchResult, error) {
	// Here query is assumed to be raw Cypher (in a secure setup, this would be validated)
	results, err := r.graphSearcher.ExecuteCypher(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	for i := range results {
		results[i].Strategy = domain.Cypher
	}
	return results, nil
}

func (r *cypherRetriever) Strategy() domain.SearchStrategy { return domain.Cypher }
func (r *cypherRetriever) RequiresLLM() bool { return false }
