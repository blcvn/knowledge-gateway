package event

import (
	"context"
	"encoding/json"
	// "github.com/nats-io/nats.go"

	"vnp-memory/services/ov-fs/internal/domain"
	"vnp-memory/services/ov-fs/internal/usecase/port"
)

type natsPublisher struct {
	// nc *nats.Conn
	// js nats.JetStreamContext
}

// Ensure interface is implemented
var _ port.EventPublisherPort = (*natsPublisher)(nil)

func NewNatsPublisher() port.EventPublisherPort {
	return &natsPublisher{}
}

func (p *natsPublisher) PublishContentWritten(ctx context.Context, event domain.ContentWritten) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	// Simulate NATS publish
	_ = payload
	// _, err = p.js.Publish("ov.content.written", payload)
	return nil
}

func (p *natsPublisher) PublishContentDeleted(ctx context.Context, event domain.ContentDeleted) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	// Simulate NATS publish
	_ = payload
	// _, err = p.js.Publish("ov.content.deleted", payload)
	return nil
}
