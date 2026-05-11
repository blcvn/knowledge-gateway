package port

import (
	"context"

	"openviking.com/ov-resource/internal/domain"
	"openviking.com/ov-resource/internal/domain/model"
)

type FileWriterPort interface {
	WriteChunks(ctx context.Context, path, accountID string, chunks []model.Chunk) error
}

type ParserPort interface {
	Parse(ctx context.Context, content []byte, filename string, config model.ParserConfig) ([]model.Chunk, error)
	Supports(filename string) bool
}

type EventPublisherPort interface {
	PublishResourceIngested(ctx context.Context, event domain.ResourceIngested) error
}
