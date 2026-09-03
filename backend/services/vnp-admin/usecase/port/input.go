// Package port defines input port interfaces (usecase boundaries).
package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/model"
)

// TenantUseCase defines tenant lifecycle operations.
type TenantUseCase interface {
	Create(ctx context.Context, name string, plan model.Plan) (*model.Tenant, error)
	Get(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	Update(ctx context.Context, id uuid.UUID, name *string, plan *model.Plan, config *model.TenantConfig) (*model.Tenant, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*model.Tenant, int, error)
}

// APIKeyUseCase defines API key lifecycle operations.
type APIKeyUseCase interface {
	Create(ctx context.Context, tenantID uuid.UUID, name string, scope model.KeyScope) (*model.APIKey, string, error) // Returns key + plaintext
	Validate(ctx context.Context, plaintext string) (*model.APIKey, *model.Tenant, error) // Returns key + tenant
	Revoke(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID) ([]*model.APIKey, error)
}

// UserUseCase defines user management operations.
type UserUseCase interface {
	Create(ctx context.Context, tenantID uuid.UUID, email, name string, role model.UserRole) (*model.User, error)
	Get(ctx context.Context, id uuid.UUID) (*model.User, error)
	List(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*model.User, int, error)
}

// EventPublisherPort abstracts NATS event publishing.
type EventPublisherPort interface {
	PublishTenantCreated(ctx context.Context, tenantID uuid.UUID) error
	PublishTenantDeleted(ctx context.Context, tenantID uuid.UUID) error
	PublishKeyRevoked(ctx context.Context, tenantID, keyID uuid.UUID) error
}

// HealthCheckerPort abstracts gRPC health fan-out.
type HealthCheckerPort interface {
	CheckAll(ctx context.Context) (map[string]bool, error)
}
