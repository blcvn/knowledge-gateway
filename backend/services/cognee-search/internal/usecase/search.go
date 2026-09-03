package usecase

import (
	"context"
	"fmt"
)

// Retriever is a search backend that can retrieve results.
type Retriever interface {
	Strategy() SearchStrategy
	Retrieve(ctx context.Context, req SearchRequest) ([]SearchResult, error)
}

// SearchUseCase dispatches queries to the appropriate retrievers.
type SearchUseCase struct {
	retrievers map[SearchStrategy]Retriever
}

// NewSearchUseCase creates a SearchUseCase with the given retrievers.
func NewSearchUseCase(retrievers ...Retriever) *SearchUseCase {
	m := make(map[SearchStrategy]Retriever, len(retrievers))
	for _, r := range retrievers {
		m[r.Strategy()] = r
	}
	return &SearchUseCase{retrievers: m}
}

// Execute dispatches the search request to all requested strategies and merges results.
func (uc *SearchUseCase) Execute(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if len(req.Strategies) == 0 {
		req.Strategies = []SearchStrategy{StrategyGraphCompletion}
	}
	if req.TopK == 0 {
		req.TopK = 10
	}

	var allResults []SearchResult
	for _, strategy := range req.Strategies {
		retriever, ok := uc.retrievers[strategy]
		if !ok {
			return nil, fmt.Errorf("unknown strategy: %s", strategy)
		}
		results, err := retriever.Retrieve(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("retriever %s: %w", strategy, err)
		}
		allResults = append(allResults, results...)
	}

	resp := &SearchResponse{Results: allResults}
	return resp, nil
}
