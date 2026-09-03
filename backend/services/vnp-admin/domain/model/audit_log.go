// Package model defines core domain entities for vnp-admin.
package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of audited operation.
type AuditAction string

const (
	AuditActionCreate  AuditAction = "create"
	AuditActionUpdate  AuditAction = "update"
	AuditActionDelete  AuditAction = "delete"
	AuditActionForget  AuditAction = "forget"
	AuditActionLogin   AuditAction = "login"
	AuditActionExport  AuditAction = "export"
)

// AuditLog represents a single audit trail entry.
type AuditLog struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     uuid.UUID              `json:"tenant_id"`
	UserID       string                 `json:"user_id"`
	Action       AuditAction            `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Metadata     map[string]any         `json:"metadata,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AuditLogFilter defines query parameters for audit log search.
type AuditLogFilter struct {
	TenantID     uuid.UUID   `json:"tenant_id"`
	UserID       string      `json:"user_id,omitempty"`
	Action       AuditAction `json:"action,omitempty"`
	ResourceType string      `json:"resource_type,omitempty"`
	From         *time.Time  `json:"from,omitempty"`
	To           *time.Time  `json:"to,omitempty"`
	Offset       int         `json:"offset"`
	Limit        int         `json:"limit"`
}

// NewAuditLog creates a new audit log entry.
func NewAuditLog(tenantID uuid.UUID, userID string, action AuditAction, resourceType, resourceID string) *AuditLog {
	return &AuditLog{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		CreatedAt:    time.Now().UTC(),
	}
}
