package agentmemory

import (
    "context"
    "log"

    "vnp-memory/services/memory-service/internal/domain/agentmemory"
    "vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type GovernanceDeleteUseCase struct {
    memRepo      port.IMemoryRepo
    searchClient port.ISearchNotifier
    graphClient  port.IGraphClient  // optional: HTTP client to cognee/graphiti
    auditRepo    port.IAuditRepo
    publisher    port.IEventPublisher
}

type GovernanceDeleteRequest struct {
    MemoryID    string
    TenantID    string
    PerformedBy string
    Reason      string
}

func (uc *GovernanceDeleteUseCase) Execute(ctx context.Context, req GovernanceDeleteRequest) error {
    mem, err := uc.memRepo.Get(ctx, req.MemoryID)
    if err != nil { return agentmemory.ErrMemoryNotFound }

    // 1. Delete from PostgreSQL (cascade: related observations via FK)
    if err := uc.memRepo.Delete(ctx, req.MemoryID); err != nil { return err }

    // 2. Remove from BM25 index (non-fatal)
    if err := uc.searchClient.RemoveMemory(ctx, req.MemoryID); err != nil {
        log.Printf("[governance] BM25 index removal failed for %s: %v", req.MemoryID, err)
    }

    // 3. Remove from Vector index (non-fatal)
    if err := uc.searchClient.RemoveMemory(ctx, req.MemoryID); err != nil {
        log.Printf("[governance] Vector index removal failed for %s: %v", req.MemoryID, err)
    }

    // 4. Remove graph edges (non-fatal; only if graphiti/cognee configured)
    if uc.graphClient != nil {
        if err := uc.graphClient.RemoveBySourceID(ctx, req.MemoryID); err != nil {
            log.Printf("[governance] Graph edge removal failed for %s: %v", req.MemoryID, err)
        }
    }

    // 5. Create audit entry
    entry := agentmemory.NewAuditEntry(req.TenantID, agentmemory.AuditGovernanceDelete, []string{req.MemoryID})
    entry.PerformedBy = req.PerformedBy
    entry.Project     = mem.Project
    entry.Reason      = req.Reason
    entry.Details     = map[string]any{"memory_type": string(mem.Type), "title": mem.Title}
    uc.auditRepo.Save(ctx, entry)

    // 6. Publish NATS event
    uc.publisher.Publish(ctx, "agentmemory.memory.governance_deleted", map[string]any{
        "memory_id": req.MemoryID, "tenant_id": req.TenantID, "performed_by": req.PerformedBy,
    })

    return nil
}
