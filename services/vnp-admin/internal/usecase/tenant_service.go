// Package usecase implements tenant lifecycle operations.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/usecase/port"
)

// TenantService implements port.TenantUseCase.
type TenantService struct {
	repo repository.TenantRepository
	pub  port.EventPublisherPort
}

func NewTenantService(repo repository.TenantRepository, pub port.EventPublisherPort) *TenantService {
	return &TenantService{repo: repo, pub: pub}
}

func (s *TenantService) Create(ctx context.Context, name string, plan model.Plan) (*model.Tenant, error) {
	// Check for duplicates
	if existing, _ := s.repo.FindByName(ctx, name); existing != nil {
		return nil, model.ErrDuplicateTenant
	}

	tenant := &model.Tenant{
		ID:        uuid.New(),
		Name:      name,
		Plan:      plan,
		Config:    model.DefaultConfig(plan),
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	// Publish event (non-blocking)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.pub.PublishTenantCreated(bgCtx, tenant.ID)
	}()

	return tenant, nil
}

func (s *TenantService) Get(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TenantService) Update(ctx context.Context, id uuid.UUID, name *string, plan *model.Plan, config *model.TenantConfig) (*model.Tenant, error) {
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		tenant.Name = *name
	}
	if plan != nil {
		tenant.Plan = *plan
	}
	if config != nil {
		tenant.Config = *config
	}
	tenant.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}
	return tenant, nil
}

func (s *TenantService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.pub.PublishTenantDeleted(bgCtx, id)
	}()
	return nil
}

func (s *TenantService) List(ctx context.Context, offset, limit int) ([]*model.Tenant, int, error) {
	return s.repo.List(ctx, offset, limit)
}
