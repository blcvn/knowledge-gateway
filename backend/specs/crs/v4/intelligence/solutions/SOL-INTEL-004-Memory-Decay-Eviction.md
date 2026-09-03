# SOL-INTEL-004 — Solution: Memory Decay & Salience Eviction

| Field | Value |
|---|---|
| **Solution ID** | SOL-INTEL-004 |
| **CR** | [CR-INTEL-004](../../../../docs/crs/v4/intelligence/CR-INTEL-004-Memory-Decay-Eviction.md) |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) §memory-service |
| **Status** | Open |
| **Priority** | 🟡 High |

---

## 1. Giải pháp

Salience-based eviction: `salience = importance × recency × frequency`. Memories với salience < threshold bị archive.

### 1.1 `services/memory-service/internal/usecase/decay.go` [NEW]

```go
type DecayUseCase struct {
    memRepo   port.MemoryRepository
    threshold float64 // default: 0.05
}

// SalienceScore calculates current salience for a memory
func SalienceScore(mem *Memory, now time.Time) float64 {
    ageDays := now.Sub(mem.CreatedAt).Hours() / 24.0

    // Importance: explicit weight (0-1)
    importance := mem.ImportanceScore
    if importance == 0 { importance = 0.5 }

    // Recency: exponential decay e^(-λt), λ=0.05/day
    recency := math.Exp(-0.05 * ageDays)

    // Frequency: normalized access count
    frequency := math.Log1p(float64(mem.AccessCount)) / 10.0

    return importance * recency * frequency
}

// RunEvictionSweep — scheduled job (NATS cron or cron job)
func (u *DecayUseCase) RunEvictionSweep(ctx context.Context, tenantID string) (*EvictionReport, error) {
    memories, _ := u.memRepo.GetAll(ctx, tenantID, ExcludeArchived)
    now := time.Now()

    var toArchive []string
    for _, mem := range memories {
        score := SalienceScore(mem, now)
        mem.SalienceScore = score
        u.memRepo.UpdateSalience(ctx, mem.ID, score)
        if score < u.threshold {
            toArchive = append(toArchive, mem.ID)
        }
    }

    u.memRepo.BatchArchive(ctx, toArchive)
    return &EvictionReport{Archived: len(toArchive), Total: len(memories)}, nil
}
```

### 1.2 Scheduled trigger

```go
// Run sweep every 24h via NATS JetStream scheduled message
// Or expose: POST /v1/memory/agent/evict (manual trigger per CR-AM-002)
func (h *AgentMemoryHandler) EvictMemories(w http.ResponseWriter, r *http.Request) {
    tenant := tenant.FromContext(r.Context())
    report, _ := h.decayUC.RunEvictionSweep(r.Context(), tenant)
    writeJSON(w, 200, report)
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/memory-service/internal/usecase/decay.go` | NEW |
| `services/memory-service/internal/domain/entity.go` | MODIFY — add SalienceScore, AccessCount |
| `deployment/dev/migrations/0XX_memory_salience.sql` | NEW |

---

## 3. Acceptance Criteria

- [ ] Salience formula: importance × recency × frequency
- [ ] Daily sweep archives memories with salience < 0.05
- [ ] Archived memories recoverable (soft delete)
- [ ] `GET /v1/memory/agent/{id}/retention` returns current salience score
