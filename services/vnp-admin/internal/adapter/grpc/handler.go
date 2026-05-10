// Package grpc implements the gRPC adapter layer for vnp-admin.
// Maps gRPC requests → usecase calls → gRPC responses.
package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/usecase/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdminHandler implements VNPAdminService gRPC.
type AdminHandler struct {
	tenants port.TenantUseCase
	keys    port.APIKeyUseCase
	users   port.UserUseCase
	health  port.HealthCheckerPort
}

func NewAdminHandler(
	tenants port.TenantUseCase,
	keys port.APIKeyUseCase,
	users port.UserUseCase,
	health port.HealthCheckerPort,
) *AdminHandler {
	return &AdminHandler{tenants: tenants, keys: keys, users: users, health: health}
}

// -- Tenant RPCs --

func (h *AdminHandler) CreateTenant(ctx context.Context, name string, plan string) (*model.Tenant, error) {
	t, err := h.tenants.Create(ctx, name, model.Plan(plan))
	if err != nil {
		return nil, mapError(err)
	}
	return t, nil
}

func (h *AdminHandler) GetTenant(ctx context.Context, idStr string) (*model.Tenant, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant id: %s", idStr)
	}
	t, err := h.tenants.Get(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return t, nil
}

func (h *AdminHandler) ListTenants(ctx context.Context, offset, limit int) ([]*model.Tenant, int, error) {
	return h.tenants.List(ctx, offset, limit)
}

func (h *AdminHandler) DeleteTenant(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	return h.tenants.Delete(ctx, id)
}

// -- API Key RPCs --

func (h *AdminHandler) CreateAPIKey(ctx context.Context, tenantIDStr, name, scope string) (*model.APIKey, string, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, "", status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	key, plaintext, err := h.keys.Create(ctx, tenantID, name, model.KeyScope(scope))
	if err != nil {
		return nil, "", mapError(err)
	}
	return key, plaintext, nil
}

func (h *AdminHandler) ValidateAPIKey(ctx context.Context, plaintext string) (*model.APIKey, *model.Tenant, error) {
	key, tenant, err := h.keys.Validate(ctx, plaintext)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "invalid api key")
	}
	return key, tenant, nil
}

func (h *AdminHandler) RevokeAPIKey(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid key id")
	}
	return h.keys.Revoke(ctx, id)
}

// -- Health --

func (h *AdminHandler) GetAggregatedHealth(ctx context.Context) (map[string]bool, error) {
	return h.health.CheckAll(ctx)
}

// mapError converts domain errors to gRPC status codes.
func mapError(err error) error {
	switch err {
	case model.ErrTenantNotFound:
		return status.Errorf(codes.NotFound, err.Error())
	case model.ErrDuplicateTenant:
		return status.Errorf(codes.AlreadyExists, err.Error())
	case model.ErrQuotaExceeded:
		return status.Errorf(codes.ResourceExhausted, err.Error())
	case model.ErrAPIKeyInvalid, model.ErrAPIKeyRevoked:
		return status.Errorf(codes.Unauthenticated, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
