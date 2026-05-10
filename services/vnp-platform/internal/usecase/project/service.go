// Package project implements space/project management.
package project

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/project"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
)

// Service implements port.ProjectUseCase.
type Service struct {
	repo port.SpaceRepository
}

func NewService(repo port.SpaceRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateSpace(ctx context.Context, tenantID uuid.UUID, name string) (*project.Space, error) {
	space := &project.Space{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, space); err != nil {
		return nil, err
	}
	return space, nil
}

func (s *Service) GetSpace(ctx context.Context, id uuid.UUID) (*project.Space, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListSpaces(ctx context.Context, tenantID uuid.UUID) ([]*project.Space, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *Service) DeleteSpace(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
