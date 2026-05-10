// Package nats implements the NATS event publisher for vnp-platform.
package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
)

// Publisher implements port.EventPublisher.
type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(conn *nats.Conn) *Publisher {
	return &Publisher{conn: conn}
}

type tenantEvent struct {
	Type     string    `json:"type"`
	TenantID string    `json:"tenant_id"`
	Name     string    `json:"name,omitempty"`
	Tier     string    `json:"tier,omitempty"`
}

func (p *Publisher) PublishTenantCreated(ctx context.Context, tenant *admin.Tenant) error {
	evt := tenantEvent{
		Type:     "admin.tenant.created",
		TenantID: tenant.ID.String(),
		Name:     tenant.Name,
		Tier:     string(tenant.Tier),
	}
	return p.publish("admin.tenant.created", evt)
}

func (p *Publisher) PublishTenantDeleted(ctx context.Context, tenantID uuid.UUID) error {
	evt := tenantEvent{
		Type:     "admin.tenant.deleted",
		TenantID: tenantID.String(),
	}
	return p.publish("admin.tenant.deleted", evt)
}

func (p *Publisher) publish(subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.conn.Publish(subject, data)
}
