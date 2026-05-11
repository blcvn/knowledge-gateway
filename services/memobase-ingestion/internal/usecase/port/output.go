package port

import (
	"context"
	
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/model"
)

type EventPublisher interface {
	PublishBufferReady(ctx context.Context, event *model.BufferReadyEvent) error
}
