package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkoukk/tiktoken-go"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/usecase/port"
)

const flushThreshold = 1024

type insertBlobUseCase struct {
	blobRepo   repository.BlobRepository
	bufferRepo repository.BufferZoneRepository
	flusher    port.BufferFlusher
	encoder    *tiktoken.Tiktoken
}

func NewInsertBlobUseCase(
	blobRepo repository.BlobRepository,
	bufferRepo repository.BufferZoneRepository,
	flusher port.BufferFlusher,
) (port.BlobInserter, error) {
	enc, err := tiktoken.EncodingForModel("gpt-4o")
	if err != nil {
		return nil, fmt.Errorf("failed to load tiktoken encoder: %w", err)
	}
	return &insertBlobUseCase{
		blobRepo:   blobRepo,
		bufferRepo: bufferRepo,
		flusher:    flusher,
		encoder:    enc,
	}, nil
}

func (uc *insertBlobUseCase) InsertBlob(ctx context.Context, req *dto.InsertBlobRequest) (*dto.InsertBlobResponse, error) {
	blobID := uuid.New().String()
	now := time.Now()

	// 1. Store blob
	blob := &model.GeneralBlob{
		ID:        blobID,
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		BlobType:  req.BlobType,
		BlobData:  req.BlobData,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.blobRepo.Insert(ctx, blob); err != nil {
		return nil, fmt.Errorf("failed to insert blob: %w", err)
	}

	// 2. Calculate token size
	tokenSize := len(uc.encoder.Encode(string(req.BlobData), nil, nil))

	// 3. Create buffer entry
	bufferEntry := &model.BufferZone{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		BlobID:    blobID,
		BlobType:  req.BlobType,
		TokenSize: tokenSize,
		Status:    model.BufferStatusIdle,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.bufferRepo.Insert(ctx, bufferEntry); err != nil {
		return nil, fmt.Errorf("failed to insert buffer entry: %w", err)
	}

	// 4. Check threshold
	totalTokens, err := uc.bufferRepo.GetTotalIdleTokens(ctx, req.ProjectID, req.UserID, req.BlobType)
	if err != nil {
		return nil, fmt.Errorf("failed to get total idle tokens: %w", err)
	}

	flushed := false
	if totalTokens >= flushThreshold {
		// Fire and forget flush
		go func(pID, uID string) {
			// create a new background context to prevent cancellation
			bgCtx := context.Background()
			_, _ = uc.flusher.FlushBuffer(bgCtx, &dto.FlushBufferRequest{
				UserID:    uID,
				ProjectID: pID,
			})
		}(req.ProjectID, req.UserID)
		flushed = true
	}

	return &dto.InsertBlobResponse{
		BlobID:           blobID,
		BufferFlushed:    flushed,
		BufferTokenCount: totalTokens,
	}, nil
}
