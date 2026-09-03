package agentmemory

import (
    "context"
    "time"

    "vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type AutoForgetUseCase struct {
    repo         port.IMemoryRepo
    searchClient port.ISearchNotifier
    publisher    port.IEventPublisher
}

func (uc *AutoForgetUseCase) StartScheduler(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C: uc.sweep(ctx)
        case <-ctx.Done(): return
        }
    }
}

func (uc *AutoForgetUseCase) sweep(ctx context.Context) {
    expired, _ := uc.repo.FindExpired(ctx)
    for _, mem := range expired {
        uc.repo.Delete(ctx, mem.ID)
        go uc.searchClient.RemoveMemory(context.Background(), mem.ID)
        uc.publisher.Publish(ctx, "agentmemory.memory.expired", map[string]any{
            "memory_id": mem.ID, "reason": "ttl",
        })
    }
}
