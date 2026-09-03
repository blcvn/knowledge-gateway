package usecase

import (
	"context"

	"vnp-memory/services/graphiti-search/internal/domain"
)

type NodeSearchUseCase struct {
	storeClient StoreSearchClient
}

func NewNodeSearchUseCase(storeClient StoreSearchClient) *NodeSearchUseCase {
	return &NodeSearchUseCase{storeClient: storeClient}
}

func (uc *NodeSearchUseCase) Execute(ctx context.Context, query domain.SearchQuery) ([]domain.SearchResult, error) {
	return uc.storeClient.NodeSearch(ctx, query.EntityLabels, query.TemporalFilter)
}
