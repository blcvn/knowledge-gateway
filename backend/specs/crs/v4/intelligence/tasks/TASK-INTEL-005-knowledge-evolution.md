# TASK-INTEL-005 — Knowledge Evolution: detect contradictions, supersede or merge old memories

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-005 |
| **Wave** | 2 |
| **Solution** | [SOL-INTEL-003](../solutions/SOL-INTEL-003-Knowledge-Evolution.md) §1.1 |
| **Component** | `services/sm-memory/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 6h |

---

## Mục tiêu

Knowledge Evolution: detect contradictions, supersede or merge old memories

---

## Công việc cụ thể

### `services/sm-memory/internal/usecase/evolution.go` [NEW]

```go
const contradictionPrompt = `Compare these two pieces of information:
OLD: %s
NEW: %s

Determine if they contradict, if the new supersedes the old, or if they should coexist.
Return JSON: {"action": "supersede|coexist|merge|discard_new", "explanation": "..."}
Action rules:
- supersede: new info explicitly replaces old (e.g., "now works at X" vs "used to work at Y")
- coexist: both valid, different context
- merge: partially overlapping, combine
- discard_new: old info more accurate/recent`

func (u *KnowledgeEvolutionUseCase) ProcessNewMemory(ctx context.Context, newMem *Memory) (*ConflictResolution, error) {
    // Find similar memories (cosine similarity > 0.85)
    candidates, _ := u.memRepo.FindSimilar(ctx, newMem.TenantID, newMem.UserID, newMem.Embedding, 0.85)
    if len(candidates) == 0 {
        return &ConflictResolution{Action: "coexist"}, nil
    }

    // Check top candidate for contradiction
    best := candidates[0]
    prompt := fmt.Sprintf(contradictionPrompt, best.Content, newMem.Content)
    resp, _ := u.llm.Complete(ctx, &port.CompletionRequest{
        Prompt: prompt, MaxTokens: 100, ResponseFormat: "json",
        Task: "contradiction_detect",
    })

    var resolution ConflictResolution
    json.Unmarshal([]byte(resp.Content), &resolution)

    switch resolution.Action {
    case "supersede":
        u.memRepo.MarkSuperseded(ctx, []string{best.ID}, newMem.ID)
    case "merge":
        mergePrompt := fmt.Sprintf("Combine these facts into one coherent statement:\n1. %s\n2. %s", best.Content, newMem.Content)
        merged, _ := u.llm.Complete(ctx, &port.CompletionRequest{Prompt: mergePrompt, MaxTokens: 200})
        newMem.Content = merged.Content
        u.memRepo.MarkSuperseded(ctx, []string{best.ID}, newMem.ID)
    }

    return &resolution, nil
}
```

### Domain changes: add SupersededBy

```go
// services/sm-memory/internal/domain/entity.go [MODIFY]
type Memory struct {
    ID           string
    TenantID     string
    UserID       string
    Content      string
    Embedding    []float32
    SupersededBy string    // ID of newer memory (if this was replaced)
    Version      int
    ImportanceScore float64
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

---

## Acceptance Criteria

- [ ] Contradictions detected with LLM
- [ ] "supersede" action marks old memory with SupersededBy field
- [ ] "merge" combines content, marks old as superseded
- [ ] Superseded memories not deleted (audit trail)
- [ ] "coexist" leaves both untouched
- [ ] Unit test with mock LLM for each action type

## Files

```
services/sm-memory/internal/usecase/evolution.go          [NEW]
services/sm-memory/internal/domain/entity.go              [MODIFY — SupersededBy]
deployment/dev/migrations/0042_memory_evolution.sql        [NEW — add superseded_by column]
```
