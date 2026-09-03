// Package domain defines core types for graphiti-admin service.
// SOL-007: Admin Service & Observability Stack (CR-GR-007)
package domain

import "time"

// Tenant represents a group-scoped namespace in the graphiti system.
type Tenant struct {
	GroupID   string
	Name      string
	CreatedAt time.Time
	Config    TenantConfig
}

// TenantConfig holds per-tenant override configuration.
type TenantConfig struct {
	MaxEpisodes   int    // 0 = unlimited
	LLMProvider   string // override default (e.g., "anthropic")
	EmbedProvider string
}

// TenantStats aggregates graph object counts for a tenant.
type TenantStats struct {
	GroupID             string
	EpisodeCount        int64
	EntityCount         int64
	EdgeCount           int64
	CommunityCount      int64
	StorageSizeEstimate string // human-readable, e.g. "2.3 MB"
}

// PostgresTenant is the database representation of a Tenant.
type PostgresTenant struct {
	GroupID       string    `db:"group_id"`
	Name          string    `db:"name"`
	MaxEpisodes   int       `db:"max_episodes"`
	LLMProvider   string    `db:"llm_provider"`
	EmbedProvider string    `db:"embed_provider"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// MaintenanceTask represents an admin background operation.
type MaintenanceTask struct {
	ID        string
	GroupID   string
	TaskType  string // "rebuild_communities" | "build_indices" | "delete_data"
	Status    string // "pending" | "running" | "done" | "failed"
	StartedAt time.Time
	DoneAt    *time.Time
	Error     string
}
