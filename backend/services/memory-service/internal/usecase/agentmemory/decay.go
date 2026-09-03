package agentmemory

import (
    "context"
    "math"
    "time"

    "vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type DecayScheduler struct {
    repo         port.IMemoryRepo
    halfLifeDays int
}

func NewDecayScheduler(repo port.IMemoryRepo, halfLifeDays int) *DecayScheduler {
    if halfLifeDays <= 0 { halfLifeDays = 30 }
    return &DecayScheduler{repo: repo, halfLifeDays: halfLifeDays}
}

func (d *DecayScheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(2 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C: d.applyDecay(ctx)
        case <-ctx.Done(): return
        }
    }
}

func (d *DecayScheduler) applyDecay(ctx context.Context) {
    memories, _ := d.repo.ListAll(ctx)
    for _, m := range memories {
        hoursSince := time.Since(m.UpdatedAt).Hours()
        factor := math.Exp(-hoursSince / (float64(d.halfLifeDays) * 24))
        newStrength := m.Strength * factor
        d.repo.UpdateStrength(ctx, m.ID, newStrength)
        if newStrength < 0.05 {
            d.repo.FlagForEviction(ctx, m.ID)
        }
    }
}
