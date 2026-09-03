// Package port defines the outbound ports for graphiti-admin service.
// SOL-007: Admin Service & Observability Stack
package port

import (
	"context"

	"vnp-memory/services/graphiti-admin/internal/domain"
)

// TenantRepository is the storage port for tenant lifecycle management.
type TenantRepository interface {
	Get(ctx context.Context, groupID string) (*domain.Tenant, error)
	Save(ctx context.Context, t domain.Tenant) error
	Delete(ctx context.Context, groupID string) error
	List(ctx context.Context) ([]*domain.Tenant, error)
}

// StorePort is the client port for graphiti-store service.
type StorePort interface {
	ClearData(ctx context.Context, groupIDs []string) error
	RemoveCommunities(ctx context.Context, groupID string) error
	GetGroupStats(ctx context.Context, groupID string) (*GroupStats, error)
	BuildIndicesAndConstraints(ctx context.Context) error
	DeleteAllIndexes(ctx context.Context) error
}

// KnowledgePort is the client port for graphiti-knowledge service.
type KnowledgePort interface {
	BuildCommunities(ctx context.Context, req BuildCommunitiesReq) (*BuildCommunitiesResp, error)
	GetTokenUsage(ctx context.Context, req GetTokenUsageReq) (map[string]*TokenUsage, error)
}

// EventPublisher publishes NATS events.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload interface{}) error
}

// GroupStats holds aggregate counts returned by graphiti-store.
type GroupStats struct {
	EpisodeCount   int64
	EntityCount    int64
	EdgeCount      int64
	CommunityCount int64
}

// BuildCommunitiesReq is the request for community detection.
type BuildCommunitiesReq struct {
	GroupID string
}

// BuildCommunitiesResp is the response from community detection.
type BuildCommunitiesResp struct {
	CommunitiesBuilt int64
}

// GetTokenUsageReq is the request for token usage report.
type GetTokenUsageReq struct {
	GroupID string // empty = all tenants
}

// TokenUsage holds per-prompt LLM usage stats.
type TokenUsage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CallCount        int64
}
