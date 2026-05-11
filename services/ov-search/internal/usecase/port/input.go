package port

import (
	"context"

	"vnp-memory/ov-search/internal/usecase/dto"
)

type SearchUseCase interface {
	Search(ctx context.Context, req dto.SearchRequest) (*dto.SearchResponse, error)
	RetrieveContext(ctx context.Context, req dto.ContextRequest) (*dto.ContextResponse, error)
}

type EmbeddingUseCase interface {
	Upsert(ctx context.Context, req dto.UpsertRequest) error
	Delete(ctx context.Context, req dto.DeleteRequest) error
}

type HotnessUseCase interface {
	Get(ctx context.Context, accountID string, paths []string) (map[string]float64, error)
	BoostSession(ctx context.Context, accountID string, paths []string) error
}
