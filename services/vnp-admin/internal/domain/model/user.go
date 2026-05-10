package model

import (
	"time"
	"github.com/google/uuid"
)

// UserRole defines RBAC roles within a tenant.
type UserRole string

const (
	UserRoleOwner  UserRole = "owner"
	UserRoleAdmin  UserRole = "admin"
	UserRoleMember UserRole = "member"
	UserRoleViewer UserRole = "viewer"
)

// User represents a user within a tenant.
type User struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Role      UserRole       `json:"role"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Active    bool           `json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// BillingEntry tracks resource usage for a tenant.
type BillingEntry struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Period    string    `json:"period"`     // "2026-05"
	APICallCount int64  `json:"api_call_count"`
	StorageUsedMB int64 `json:"storage_used_mb"`
	LLMTokensUsed int64 `json:"llm_tokens_used"`
	CreatedAt time.Time `json:"created_at"`
}
