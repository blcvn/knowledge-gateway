// Package analytics implements usage tracking.
package analytics

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/analytics"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
)

// Service implements port.AnalyticsUseCase.
type Service struct {
	repo port.UsageRepository
}

func NewService(repo port.UsageRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) TrackUsage(ctx context.Context, record *analytics.UsageRecord) error {
	return s.repo.Upsert(ctx, record)
}

func (s *Service) GetUsageReport(ctx context.Context, tenantID uuid.UUID, period string) ([]*analytics.UsageRecord, error) {
	return s.repo.FindByTenant(ctx, tenantID, period)
}
