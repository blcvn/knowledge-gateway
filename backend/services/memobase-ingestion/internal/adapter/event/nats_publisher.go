package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/usecase/port"
)

type natsPublisher struct {
	js nats.JetStreamContext
}

func NewNatsPublisher(js nats.JetStreamContext) port.EventPublisher {
	return &natsPublisher{js: js}
}

func (p *natsPublisher) PublishBufferReady(ctx context.Context, event *model.BufferReadyEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal buffer ready event: %w", err)
	}

	subject := "memobase.buffer.ready"
	_, err = p.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish to NATS subject %s: %w", subject, err)
	}
	return nil
}
