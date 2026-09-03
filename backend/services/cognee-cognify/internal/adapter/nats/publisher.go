package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type eventPublisher struct {
	nc *nats.Conn
}

func NewEventPublisher(nc *nats.Conn) port.EventPublisher {
	return &eventPublisher{nc: nc}
}

func (p *eventPublisher) PublishPipelineCompleted(ctx context.Context, event domain.PipelineCompletedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	// cognee.pipeline.completed
	return p.nc.Publish("cognee.pipeline.completed", data)
}

func (p *eventPublisher) PublishStageAdvanced(ctx context.Context, event domain.StageAdvancedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.nc.Publish("cognee.pipeline.stage_advanced", data)
}

// PublishPipelineEvent publishes a generic pipeline lifecycle event to NATS.
// [NEW] CE-006: used by MemifyUseCase for started/completed/failed events.
func (p *eventPublisher) PublishPipelineEvent(ctx context.Context, subject string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal pipeline event: %w", err)
	}
	return p.nc.Publish(subject, data)
}
