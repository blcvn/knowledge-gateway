package usecase

import (
	"context"
	"fmt"

	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

type GetStatusUseCase struct {
	sagaRepo SagaStateRepo
}

func NewGetStatusUseCase(repo SagaStateRepo) *GetStatusUseCase {
	return &GetStatusUseCase{sagaRepo: repo}
}

func (uc *GetStatusUseCase) Execute(ctx context.Context, sagaID string) (*domain.SagaState, error) {
	state, err := uc.sagaRepo.Get(ctx, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saga state: %w", err)
	}
	return state, nil
}
