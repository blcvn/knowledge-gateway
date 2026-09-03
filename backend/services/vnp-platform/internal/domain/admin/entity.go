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
	Active    bool           `json:"active"`
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
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`          // SHA-256 hash, never exposed
	KeyPrefix   string     `json:"key_prefix"` // First 8 chars for identification
	Permissions []string   `json:"permissions"`
	Active      bool       `json:"active"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
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

// HTTP-facing HealthStatus values (capital first letter — TASK-007 / SOL-004)
type HTTPHealthStatus string

const (
	StatusHealthy  HTTPHealthStatus = "Healthy"
	StatusWarning  HTTPHealthStatus = "Warning"
	StatusCritical HTTPHealthStatus = "Critical"
	StatusUnknown  HTTPHealthStatus = "Unknown"
)

// GRPCToHTTPHealth maps gRPC health check status strings to frontend-facing values.
func GRPCToHTTPHealth(grpcStatus string) HTTPHealthStatus {
	switch grpcStatus {
	case "SERVING":
		return StatusHealthy
	case "NOT_SERVING":
		return StatusCritical
	default:
		return StatusUnknown
	}
}

// ─── OrgSettings + Members (TASK-008 / SOL-002) ───────────────────────────────

// OrgSettings is the tenant configuration view exposed to admin users.
type OrgSettings struct {
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Domain             string `json:"domain,omitempty"`
	Timezone           string `json:"timezone"`
	MaxAgents          int    `json:"max_agents"`
	MaxMemoriesPerUser int    `json:"max_memories_per_user"`
	Plan               string `json:"plan"` // "free" | "pro" | "enterprise"
}

// OrgMember represents a user within the organization for the members list.
type OrgMember struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Status   string    `json:"status"` // "active" | "inactive"
	JoinedAt time.Time `json:"joined_at"`
}

// OrgRole defines a role and its permissions.
type OrgRole struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// ─── Webhook (TASK-008 / SOL-002) ─────────────────────────────────────────────

// Webhook represents a registered webhook endpoint for SDK events.
type Webhook struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	URL         string     `json:"url"`
	Events      []string   `json:"events"`
	Status      string     `json:"status"`       // "active" | "paused" | "failed"
	SecretHash  string     `json:"-"`            // SHA-256 of signing secret — never exposed in JSON
	SuccessRate float64    `json:"success_rate"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreateWebhookPayload is the request body for creating a webhook.
type CreateWebhookPayload struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret,omitempty"` // Optional signing secret — stored as hash
}

// WebhookStatus values
const (
	WebhookActive = "active"
	WebhookPaused = "paused"
	WebhookFailed = "failed"
)

