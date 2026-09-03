package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	"vnp-memory/services/cognee-ingestion/internal/usecase"
)

const subjectDataIngested = "cognee.ingestion.data.ingested"

// Publisher publishes ingestion events to NATS.
type Publisher struct {
	nc     *nats.Conn
	logger *slog.Logger
}

// NewPublisher creates a new NATS event publisher.
func NewPublisher(nc *nats.Conn, logger *slog.Logger) *Publisher {
	return &Publisher{nc: nc, logger: logger}
}

// PublishDataIngested publishes a DataIngestedEvent to NATS.
// node_sets is included so downstream cognify service can attach labels.
func (p *Publisher) PublishDataIngested(ctx context.Context, evt usecase.DataIngestedEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		p.logger.Error("failed to marshal DataIngestedEvent", "error", err)
		return
	}
	if err := p.nc.Publish(subjectDataIngested, data); err != nil {
		p.logger.Error("failed to publish DataIngestedEvent", "error", err)
	}
}
