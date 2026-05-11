package retriever

import (
	"context"
	"fmt"

	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type Registry struct {
	retrievers map[domain.SearchStrategy]port.Retriever
}

func NewRegistry(retrievers []port.Retriever) *Registry {
	m := make(map[domain.SearchStrategy]port.Retriever)
	for _, r := range retrievers {
		m[r.Strategy()] = r
	}
	return &Registry{retrievers: m}
}

func (r *Registry) Get(strategy domain.SearchStrategy) (port.Retriever, error) {
	retriever, ok := r.retrievers[strategy]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrStrategyNotFound, strategy)
	}
	return retriever, nil
}
