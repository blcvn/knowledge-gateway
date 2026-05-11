package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

type NatsPublisher struct {
	js jetstream.JetStream
}

func NewNatsPublisher(js jetstream.JetStream) *NatsPublisher {
	return &NatsPublisher{js: js}
}

func (p *NatsPublisher) Publish(ctx context.Context, event domain.DomainEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := event.EventName()

	var pubErr error
	// Retry 3 times with backoff on transient failures
	for i := 0; i < 3; i++ {
		_, pubErr = p.js.Publish(ctx, subject, data)
		if pubErr == nil {
			return nil
		}
		// Basic backoff
		time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
	}

	return fmt.Errorf("failed to publish event after retries: %w", pubErr)
}
