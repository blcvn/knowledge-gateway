package usecase

import (
	"context"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/port"
)

type rerankUseCase struct {
	// cross-encoder client port
}

func NewRerankUseCase() port.RerankUseCase {
	return &rerankUseCase{}
}

func (uc *rerankUseCase) Execute(ctx context.Context, req domain.RerankRequest) (domain.RerankResult, error) {
	// Neural reranking implementation
	return domain.RerankResult{}, nil
}
