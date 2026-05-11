package usecase

import (
	"context"
	"fmt"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/port"
)

type flushBufferUseCase struct {
	bufferRepo repository.BufferZoneRepository
	publisher  port.EventPublisher
}

func NewFlushBufferUseCase(
	bufferRepo repository.BufferZoneRepository,
	publisher port.EventPublisher,
) port.BufferFlusher {
	return &flushBufferUseCase{
		bufferRepo: bufferRepo,
		publisher:  publisher,
	}
}

func (uc *flushBufferUseCase) FlushBuffer(ctx context.Context, req *dto.FlushBufferRequest) (*dto.FlushResponse, error) {
	// For simplicity, we trigger flush for each BlobType separately or together.
	// We'll iterate over the 3 types. In a real system, you might filter by type.
	
	blobTypes := []model.BlobType{model.BlobTypeChat, model.BlobTypeDoc, model.BlobTypeSummary}
	totalFlushed := 0

	for _, bt := range blobTypes {
		// 1. Optimistic concurrency update
		updatedIDs, err := uc.bufferRepo.UpdateStatusForIdle(ctx, req.ProjectID, req.UserID, bt, model.BufferStatusProcessing)
		if err != nil {
			continue // Log error in real implementation, but proceed
		}

		if len(updatedIDs) == 0 {
			continue
		}

		// 2. Publish event
		event := &model.BufferReadyEvent{
			UserID:    req.UserID,
			ProjectID: req.ProjectID,
			BufferIDs: updatedIDs,
			BlobType:  bt,
		}

		if err := uc.publisher.PublishBufferReady(ctx, event); err != nil {
			// If publish fails, we should ideally mark them back as FAILED or retry.
			// Simplified handling here.
			for _, id := range updatedIDs {
				_ = uc.bufferRepo.UpdateStatus(ctx, req.ProjectID, id, model.BufferStatusFailed)
			}
			return nil, fmt.Errorf("failed to publish buffer ready event: %w", err)
		}

		totalFlushed += len(updatedIDs)
	}

	return &dto.FlushResponse{
		FlushedCount: totalFlushed,
	}, nil
}
