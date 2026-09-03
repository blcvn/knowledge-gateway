// Package ingestion implements the buffer zone FSM + YOLO merge pipeline.
package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/engine"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/ingestion"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/port"
)

// Service implements port.IngestionUseCase.
type Service struct {
	blobs   port.BlobRepository
	buffers port.BufferRepository
	engine  port.EngineUseCase
	pub     port.EventPublisher
}

func NewService(blobs port.BlobRepository, buffers port.BufferRepository, eng port.EngineUseCase, pub port.EventPublisher) *Service {
	return &Service{blobs: blobs, buffers: buffers, engine: eng, pub: pub}
}

func (s *Service) IngestBlob(ctx context.Context, tenantID, userID uuid.UUID, content, blobType string, tokens int) (*ingestion.Blob, error) {
	blob := &ingestion.Blob{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		Content: content, Type: blobType, Tokens: tokens, CreatedAt: time.Now(),
	}
	if err := s.blobs.Create(ctx, blob); err != nil {
		return nil, fmt.Errorf("create blob: %w", err)
	}

	// Update buffer zone FSM
	buf, err := s.buffers.FindOrCreate(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("find buffer: %w", err)
	}

	buf.BlobIDs = append(buf.BlobIDs, blob.ID)
	buf.TokenCount += tokens
	buf.State = ingestion.BufferAccumulating

	// Check if buffer is ready to flush
	if buf.TokenCount >= buf.Threshold {
		buf.State = ingestion.BufferReady
	}
	if err := s.buffers.Update(ctx, buf); err != nil {
		return nil, fmt.Errorf("update buffer: %w", err)
	}

	// Auto-flush if ready
	if buf.State == ingestion.BufferReady {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, _ = s.FlushBuffer(bgCtx, tenantID, userID)
		}()
	}

	return blob, nil
}

// FlushBuffer triggers YOLO merge — calls engine LOCALLY (no gRPC).
func (s *Service) FlushBuffer(ctx context.Context, tenantID, userID uuid.UUID) (*engine.MergeResult, error) {
	buf, err := s.buffers.FindOrCreate(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	buf.State = ingestion.BufferProcessing
	_ = s.buffers.Update(ctx, buf)

	blobs, err := s.blobs.FindByIDs(ctx, buf.BlobIDs)
	if err != nil {
		return nil, fmt.Errorf("find blobs: %w", err)
	}

	// Local YOLO merge call (was cross-service gRPC to memobase-engine)
	result, err := s.engine.YOLOMerge(ctx, tenantID, userID, blobs)
	if err != nil {
		buf.State = ingestion.BufferReady // Retry later
		_ = s.buffers.Update(ctx, buf)
		return nil, fmt.Errorf("yolo merge: %w", err)
	}

	// Reset buffer
	now := time.Now()
	buf.State = ingestion.BufferIdle
	buf.BlobIDs = nil
	buf.TokenCount = 0
	buf.LastFlushed = &now
	_ = s.buffers.Update(ctx, buf)

	_ = s.pub.PublishFlushCompleted(ctx, tenantID, userID)
	return result, nil
}

func (s *Service) GetBufferState(ctx context.Context, tenantID, userID uuid.UUID) (*ingestion.BufferZone, error) {
	return s.buffers.FindOrCreate(ctx, tenantID, userID)
}

func (s *Service) SetThreshold(ctx context.Context, tenantID, userID uuid.UUID, threshold int) error {
	buf, err := s.buffers.FindOrCreate(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	buf.Threshold = threshold
	return s.buffers.Update(ctx, buf)
}
