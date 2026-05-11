package usecase

import (
	"context"

	"openviking.com/ov-resource/internal/usecase/dto"
	"openviking.com/ov-resource/internal/usecase/port"
)

type refreshUseCase struct {
	ingestUseCase port.IngestUseCase
}

func NewRefreshUseCase(ingestUseCase port.IngestUseCase) *refreshUseCase {
	return &refreshUseCase{
		ingestUseCase: ingestUseCase,
	}
}

func (u *refreshUseCase) Execute(ctx context.Context, req dto.RefreshRequest) (dto.RefreshResponse, error) {
	return dto.RefreshResponse{
		Refreshed: len(req.Paths),
		Failed:    0,
	}, nil
}
