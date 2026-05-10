// Package port defines input/output port interfaces for memobase-pipeline.
//
// Consolidated from: memobase-ingestion + memobase-engine
// Key optimization: Buffer flush triggers YOLO merge via local call
// (eliminates 10-15ms gRPC overhead per merge).
package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/engine"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/ingestion"
)

// --- Input Ports ---

// IngestionUseCase handles blob ingestion and buffer zone FSM.
type IngestionUseCase interface {
	IngestBlob(ctx context.Context, tenantID, userID uuid.UUID, content, blobType string, tokens int) (*ingestion.Blob, error)
	GetBufferState(ctx context.Context, tenantID, userID uuid.UUID) (*ingestion.BufferZone, error)
	FlushBuffer(ctx context.Context, tenantID, userID uuid.UUID) (*engine.MergeResult, error) // Triggers YOLO merge locally
	SetThreshold(ctx context.Context, tenantID, userID uuid.UUID, threshold int) error
}

// EngineUseCase handles YOLO merge and profile management.
type EngineUseCase interface {
	// YOLOMerge performs the 3-LLM-call merge operation.
	// Must complete in exactly 3 LLM calls (YOLO constraint).
	YOLOMerge(ctx context.Context, tenantID, userID uuid.UUID, blobs []ingestion.Blob) (*engine.MergeResult, error)

	GetProfile(ctx context.Context, tenantID, userID uuid.UUID) (*engine.Profile, error)
	GetEventGists(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]engine.EventGist, error)
}

// --- Output Ports ---

// BlobRepository persists raw conversation blobs.
type BlobRepository interface {
	Create(ctx context.Context, blob *ingestion.Blob) error
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]ingestion.Blob, error)
	DeleteByIDs(ctx context.Context, ids []uuid.UUID) error
}

// BufferRepository persists buffer zone state (FSM).
type BufferRepository interface {
	FindOrCreate(ctx context.Context, tenantID, userID uuid.UUID) (*ingestion.BufferZone, error)
	Update(ctx context.Context, buf *ingestion.BufferZone) error
}

// ProfileRepository persists user profiles.
type ProfileRepository interface {
	FindByUser(ctx context.Context, tenantID, userID uuid.UUID) (*engine.Profile, error)
	Upsert(ctx context.Context, profile *engine.Profile) error
}

// GistRepository persists event gists.
type GistRepository interface {
	Create(ctx context.Context, gist *engine.EventGist) error
	FindByUser(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]engine.EventGist, error)
}

// LLMService abstracts the LLM for YOLO merge operations.
type LLMService interface {
	ExtractTopics(ctx context.Context, content string) ([]engine.TopicEntry, error)
	GenerateGist(ctx context.Context, content string) (*engine.EventGist, error)
	MergeTraits(ctx context.Context, existing map[string]any, newContent string) (map[string]any, error)
}

// EventPublisher publishes pipeline events to NATS.
type EventPublisher interface {
	PublishBlobIngested(ctx context.Context, tenantID, userID uuid.UUID) error
	PublishFlushCompleted(ctx context.Context, tenantID, userID uuid.UUID) error
}
