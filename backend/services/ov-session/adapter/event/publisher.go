package event

import (
	"context"

	"github.com/vnp-memory/services/ov-session/domain"
)

type Publisher interface {
	PublishSessionCommitted(ctx context.Context, evt *domain.SessionCommitted) error
	PublishMemoryExtracted(ctx context.Context, evt *domain.MemoryExtracted) error
}

type publisherImpl struct{}

func NewPublisher() Publisher {
	return &publisherImpl{}
}

func (p *publisherImpl) PublishSessionCommitted(ctx context.Context, evt *domain.SessionCommitted) error {
	// NATS publish ov.session.committed
	return nil
}

func (p *publisherImpl) PublishMemoryExtracted(ctx context.Context, evt *domain.MemoryExtracted) error {
	// NATS publish ov.session.memory.extracted
	return nil
}
