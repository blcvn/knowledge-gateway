package port

import (
	"context"
	"vnp-memory/services/cognee-search/internal/usecase/dto"
)

type SearchUseCase interface {
	Execute(ctx context.Context, req dto.SearchRequest) (*dto.SearchResponse, error)
}

type RAGCompleteUseCase interface {
	Execute(ctx context.Context, req dto.RAGRequest) (*dto.RAGResponse, error)
}

type ExploreGraphUseCase interface {
	Execute(ctx context.Context, req dto.ExploreRequest) (*dto.ExploreResponse, error)
}
