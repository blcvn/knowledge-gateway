// Package grpc implements gRPC handler adapters for vnp-platform.
// Maps 7 legacy gRPC service definitions to consolidated usecases.
package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdminHandler implements VnpAdminService gRPC.
type AdminHandler struct {
	tenants port.TenantUseCase
	keys    port.APIKeyUseCase
	users   port.UserUseCase
	health  port.HealthUseCase
}

func NewAdminHandler(t port.TenantUseCase, k port.APIKeyUseCase, u port.UserUseCase, h port.HealthUseCase) *AdminHandler {
	return &AdminHandler{tenants: t, keys: k, users: u, health: h}
}

func (h *AdminHandler) CreateTenant(ctx context.Context, name, slug string, tier string) (*admin.Tenant, error) {
	return h.tenants.CreateTenant(ctx, name, slug, admin.SubscriptionTier(tier))
}

func (h *AdminHandler) GetTenant(ctx context.Context, idStr string) (*admin.Tenant, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id")
	}
	return h.tenants.GetTenant(ctx, id)
}

func (h *AdminHandler) DeleteTenant(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid id")
	}
	return h.tenants.DeleteTenant(ctx, id)
}

func (h *AdminHandler) ListTenants(ctx context.Context, offset, limit int) ([]*admin.Tenant, int, error) {
	return h.tenants.ListTenants(ctx, offset, limit)
}

func (h *AdminHandler) CreateAPIKey(ctx context.Context, tenantIDStr, name string, permissions []string) (*admin.APIKey, string, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, "", status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	return h.keys.CreateKey(ctx, tenantID, name, permissions)
}

func (h *AdminHandler) ValidateAPIKey(ctx context.Context, rawKey string) (*admin.APIKey, error) {
	return h.keys.ValidateKey(ctx, rawKey)
}

func (h *AdminHandler) RevokeAPIKey(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid id")
	}
	return h.keys.RevokeKey(ctx, id)
}

func (h *AdminHandler) AggregatedHealth(ctx context.Context) ([]*admin.HealthStatus, error) {
	return h.health.AggregatedHealth(ctx)
}
