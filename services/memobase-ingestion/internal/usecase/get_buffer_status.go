package usecase

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/port"
)

type getBufferStatusUseCase struct {
	bufferRepo repository.BufferZoneRepository
}

func NewGetBufferStatusUseCase(bufferRepo repository.BufferZoneRepository) port.BufferStatusGetter {
	return &getBufferStatusUseCase{
		bufferRepo: bufferRepo,
	}
}

func (uc *getBufferStatusUseCase) GetBufferStatus(ctx context.Context, req *dto.BufferStatusRequest) (*dto.BufferStatus, error) {
	agg, err := uc.bufferRepo.GetStatusAggregation(ctx, req.ProjectID, req.UserID)
	if err != nil {
		return nil, err
	}

	return &dto.BufferStatus{
		UserID:          req.UserID,
		ProjectID:       req.ProjectID,
		IdleCount:       agg.IdleCount,
		ProcessingCount: agg.ProcessingCount,
		FailedCount:     agg.FailedCount,
		TotalTokens:     agg.TotalTokens,
		Threshold:       1024,
	}, nil
}
