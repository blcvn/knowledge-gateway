package usecase

import (
	"context"

	"vnp-memory/services/graphiti-search/internal/domain"
)

type EdgeSearchUseCase struct {
	storeClient StoreSearchClient
}

func NewEdgeSearchUseCase(storeClient StoreSearchClient) *EdgeSearchUseCase {
	return &EdgeSearchUseCase{storeClient: storeClient}
}

func (uc *EdgeSearchUseCase) Execute(ctx context.Context, edgeType string, temporalFilter *domain.TemporalWindow) ([]domain.SearchResult, error) {
	return uc.storeClient.EdgeSearch(ctx, edgeType, temporalFilter)
}
