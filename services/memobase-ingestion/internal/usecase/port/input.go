package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/dto"
)

type BlobInserter interface {
	InsertBlob(ctx context.Context, req *dto.InsertBlobRequest) (*dto.InsertBlobResponse, error)
}

type BufferFlusher interface {
	FlushBuffer(ctx context.Context, req *dto.FlushBufferRequest) (*dto.FlushResponse, error)
}

type BufferStatusGetter interface {
	GetBufferStatus(ctx context.Context, req *dto.BufferStatusRequest) (*dto.BufferStatus, error)
}

type BlobDeleter interface {
	DeleteBlob(ctx context.Context, req *dto.DeleteBlobRequest) error
}
