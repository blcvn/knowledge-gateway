package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	agentmemoryuc "vnp-memory/services/memory-service/internal/usecase/agentmemory"
	"vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

// GovernanceHandler implements the GovernanceService gRPC interface
type GovernanceHandler struct {
	deleteUC  *agentmemoryuc.GovernanceDeleteUseCase
	auditRepo port.IAuditRepo
}

// NewGovernanceHandler creates a new GovernanceHandler
func NewGovernanceHandler(
	deleteUC *agentmemoryuc.GovernanceDeleteUseCase,
	auditRepo port.IAuditRepo,
) *GovernanceHandler {
	return &GovernanceHandler{
		deleteUC:  deleteUC,
		auditRepo: auditRepo,
	}
}

// GovernanceDeleteRPC handles the governance cascade delete call
func (h *GovernanceHandler) GovernanceDeleteRPC(ctx context.Context, memoryID, tenantID, performedBy, reason string) error {
	err := h.deleteUC.Execute(ctx, agentmemoryuc.GovernanceDeleteRequest{
		MemoryID:    memoryID,
		TenantID:    tenantID,
		PerformedBy: performedBy,
		Reason:      reason,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "governance delete: %v", err)
	}
	return nil
}

// ListAuditRPC returns audit entries for a given tenant and operation filter
func (h *GovernanceHandler) ListAuditRPC(ctx context.Context, filter port.AuditFilter) ([]agentmemory.AuditEntry, error) {
	entries, err := h.auditRepo.List(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list audit: %v", err)
	}
	return entries, nil
}
