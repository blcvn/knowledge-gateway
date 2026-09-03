// Package event implements the EventPublisher port via NATS JetStream.
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"vnp-memory/services/cognee-ingestion/internal/domain"
)

const (
	// SubjectDataIngested is the NATS subject for ingested data events.
	SubjectDataIngested = "cognee.data.ingested"
)

// NATSPublisher publishes domain events to NATS JetStream.
type NATSPublisher struct {
	js     nats.JetStreamContext
	logger *slog.Logger
}

// NewNATSPublisher creates a publisher with the given JetStream context.
func NewNATSPublisher(js nats.JetStreamContext, logger *slog.Logger) *NATSPublisher {
	return &NATSPublisher{
		js:     js,
		logger: logger.With("adapter", "nats_publisher"),
	}
}

// PublishDataIngested publishes a DataIngestedEvent to NATS JetStream.
func (p *NATSPublisher) PublishDataIngested(ctx context.Context, event domain.DataIngestedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	ack, err := p.js.Publish(SubjectDataIngested, data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", SubjectDataIngested, err)
	}

	p.logger.Info("event published",
		"subject", SubjectDataIngested,
		"dataset_id", event.DatasetID,
		"item_count", len(event.ItemIDs),
		"stream", ack.Stream,
		"sequence", ack.Sequence,
	)
	return nil
}
