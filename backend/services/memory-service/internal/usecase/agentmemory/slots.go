package agentmemory

import (
    "context"

    "vnp-memory/services/memory-service/internal/domain/agentmemory"
    "vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type SlotsUseCase struct { repo port.ISlotsRepo }

type WriteSlotRequest struct {
    TenantID string; Project string; Scope string; Label string
    Content  string; Mode string  // "replace" | "append"
}

func (uc *SlotsUseCase) WriteSlot(ctx context.Context, req WriteSlotRequest) error {
    slot, err := uc.repo.GetSlot(ctx, req.TenantID, req.Scope, req.Label)
    if err != nil {
        return uc.repo.CreateSlot(ctx, agentmemory.MemorySlot{
            TenantID: req.TenantID, Project: req.Project, Scope: req.Scope,
            Label: req.Label, Content: req.Content,
        })
    }
    if slot.ReadOnly { return agentmemory.ErrSlotReadOnly }
    switch req.Mode {
    case "append": slot.Content = slot.Content + "\n" + req.Content
    default:       slot.Content = req.Content
    }
    if slot.SizeLimit > 0 && len(slot.Content) > slot.SizeLimit {
        return agentmemory.ErrSlotSizeExceeded
    }
    return uc.repo.UpdateSlot(ctx, *slot)
}

func (uc *SlotsUseCase) GetSlot(ctx context.Context, tenantID, scope, label string) (*agentmemory.MemorySlot, error) {
    return uc.repo.GetSlot(ctx, tenantID, scope, label)
}

func (uc *SlotsUseCase) DeleteSlot(ctx context.Context, tenantID, scope, label string) error {
    return uc.repo.DeleteSlot(ctx, tenantID, scope, label)
}

func (uc *SlotsUseCase) ListSlots(ctx context.Context, tenantID, scope string) ([]agentmemory.MemorySlot, error) {
    return uc.repo.ListSlots(ctx, tenantID, scope)
}
