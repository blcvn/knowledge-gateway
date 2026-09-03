package usecase

import (
	"context"

	"vnp-memory/services/graphiti-search/internal/domain"
)

type CommunitySearchUseCase struct {
	storeClient StoreSearchClient
}

func NewCommunitySearchUseCase(storeClient StoreSearchClient) *CommunitySearchUseCase {
	return &CommunitySearchUseCase{storeClient: storeClient}
}

func (uc *CommunitySearchUseCase) Execute(ctx context.Context, query string) ([]domain.SearchResult, error) {
	return uc.storeClient.CommunitySearch(ctx, query)
}
