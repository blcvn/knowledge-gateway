package port

import (
	"context"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
)

type ExtractEntitiesUseCase interface {
	Execute(ctx context.Context, req dto.ExtractEntitiesRequest) ([]domain.ExtractedEntity, domain.TokenUsage, error)
}

type ResolveEntitiesUseCase interface {
	Execute(ctx context.Context, req dto.ResolveEntitiesRequest) ([]domain.Resolution, error)
}

type ExtractEdgesUseCase interface {
	Execute(ctx context.Context, req dto.ExtractEdgesRequest) ([]domain.ExtractedEdge, domain.TokenUsage, error)
}

type ResolveEdgesUseCase interface {
	Execute(ctx context.Context, req dto.ResolveEdgesRequest) (dto.ResolveEdgesResponse, error)
}

type EmbedUseCase interface {
	Execute(ctx context.Context, req dto.EmbedRequest) (domain.EmbeddingResult, error)
}

type UpdateCommunityUseCase interface {
	Execute(ctx context.Context, req dto.UpdateCommunityRequest) (dto.UpdateCommunityResponse, error)
}

type RerankUseCase interface {
	Execute(ctx context.Context, req domain.RerankRequest) (domain.RerankResult, error)
}
