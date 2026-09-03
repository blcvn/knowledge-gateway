// Package repository defines output port interfaces for persistence.
package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/model"
)

// TenantRepository persists tenants.
type TenantRepository interface {
	Create(ctx context.Context, t *model.Tenant) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	FindByName(ctx context.Context, name string) (*model.Tenant, error)
	Update(ctx context.Context, t *model.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*model.Tenant, int, error)
}

// APIKeyRepository persists API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, k *model.APIKey) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.APIKey, error)
	FindByHash(ctx context.Context, hash string) (*model.APIKey, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

// UserRepository persists users.
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*model.User, error)
	Update(ctx context.Context, u *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*model.User, int, error)
}

// AuditLogRepository persists audit log entries.
type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	Search(ctx context.Context, filter model.AuditLogFilter) ([]*model.AuditLog, int, error)
}

// PolicyRepository persists OPA governance policies.
type PolicyRepository interface {
	Create(ctx context.Context, p *model.Policy) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Policy, error)
	Update(ctx context.Context, p *model.Policy) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.Policy, error)
}

