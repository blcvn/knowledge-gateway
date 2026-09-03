# SOL-INTEL-003 — Solution: Knowledge Evolution & Contradiction Resolution

| Field | Value |
|---|---|
| **Solution ID** | SOL-INTEL-003 |
| **CR** | [CR-INTEL-003](../../../../docs/crs/v4/intelligence/CR-INTEL-003-Knowledge-Evolution.md) |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

---

## 1. Giải pháp

Knowledge Evolution = khi memory mới contradicts memory cũ, resolve intelligently thay vì overwrite blindly.

### 1.1 `services/sm-memory/internal/usecase/evolution.go` [NEW]

```go
type KnowledgeEvolutionUseCase struct {
    memRepo port.MemoryRepository
    llm     port.LLMClient
}

type ConflictResolution struct {
    Action      string  // "supersede" | "coexist" | "merge" | "discard_new"
    Explanation string
    NewMemoryID string
    SupersededIDs []string
}

func (u *KnowledgeEvolutionUseCase) ProcessNewMemory(ctx context.Context, newMem *Memory) (*ConflictResolution, error) {
    // 1. Find semantically similar existing memories
    candidates, _ := u.memRepo.FindSimilar(ctx, newMem.TenantID, newMem.UserID, newMem.Embedding, threshold: 0.85)
    if len(candidates) == 0 {
        // No conflict — just store
        return &ConflictResolution{Action: "coexist"}, nil
    }

    // 2. LLM: detect contradiction or evolution
    prompt := buildContradictionPrompt(newMem, candidates)
    decision, _ := u.llm.Complete(ctx, &port.CompletionRequest{
        Prompt: prompt, MaxTokens: 200, ResponseFormat: "json",
    })

    var resolution ConflictResolution
    json.Unmarshal([]byte(decision.Content), &resolution)

    // 3. Apply resolution
    switch resolution.Action {
    case "supersede":
        u.memRepo.MarkSuperseded(ctx, resolution.SupersededIDs, newMem.ID)
    case "merge":
        merged, _ := u.llm.MergeMemories(ctx, append(candidates, newMem))
        u.memRepo.Store(ctx, merged)
        resolution.NewMemoryID = merged.ID
    }

    return &resolution, nil
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/sm-memory/internal/usecase/evolution.go` | NEW |
| `services/sm-memory/internal/domain/entity.go` | MODIFY — add SupersededBy field |
| `deployment/dev/migrations/0XX_memory_versions.sql` | NEW — superseded_by column |

---

## 3. Acceptance Criteria

- [ ] Contradictions detected with 90%+ accuracy (LLM benchmark)
- [ ] Superseded memories marked, not deleted (audit trail)
- [ ] Merge produces coherent combined memory
- [ ] Conflict resolution logged per memory_id
