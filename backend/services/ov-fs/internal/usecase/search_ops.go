package usecase

import (
	"context"

	"vnp-memory/services/ov-fs/internal/domain/repository"
	"vnp-memory/services/ov-fs/internal/usecase/dto"
	"vnp-memory/services/ov-fs/internal/usecase/port"
)

type searchUseCase struct {
	fileRepo repository.FileRepository
}

func NewSearchUseCase(fileRepo repository.FileRepository) port.SearchUseCase {
	return &searchUseCase{fileRepo: fileRepo}
}

func (uc *searchUseCase) Grep(ctx context.Context, req dto.GrepRequest) (*dto.GrepResponse, error) {
	matches, err := uc.fileRepo.Grep(ctx, req.AccountID, req.Pattern, req.Path, req.CaseInsensitive, req.MaxResults)
	if err != nil {
		return nil, err
	}

	var dtoMatches []*dto.GrepMatch
	for _, m := range matches {
		dtoMatches = append(dtoMatches, &dto.GrepMatch{
			Path:       m.Path,
			LineNumber: m.LineNumber,
			Content:    m.Content,
			Score:      m.Score,
		})
	}

	return &dto.GrepResponse{Matches: dtoMatches}, nil
}

func (uc *searchUseCase) Glob(ctx context.Context, req dto.GlobRequest) (*dto.GlobResponse, error) {
	paths, err := uc.fileRepo.Glob(ctx, req.AccountID, req.Pattern, req.Root)
	if err != nil {
		return nil, err
	}
	return &dto.GlobResponse{Paths: paths}, nil
}
