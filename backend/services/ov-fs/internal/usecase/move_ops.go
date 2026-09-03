package usecase

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/repository"
	"vnp-memory/services/ov-fs/internal/usecase/dto"
	"vnp-memory/services/ov-fs/internal/usecase/port"
)

type moveUseCase struct {
	fileRepo repository.FileRepository
}

func NewMoveUseCase(fileRepo repository.FileRepository) port.MoveUseCase {
	return &moveUseCase{fileRepo: fileRepo}
}

func (uc *moveUseCase) Move(ctx context.Context, req dto.MoveRequest) error {
	// The mv lock logic is expected to be handled within the fileRepo or via an interceptor/lock manager 
	// wrapped around the usecase/repo.
	return uc.fileRepo.Move(ctx, req.AccountID, req.Source, req.Destination, req.Overwrite)
}
