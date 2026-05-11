package repository

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/model"
)

// BlobRepository defines operations for the general blobs
type BlobRepository interface {
	Insert(ctx context.Context, blob *model.GeneralBlob) error
	FindByID(ctx context.Context, projectID, blobID string) (*model.GeneralBlob, error)
	Delete(ctx context.Context, projectID, blobID string) error
}
