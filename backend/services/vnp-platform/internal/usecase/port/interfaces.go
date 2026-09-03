// Package port defines input/output port interfaces for vnp-platform usecases.
// These interfaces enforce Clean Architecture dependency rules:
// domain ← usecase → port (interfaces) ← adapter (implementations)
package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/analytics"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/event"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/project"
)

// --- Input Ports (Usecase interfaces called by handlers) ---

// TenantUseCase defines tenant management operations.
type TenantUseCase interface {
	CreateTenant(ctx context.Context, name, slug string, tier admin.SubscriptionTier) (*admin.Tenant, error)
	GetTenant(ctx context.Context, id uuid.UUID) (*admin.Tenant, error)
	UpdateTenant(ctx context.Context, id uuid.UUID, updates map[string]any) (*admin.Tenant, error)
	DeleteTenant(ctx context.Context, id uuid.UUID) error
	ListTenants(ctx context.Context, offset, limit int) ([]*admin.Tenant, int, error)
}

// APIKeyUseCase defines API key lifecycle operations.
type APIKeyUseCase interface {
	CreateKey(ctx context.Context, tenantID uuid.UUID, name string, permissions []string) (*admin.APIKey, string, error)
	ValidateKey(ctx context.Context, rawKey string) (*admin.APIKey, error)
	RevokeKey(ctx context.Context, id uuid.UUID) error
	ListKeys(ctx context.Context, tenantID uuid.UUID) ([]*admin.APIKey, error)
}

// UserUseCase defines user management operations.
type UserUseCase interface {
	CreateUser(ctx context.Context, tenantID uuid.UUID, email, name string, role admin.UserRole) (*admin.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*admin.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]any) (*admin.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*admin.User, int, error)
}

// HealthUseCase defines health check aggregation.
type HealthUseCase interface {
	AggregatedHealth(ctx context.Context) ([]*admin.HealthStatus, error)
}

// EventUseCase defines event timeline operations.
type EventUseCase interface {
	CreateEvent(ctx context.Context, evt *event.UserEvent) error
	GetTimeline(ctx context.Context, tenantID, userID uuid.UUID, limit int) (*event.Timeline, error)
	SearchEvents(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]event.UserEvent, error)
}

// AnalyticsUseCase defines usage tracking operations.
type AnalyticsUseCase interface {
	TrackUsage(ctx context.Context, record *analytics.UsageRecord) error
	GetUsageReport(ctx context.Context, tenantID uuid.UUID, period string) ([]*analytics.UsageRecord, error)
}

// ProjectUseCase defines space/tag management operations.
type ProjectUseCase interface {
	CreateSpace(ctx context.Context, tenantID uuid.UUID, name string) (*project.Space, error)
	GetSpace(ctx context.Context, id uuid.UUID) (*project.Space, error)
	ListSpaces(ctx context.Context, tenantID uuid.UUID) ([]*project.Space, error)
	DeleteSpace(ctx context.Context, id uuid.UUID) error
}

// --- Output Ports (Repository/Infrastructure interfaces) ---

// TenantRepository persists tenant data.
type TenantRepository interface {
	Create(ctx context.Context, tenant *admin.Tenant) error
	FindByID(ctx context.Context, id uuid.UUID) (*admin.Tenant, error)
	Update(ctx context.Context, tenant *admin.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*admin.Tenant, int, error)
}

// UserRepository persists user data.
type UserRepository interface {
	Create(ctx context.Context, user *admin.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*admin.User, error)
	Update(ctx context.Context, user *admin.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*admin.User, int, error)
}

// APIKeyRepository persists API key data.
type APIKeyRepository interface {
	Create(ctx context.Context, key *admin.APIKey) error
	FindByHash(ctx context.Context, hash string) (*admin.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*admin.APIKey, error)
}

// EventRepository persists timeline events.
type EventRepository interface {
	Create(ctx context.Context, evt *event.UserEvent) error
	FindByUser(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]event.UserEvent, int, error)
	SearchByText(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]event.UserEvent, error)
}

// UsageRepository persists analytics data.
type UsageRepository interface {
	Upsert(ctx context.Context, record *analytics.UsageRecord) error
	FindByTenant(ctx context.Context, tenantID uuid.UUID, period string) ([]*analytics.UsageRecord, error)
}

// SpaceRepository persists project/space data.
type SpaceRepository interface {
	Create(ctx context.Context, space *project.Space) error
	FindByID(ctx context.Context, id uuid.UUID) (*project.Space, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*project.Space, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EventPublisher publishes domain events to message bus.
type EventPublisher interface {
	PublishTenantCreated(ctx context.Context, tenant *admin.Tenant) error
	PublishTenantDeleted(ctx context.Context, tenantID uuid.UUID) error
}

// ServiceHealthChecker checks downstream service health.
type ServiceHealthChecker interface {
	CheckAll(ctx context.Context) ([]*admin.HealthStatus, error)
}
