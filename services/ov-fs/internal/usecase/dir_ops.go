package usecase

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/repository"
	"vnp-memory/services/ov-fs/internal/usecase/dto"
	"vnp-memory/services/ov-fs/internal/usecase/port"
	"vnp-memory/services/ov-fs/internal/domain/model"
)

type dirUseCase struct {
	fileRepo repository.FileRepository
}

func NewDirUseCase(fileRepo repository.FileRepository) port.DirUseCase {
	return &dirUseCase{fileRepo: fileRepo}
}

func (uc *dirUseCase) MkDir(ctx context.Context, req dto.MkDirRequest) error {
	return uc.fileRepo.MkDir(ctx, req.AccountID, req.Path, req.CreateParents)
}

func (uc *dirUseCase) ListDir(ctx context.Context, req dto.ListDirRequest) (*dto.ListDirResponse, error) {
	entries, err := uc.fileRepo.ListDir(ctx, req.AccountID, req.Path, req.Recursive, req.IncludeMetadata)
	if err != nil {
		return nil, err
	}
	return &dto.ListDirResponse{Entries: entries}, nil
}

func (uc *dirUseCase) Tree(ctx context.Context, req dto.TreeRequest) (*dto.TreeResponse, error) {
	root, err := uc.fileRepo.Tree(ctx, req.AccountID, req.Root, model.TreeOptions{
		MaxDepth:         req.MaxDepth,
		IncludeAbstracts: req.IncludeAbstracts,
	})
	if err != nil {
		return nil, err
	}
	return &dto.TreeResponse{Root: root}, nil
}
