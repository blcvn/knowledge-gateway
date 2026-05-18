// Package usecase implements policy management business logic for vnp-admin.
package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/repository"
)

// PolicyService handles OPA policy CRUD operations.
type PolicyService struct {
	repo   repository.PolicyRepository
	logger *slog.Logger
}

// NewPolicyService creates a new policy service.
func NewPolicyService(repo repository.PolicyRepository, logger *slog.Logger) *PolicyService {
	return &PolicyService{repo: repo, logger: logger}
}

// Create creates a new policy in draft status.
func (s *PolicyService) Create(ctx context.Context, tenantID uuid.UUID, name, description, regoCode, scope string) (*model.Policy, error) {
	p := model.NewPolicy(tenantID, name, description, regoCode, scope)
	if err := s.repo.Create(ctx, p); err != nil {
		s.logger.Error("failed to create policy", "error", err, "name", name)
		return nil, err
	}
	s.logger.Info("policy created", "id", p.ID, "name", name, "scope", scope)
	return p, nil
}

// Get retrieves a policy by ID.
func (s *PolicyService) Get(ctx context.Context, id uuid.UUID) (*model.Policy, error) {
	return s.repo.FindByID(ctx, id)
}

// Update modifies an existing policy.
func (s *PolicyService) Update(ctx context.Context, id uuid.UUID, name, description, regoCode *string, status *model.PolicyStatus) (*model.Policy, error) {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		p.Name = *name
	}
	if description != nil {
		p.Description = *description
	}
	if regoCode != nil {
		p.RegoCode = *regoCode
	}
	if status != nil {
		p.Status = *status
	}
	p.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	s.logger.Info("policy updated", "id", id, "name", p.Name)
	return p, nil
}

// Delete removes a policy.
func (s *PolicyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// ListByTenant returns all policies for a tenant.
func (s *PolicyService) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.Policy, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}
