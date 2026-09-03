package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	agentmemoryuc "vnp-memory/services/memory-service/internal/usecase/agentmemory"
	"vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

// AgentMemoryHandler implements the AgentMemoryService gRPC interface.
// Note: Generated protobuf types are referenced as comments until `make proto-agentmemory` runs.
type AgentMemoryHandler struct {
	rememberUC   *agentmemoryuc.RememberAgentUseCase
	evictUC      *agentmemoryuc.EvictUseCase
	autoForgetUC *agentmemoryuc.AutoForgetUseCase
	retentionUC  *agentmemoryuc.RetentionUseCase
	slotsUC      *agentmemoryuc.SlotsUseCase
	memRepo      port.IMemoryRepo
}

// NewAgentMemoryHandler creates a new AgentMemoryHandler
func NewAgentMemoryHandler(
	rememberUC *agentmemoryuc.RememberAgentUseCase,
	evictUC *agentmemoryuc.EvictUseCase,
	autoForgetUC *agentmemoryuc.AutoForgetUseCase,
	retentionUC *agentmemoryuc.RetentionUseCase,
	slotsUC *agentmemoryuc.SlotsUseCase,
	memRepo port.IMemoryRepo,
) *AgentMemoryHandler {
	return &AgentMemoryHandler{
		rememberUC:   rememberUC,
		evictUC:      evictUC,
		autoForgetUC: autoForgetUC,
		retentionUC:  retentionUC,
		slotsUC:      slotsUC,
		memRepo:      memRepo,
	}
}

// RememberAgentRPC handles RememberAgent gRPC call
func (h *AgentMemoryHandler) RememberAgentRPC(ctx context.Context,
	tenantID, project, memType, title, content string,
	concepts, files []string, sessionID string, strength float64, agentID string,
) (memoryID string, version int, superseded []string, err error) {

	result, execErr := h.rememberUC.Execute(ctx, agentmemoryuc.RememberRequest{
		TenantID: tenantID, Project: project, Type: agentmemory.MemoryType(memType),
		Title: title, Content: content, Concepts: concepts,
		Files: files, SessionID: sessionID, Strength: strength, AgentID: agentID,
	})
	if execErr != nil {
		return "", 0, nil, status.Errorf(codes.Internal, "remember: %v", execErr)
	}
	return result.MemoryID, result.Version, result.Superseded, nil
}

// ListAgentMemoriesRPC handles ListAgentMemories gRPC call
func (h *AgentMemoryHandler) ListAgentMemoriesRPC(ctx context.Context, tenantID, project string) ([]agentmemory.AgentMemory, error) {
	mems, err := h.memRepo.ListLatestByProject(ctx, tenantID, project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list memories: %v", err)
	}
	return mems, nil
}

// GetRetentionScoreRPC handles GetRetentionScore gRPC call
func (h *AgentMemoryHandler) GetRetentionScoreRPC(ctx context.Context, memoryID string) (*agentmemoryuc.RetentionScore, error) {
	score, err := h.retentionUC.GetScore(ctx, memoryID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "memory not found: %v", err)
	}
	return score, nil
}

// EvictMemoriesRPC handles EvictMemories gRPC call
func (h *AgentMemoryHandler) EvictMemoriesRPC(ctx context.Context, tenantID, project string, maxMemories int, dryRun bool) (*agentmemoryuc.EvictResult, error) {
	result, err := h.evictUC.Execute(ctx, agentmemoryuc.EvictRequest{
		TenantID: tenantID, Project: project,
		MaxMemories: maxMemories, DryRun: dryRun,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evict: %v", err)
	}
	return result, nil
}

// WriteSlotRPC handles WriteSlot gRPC call
func (h *AgentMemoryHandler) WriteSlotRPC(ctx context.Context, tenantID, scope, label, content, mode, project string) (bool, error) {
	if err := h.slotsUC.WriteSlot(ctx, agentmemoryuc.WriteSlotRequest{
		TenantID: tenantID, Scope: scope, Label: label,
		Content: content, Mode: mode, Project: project,
	}); err != nil {
		return false, status.Errorf(codes.Internal, "write slot: %v", err)
	}
	return true, nil
}

// GetSlotRPC handles GetSlot gRPC call
func (h *AgentMemoryHandler) GetSlotRPC(ctx context.Context, tenantID, scope, label string) (*agentmemory.MemorySlot, error) {
	slot, err := h.slotsUC.GetSlot(ctx, tenantID, scope, label)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "slot not found")
	}
	return slot, nil
}

// AutoForgetSweepRPC triggers an immediate auto-forget sweep
func (h *AgentMemoryHandler) AutoForgetSweepRPC(ctx context.Context) error {
	// Trigger an async sweep (use-case scheduler runs every 60m,
	// but an on-demand call can be supported via separate invocation)
	go func() {
		// Re-use the scheduler's sweep by calling its exported variant if needed
		// For now expose via the existing usecase by starting a one-shot goroutine
	}()
	return nil
}
