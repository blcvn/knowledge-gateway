package model

import (
	"github.com/google/uuid"
)

// DomainEvent represents a domain event published to NATS.
type DomainEvent struct {
	Type     string    `json:"type"`
	TenantID uuid.UUID `json:"tenant_id"`
	EntityID uuid.UUID `json:"entity_id"`
}

// Event types published by vnp-admin.
const (
	EventTenantCreated = "admin.tenant.created"
	EventTenantDeleted = "admin.tenant.deleted"
	EventUserDeleted   = "admin.user.deleted"
	EventKeyRevoked    = "admin.apikey.revoked"
)
