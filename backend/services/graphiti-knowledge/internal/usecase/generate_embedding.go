package usecase

import (
	"context"
	"vnp-memory/services/graphiti-knowledge/domain"
	"vnp-memory/services/graphiti-knowledge/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/usecase/port"
)

type generateEmbeddingUseCase struct {
	embedder port.EmbedderClient
}

func NewGenerateEmbeddingUseCase(embedder port.EmbedderClient) port.EmbedUseCase {
	return &generateEmbeddingUseCase{embedder: embedder}
}

func (uc *generateEmbeddingUseCase) Execute(ctx context.Context, req dto.EmbedRequest) (domain.EmbeddingResult, error) {
	vec, err := uc.embedder.Embed(ctx, req.Text, req.Model)
	if err != nil {
		return domain.EmbeddingResult{}, err
	}
	
	return domain.EmbeddingResult{Vector: vec}, nil
}
