// Package usecase implements the tenant management use case for graphiti-admin.
// SOL-007: Admin Service & Observability Stack (CR-GR-007)
package usecase

import (
	"context"
	"fmt"
	"time"

	"vnp-memory/services/graphiti-admin/internal/domain"
	"vnp-memory/services/graphiti-admin/internal/usecase/port"
)

// TenantManagementUseCase handles tenant CRUD and stats.
type TenantManagementUseCase struct {
	tenantRepo port.TenantRepository
	storePort  port.StorePort
	publisher  port.EventPublisher
}

// NewTenantManagementUseCase constructs the use case.
func NewTenantManagementUseCase(
	tenantRepo port.TenantRepository,
	storePort port.StorePort,
	publisher port.EventPublisher,
) *TenantManagementUseCase {
	return &TenantManagementUseCase{
		tenantRepo: tenantRepo,
		storePort:  storePort,
		publisher:  publisher,
	}
}

// CreateTenantReq is the input for CreateTenant.
type CreateTenantReq struct {
	GroupID string
	Name    string
	Config  domain.TenantConfig
}

// CreateTenant creates a new tenant (idempotent).
func (uc *TenantManagementUseCase) CreateTenant(ctx context.Context, req CreateTenantReq) (*domain.Tenant, error) {
	// Idempotent — return existing if already exists
	existing, _ := uc.tenantRepo.Get(ctx, req.GroupID)
	if existing != nil {
		return existing, nil
	}

	tenant := domain.Tenant{
		GroupID:   req.GroupID,
		Name:      req.Name,
		CreatedAt: time.Now(),
		Config:    req.Config,
	}
	if err := uc.tenantRepo.Save(ctx, tenant); err != nil {
		return nil, fmt.Errorf("save tenant: %w", err)
	}

	// Publish event → Store can init schema (for per-group-id mode)
	_ = uc.publisher.Publish(ctx, "graphiti.tenant.created", map[string]any{
		"group_id": req.GroupID,
		"name":     req.Name,
	})

	return &tenant, nil
}

// DeleteTenant removes a tenant and all its graph data.
func (uc *TenantManagementUseCase) DeleteTenant(ctx context.Context, groupID string) error {
	// 1. Clear all graph data for this tenant
	if err := uc.storePort.ClearData(ctx, []string{groupID}); err != nil {
		return fmt.Errorf("clear data: %w", err)
	}

	// 2. Remove tenant record
	if err := uc.tenantRepo.Delete(ctx, groupID); err != nil {
		return fmt.Errorf("delete tenant record: %w", err)
	}

	_ = uc.publisher.Publish(ctx, "graphiti.tenant.deleted", map[string]any{"group_id": groupID})
	return nil
}

// ListTenants returns all tenants.
func (uc *TenantManagementUseCase) ListTenants(ctx context.Context) ([]*domain.Tenant, error) {
	return uc.tenantRepo.List(ctx)
}

// GetTenantStats returns aggregate graph counts for a tenant.
func (uc *TenantManagementUseCase) GetTenantStats(ctx context.Context, groupID string) (*domain.TenantStats, error) {
	stats := &domain.TenantStats{GroupID: groupID}

	counts, err := uc.storePort.GetGroupStats(ctx, groupID)
	if err != nil {
		return stats, nil // non-fatal: return empty stats
	}

	stats.EpisodeCount   = counts.EpisodeCount
	stats.EntityCount    = counts.EntityCount
	stats.EdgeCount      = counts.EdgeCount
	stats.CommunityCount = counts.CommunityCount
	return stats, nil
}
