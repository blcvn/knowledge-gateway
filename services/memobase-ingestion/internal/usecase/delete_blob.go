package usecase

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/port"
)

type deleteBlobUseCase struct {
	blobRepo   repository.BlobRepository
	bufferRepo repository.BufferZoneRepository
}

func NewDeleteBlobUseCase(
	blobRepo repository.BlobRepository,
	bufferRepo repository.BufferZoneRepository,
) port.BlobDeleter {
	return &deleteBlobUseCase{
		blobRepo:   blobRepo,
		bufferRepo: bufferRepo,
	}
}

func (uc *deleteBlobUseCase) DeleteBlob(ctx context.Context, req *dto.DeleteBlobRequest) error {
	// First delete buffer reference if exists to avoid foreign key constraints violations
	_ = uc.bufferRepo.DeleteByBlobID(ctx, req.ProjectID, req.BlobID)
	
	// Then delete the actual blob
	return uc.blobRepo.Delete(ctx, req.ProjectID, req.BlobID)
}
