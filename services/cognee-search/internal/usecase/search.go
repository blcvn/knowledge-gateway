package usecase

import (
	"context"
	"errors"
	"sync"
	"golang.org/x/sync/errgroup"
	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/dto"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type Registry interface {
	Get(strategy domain.SearchStrategy) (port.Retriever, error)
}

type searchUseCase struct {
	registry Registry
	reranker port.Reranker
}

func NewSearchUseCase(registry Registry, reranker port.Reranker) port.SearchUseCase {
	return &searchUseCase{
		registry: registry,
		reranker: reranker,
	}
}

func (uc *searchUseCase) Execute(ctx context.Context, req dto.SearchRequest) (*dto.SearchResponse, error) {
	if req.Query == "" {
		return nil, domain.ErrEmptyQuery
	}
	if len(req.Strategies) == 0 {
		return nil, errors.New("at least one strategy must be provided")
	}

	// Phase 1: RETRIEVE — run selected strategies concurrently
	var allResults []domain.SearchResult
	var mu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)

	for _, s := range req.Strategies {
		strategy := domain.SearchStrategy(s)
		
		retriever, err := uc.registry.Get(strategy)
		if err != nil {
			return nil, err
		}

		g.Go(func() error {
			results, retrieveErr := retriever.Retrieve(gCtx, req.Query, req.TopK, req.Filters)
			if retrieveErr != nil {
				// Depending on requirements, we can log and continue or return the error.
				// Returning error aborts all other running routines via gCtx.
				return retrieveErr
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Phase 2: MERGE — RRF deduplication + scoring
	// We pass TopK * 3 to merge to have a good candidate pool for reranking
	merged := merge(allResults, req.TopK*3)

	// Phase 3: RERANK — cross-encoder
	if req.Rerank && uc.reranker != nil && len(merged) > 0 {
		merged = uc.reranker.Rerank(ctx, req.Query, merged, req.TopK)
	}

	// Ensure we only return TopK at most
	if len(merged) > req.TopK {
		merged = merged[:req.TopK]
	}

	return &dto.SearchResponse{
		Results: merged,
	}, nil
}
