package port

import (
	"context"

	"openviking.com/ov-resource/internal/domain/model"
	"openviking.com/ov-resource/internal/usecase/dto"
)

type IngestUseCase interface {
	Execute(ctx context.Context, req dto.IngestRequest) (dto.IngestResponse, error)
}

type ParseUseCase interface {
	Execute(ctx context.Context, req dto.ParseRequest) (dto.ParseResponse, error)
}

type WatchUseCase interface {
	Execute(ctx context.Context, req dto.WatchRequest) (<-chan model.WatchEvent, error)
	StartManager(ctx context.Context) error
}

type RefreshUseCase interface {
	Execute(ctx context.Context, req dto.RefreshRequest) (dto.RefreshResponse, error)
}
