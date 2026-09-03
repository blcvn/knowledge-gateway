package event

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/domain/model"
	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type natsPublisher struct {
	// natsConn *nats.Conn
}

// NewNatsPublisher creates a new NATS JetStream publisher.
func NewNatsPublisher() port.EventPublisher {
	return &natsPublisher{}
}

func (p *natsPublisher) PublishEngineCompleted(ctx context.Context, result model.PipelineResult) error {
	return nil
}

func (p *natsPublisher) PublishProfileChanged(ctx context.Context, userID, projectID string) error {
	return nil
}

func (p *natsPublisher) PublishEventCreated(ctx context.Context, event model.UserEvent) error {
	return nil
}
