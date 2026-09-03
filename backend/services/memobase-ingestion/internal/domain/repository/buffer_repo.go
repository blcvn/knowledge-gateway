package repository

import (
	"context"
	"time"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/domain/model"
)

// BufferStatusAggregation holds metrics for buffer states
type BufferStatusAggregation struct {
	IdleCount       int `json:"idle_count"`
	ProcessingCount int `json:"processing_count"`
	FailedCount     int `json:"failed_count"`
	TotalTokens     int `json:"total_tokens"`
}

// BufferZoneRepository defines operations for buffer zones and FSM transitions
type BufferZoneRepository interface {
	Insert(ctx context.Context, buffer *model.BufferZone) error
	GetTotalIdleTokens(ctx context.Context, projectID, userID string, blobType model.BlobType) (int, error)
	GetStatusAggregation(ctx context.Context, projectID, userID string) (*BufferStatusAggregation, error)
	
	// FSM Transitions
	UpdateStatusForIdle(ctx context.Context, projectID, userID string, blobType model.BlobType, targetStatus model.BufferStatus) ([]string, error)
	GetIdleSince(ctx context.Context, since time.Time) ([]*model.BufferZone, error)
	UpdateStatus(ctx context.Context, projectID, bufferID string, status model.BufferStatus) error
	
	DeleteByBlobID(ctx context.Context, projectID, blobID string) error
}
