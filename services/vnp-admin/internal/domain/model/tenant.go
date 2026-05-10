// Package model defines core domain entities for vnp-admin.
// Reference: specs/tdd.md §2.1 Domain Layer
package model

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents a tenant subscription tier.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanStarter    Plan = "starter"
	PlanEnterprise Plan = "enterprise"
)

// Tenant is the root aggregate for multi-tenant isolation.
type Tenant struct {
	ID        uuid.UUID    `json:"id"`
	Name      string       `json:"name"`
	Plan      Plan         `json:"plan"`
	Config    TenantConfig `json:"config"`
	Active    bool         `json:"active"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// TenantConfig holds per-tenant feature flags and quotas (stored as JSONB).
type TenantConfig struct {
	MaxAPIKeys      int            `json:"max_api_keys"`
	MaxUsers        int            `json:"max_users"`
	EnabledEngines  []string       `json:"enabled_engines"`
	RateLimitRPM    int            `json:"rate_limit_rpm"`
	StorageQuotaMB  int64          `json:"storage_quota_mb"`
	FeatureFlags    map[string]bool `json:"feature_flags,omitempty"`
}

// DefaultConfig returns the default tenant config for a given plan.
func DefaultConfig(plan Plan) TenantConfig {
	switch plan {
	case PlanEnterprise:
		return TenantConfig{
			MaxAPIKeys: 100, MaxUsers: 1000,
			EnabledEngines: []string{"cognee", "graphiti", "memobase", "openviking", "zep", "supermemory"},
			RateLimitRPM: 10000, StorageQuotaMB: 102400,
		}
	case PlanStarter:
		return TenantConfig{
			MaxAPIKeys: 10, MaxUsers: 50,
			EnabledEngines: []string{"cognee", "graphiti", "memobase", "zep"},
			RateLimitRPM: 1000, StorageQuotaMB: 10240,
		}
	default: // Free
		return TenantConfig{
			MaxAPIKeys: 2, MaxUsers: 5,
			EnabledEngines: []string{"cognee", "zep"},
			RateLimitRPM: 100, StorageQuotaMB: 1024,
		}
	}
}
