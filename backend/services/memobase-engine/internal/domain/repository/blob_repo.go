package repository

import (
	"context"

	"github.com/google/uuid"
	"vnp-memory/services/memobase-engine/internal/domain/model"
)

// BlobRepository defines the interface for reading blobs.
type BlobRepository interface {
	// GetBlobs retrieves a list of blobs by their IDs.
	GetBlobs(ctx context.Context, ids []uuid.UUID) ([]model.Blob, error)
}
