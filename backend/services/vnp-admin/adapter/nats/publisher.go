// Package nats implements the NATS event publisher adapter.
package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/model"
)

// Publisher implements port.EventPublisherPort.
type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(conn *nats.Conn) *Publisher {
	return &Publisher{conn: conn}
}

func (p *Publisher) PublishTenantCreated(ctx context.Context, tenantID uuid.UUID) error {
	return p.publish(model.EventTenantCreated, tenantID, tenantID)
}

func (p *Publisher) PublishTenantDeleted(ctx context.Context, tenantID uuid.UUID) error {
	return p.publish(model.EventTenantDeleted, tenantID, tenantID)
}

func (p *Publisher) PublishKeyRevoked(ctx context.Context, tenantID, keyID uuid.UUID) error {
	return p.publish(model.EventKeyRevoked, tenantID, keyID)
}

func (p *Publisher) publish(eventType string, tenantID, entityID uuid.UUID) error {
	evt := model.DomainEvent{
		Type:     eventType,
		TenantID: tenantID,
		EntityID: entityID,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.conn.Publish(eventType, data)
}
