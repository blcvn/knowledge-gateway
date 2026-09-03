// Package usecase implements audit log business logic for vnp-admin.
package usecase

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/domain/repository"
)

// AuditService handles audit log recording and querying.
type AuditService struct {
	repo   repository.AuditLogRepository
	logger *slog.Logger
}

// NewAuditService creates a new audit service.
func NewAuditService(repo repository.AuditLogRepository, logger *slog.Logger) *AuditService {
	return &AuditService{repo: repo, logger: logger}
}

// Record creates a new audit log entry.
func (s *AuditService) Record(ctx context.Context, tenantID uuid.UUID, userID string, action model.AuditAction, resourceType, resourceID string) (*model.AuditLog, error) {
	log := model.NewAuditLog(tenantID, userID, action, resourceType, resourceID)
	if err := s.repo.Create(ctx, log); err != nil {
		s.logger.Error("failed to record audit log", "error", err, "action", action)
		return nil, err
	}
	s.logger.Info("audit log recorded", "id", log.ID, "action", action, "resource", resourceType+"/"+resourceID)
	return log, nil
}

// Search queries audit logs with filters.
func (s *AuditService) Search(ctx context.Context, filter model.AuditLogFilter) ([]*model.AuditLog, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.Search(ctx, filter)
}
