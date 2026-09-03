package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	nc *nats.Conn
}

func NewNATSPublisher(nc *nats.Conn) *NATSPublisher {
	return &NATSPublisher{nc: nc}
}

type PipelineEvent struct {
	TenantID uuid.UUID `json:"tenant_id"`
	UserID   uuid.UUID `json:"user_id"`
	Event    string    `json:"event"`
}

func (p *NATSPublisher) PublishBlobIngested(ctx context.Context, tenantID, userID uuid.UUID) error {
	return p.publish("memobase.blob.ingested", tenantID, userID)
}

func (p *NATSPublisher) PublishFlushCompleted(ctx context.Context, tenantID, userID uuid.UUID) error {
	return p.publish("memobase.pipeline.completed", tenantID, userID)
}

func (p *NATSPublisher) publish(subject string, tenantID, userID uuid.UUID) error {
	evt := PipelineEvent{
		TenantID: tenantID,
		UserID:   userID,
		Event:    subject,
	}
	
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	
	if err := p.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("publish to nats: %w", err)
	}
	return nil
}
