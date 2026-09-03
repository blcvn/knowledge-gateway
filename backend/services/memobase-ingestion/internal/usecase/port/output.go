package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-memory/services/memobase-ingestion/internal/domain"
)

// BlobRepository persists raw blobs to PostgreSQL.
type BlobRepository interface {
	Save(ctx context.Context, blob *domain.Blob) (*domain.Blob, error)
	GetByID(ctx context.Context, blobID uuid.UUID, projectID string) (*domain.Blob, error)
	Delete(ctx context.Context, blobID uuid.UUID, projectID string) error
	DeleteByUser(ctx context.Context, userID uuid.UUID, projectID string) error
	GetForProcessing(ctx context.Context, bufferIDs []uuid.UUID, projectID string) ([]*domain.Blob, error)
	DeleteBatch(ctx context.Context, blobIDs []uuid.UUID, projectID string) error
}

// BufferRepository manages the buffer_zones queue.
type BufferRepository interface {
	Save(ctx context.Context, entry *domain.BufferZone) (*domain.BufferZone, error)

	// AcquireProcessingLock atomically transitions idle → processing.
	// Returns empty slice (not error) if another flush is already running.
	AcquireProcessingLock(ctx context.Context, userID uuid.UUID, projectID string, blobType domain.BlobType) ([]*domain.BufferZone, error)

	GetTotalIdleTokens(ctx context.Context, userID uuid.UUID, projectID string) (int, error)
	GetBufferCapacity(ctx context.Context, userID uuid.UUID, projectID string, blobType domain.BlobType) (*CapacityInfo, error)
	MarkDone(ctx context.Context, bufferIDs []uuid.UUID, projectID string) error
	MarkFailed(ctx context.Context, bufferIDs []uuid.UUID, projectID string, errMsg string) error
	GetUsersWithStaleIdleBuffers(ctx context.Context, idleTimeout time.Duration) ([]*UserProject, error)
}

// CapacityInfo reports current buffer usage for a user/project/type combination.
type CapacityInfo struct {
	NumBlobs  int
	NumTokens int
}

// UserProject identifies a user within a project.
type UserProject struct {
	UserID    uuid.UUID
	ProjectID string
}

// EventPublisher sends NATS JetStream events.
type EventPublisher interface {
	PublishBufferReady(ctx context.Context, payload BufferReadyPayload) error
}

// BufferReadyPayload is the NATS event payload for memobase.buffer.ready.
type BufferReadyPayload struct {
	UserID    string   `json:"user_id"`
	ProjectID string   `json:"project_id"`
	BufferIDs []string `json:"buffer_ids"`
	BlobType  string   `json:"blob_type"`
}
