package agentmemory

import (
	"context"

	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	"vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

// AuditListUseCase handles listing audit entries with filters
type AuditListUseCase struct {
	auditRepo port.IAuditRepo
}

// NewAuditListUseCase creates a new audit list use case
func NewAuditListUseCase(auditRepo port.IAuditRepo) *AuditListUseCase {
	return &AuditListUseCase{auditRepo: auditRepo}
}

// ListAudit returns audit entries matching the provided filter
func (uc *AuditListUseCase) ListAudit(ctx context.Context, filter port.AuditFilter) ([]agentmemory.AuditEntry, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	return uc.auditRepo.List(ctx, filter)
}
