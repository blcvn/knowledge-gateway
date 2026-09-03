# TASK-INTEL-006 — Memory Decay & Salience Eviction: time-based score decay, archive low-salience

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-006 |
| **Wave** | 2 |
| **Solution** | [SOL-INTEL-004](../solutions/SOL-INTEL-004-Memory-Decay-Eviction.md) §1.1 |
| **Component** | `services/memory-service/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-INTEL-007 |
| **Estimated** | 5h |

---

## Mục tiêu

Memory Decay & Salience Eviction: time-based score decay, archive low-salience

---

## Công việc cụ thể

### `services/memory-service/internal/usecase/decay.go` [NEW]

```go
// SalienceScore: importance × recency × frequency
// recency = e^(-λt), λ=0.05/day
func SalienceScore(mem *Memory, now time.Time) float64 {
    ageDays := now.Sub(mem.CreatedAt).Hours() / 24.0
    importance := mem.ImportanceScore
    if importance == 0 { importance = 0.5 }
    recency   := math.Exp(-0.05 * ageDays)
    frequency  := math.Log1p(float64(mem.AccessCount)) / 10.0
    return importance * recency * frequency
}

type DecayUseCase struct {
    memRepo   port.MemoryRepository
    threshold float64 // default: 0.05
}

func NewDecayUseCase(repo port.MemoryRepository) *DecayUseCase {
    return &DecayUseCase{memRepo: repo, threshold: 0.05}
}

// RunEvictionSweep — archive memories with salience < threshold
func (u *DecayUseCase) RunEvictionSweep(ctx context.Context, tenantID string) (*EvictionReport, error) {
    memories, err := u.memRepo.GetAll(ctx, tenantID, ExcludeArchived)
    if err != nil { return nil, err }
    now := time.Now()

    var toArchive []string
    for _, mem := range memories {
        score := SalienceScore(mem, now)
        u.memRepo.UpdateSalience(ctx, mem.ID, score)
        if score < u.threshold {
            toArchive = append(toArchive, mem.ID)
        }
    }
    if len(toArchive) > 0 {
        u.memRepo.BatchArchive(ctx, toArchive)
    }

    return &EvictionReport{
        Archived: len(toArchive), Total: len(memories),
        Threshold: u.threshold, RunAt: now,
    }, nil
}
```

### Unit tests

```go
func TestSalienceScore_NewMemory(t *testing.T) {
    mem := &Memory{CreatedAt: time.Now(), ImportanceScore: 0.8, AccessCount: 5}
    score := SalienceScore(mem, time.Now())
    assert.Greater(t, score, 0.4, "new memory with high importance should have high salience")
}

func TestSalienceScore_OldMemory(t *testing.T) {
    mem := &Memory{
        CreatedAt: time.Now().Add(-365 * 24 * time.Hour), // 1 year old
        ImportanceScore: 0.5, AccessCount: 1,
    }
    score := SalienceScore(mem, time.Now())
    assert.Less(t, score, 0.05, "year-old low-access memory should be below threshold")
}

func TestEvictionSweep_ArchivesLowSalience(t *testing.T) { ... }
```

### HTTP endpoint (manual trigger)

```go
// POST /v1/memory/agent/evict
func (h *AgentMemoryHandler) EvictMemories(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    report, err := h.decayUC.RunEvictionSweep(r.Context(), tenantID)
    if err != nil { writeError(w, 500, "evict_failed", err.Error()); return }
    writeJSON(w, 200, report)
}
```

---

## Acceptance Criteria

- [ ] SalienceScore formula: importance × e^(-0.05×days) × log1p(access)/10
- [ ] RunEvictionSweep archives memories with score < 0.05
- [ ] Archived memories soft-deleted (archived=true, not deleted)
- [ ] `go test` covers new/old memory edge cases
- [ ] POST /v1/memory/agent/evict triggers sweep

## Files

```
services/memory-service/internal/usecase/decay.go          [NEW]
services/memory-service/internal/usecase/decay_test.go     [NEW]
services/memory-service/internal/domain/entity.go          [MODIFY — salience, access_count]
gateway/adapter/handler/agentmemory_handler.go             [MODIFY — add evict endpoint]
```

**Trạng thái:** ✅ Implemented

---

**Ghi chú audit:** memory-service/usecase/agentmemory/decay.go: DecayScheduler with exponential half-life decay; FlagForEviction at strength<0.05
