// Package admin defines domain entities for the admin sub-domain
// of the vnp-platform consolidated service.
//
// Absorbed from: vnp-admin, ov-admin, zep-admin
package admin

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a top-level organizational unit.
// Maps to: vnp-admin.Tenant, ov-admin.Account, zep-admin.Project
type Tenant struct {
	ID             uuid.UUID         `json:"id"`
	Name           string            `json:"name"`
	Slug           string            `json:"slug"`
	Tier           SubscriptionTier  `json:"tier"`
	Status         TenantStatus      `json:"status"`
	Metadata       map[string]any    `json:"metadata,omitempty"`
	EngineAliases  map[string]string `json:"engine_aliases,omitempty"` // engine → engine-specific key
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// SubscriptionTier defines resource quota levels.
type SubscriptionTier string

const (
	TierFree       SubscriptionTier = "free"
	TierPro        SubscriptionTier = "pro"
	TierEnterprise SubscriptionTier = "enterprise"
)

// TenantStatus defines tenant lifecycle states.
type TenantStatus string

const (
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
	TenantDeleted   TenantStatus = "deleted"
)

// User represents an authenticated user within a tenant.
type User struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Role      UserRole       `json:"role"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// UserRole defines RBAC roles.
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleEditor UserRole = "editor"
	RoleViewer UserRole = "viewer"
)

// APIKey represents an API access credential.
type APIKey struct {
	ID          uuid.UUID    `json:"id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	Name        string       `json:"name"`
	KeyHash     string       `json:"-"`          // SHA-256 hash, never exposed
	KeyPrefix   string       `json:"key_prefix"` // First 8 chars for identification
	Permissions []string     `json:"permissions"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
	RevokedAt   *time.Time   `json:"revoked_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// IsRevoked returns true if the key has been revoked.
func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

// IsExpired returns true if the key has expired.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// HealthStatus represents aggregated health of a downstream service.
type HealthStatus struct {
	Service   string        `json:"service"`
	Status    string        `json:"status"` // "SERVING", "NOT_SERVING", "UNKNOWN"
	Latency   time.Duration `json:"latency"`
	CheckedAt time.Time     `json:"checked_at"`
}
