# SOL-MB-002 — Solution: Memory Engine: Profile YOLO Merge

| Field | Value |
|---|---|
| **Solution ID** | SOL-MB-002 |
| **CR** | CR-MB-002 |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/memobase-engine` |

---

## 1. Giải pháp

YOLO Merge = update user profile with new facts without full rewrite. New facts override matching categories.

### `services/memobase-engine/internal/usecase/yolo.go` [NEW]

```go
// YOLO = You Only Live Once — merge aggressively
func yoloMerge(existing *UserProfile, newFacts map[string]ProfileFact) *UserProfile {
    if existing == nil {
        return &UserProfile{Categories: newFacts}
    }
    for k, v := range newFacts {
        if existing.Categories == nil { existing.Categories = map[string]ProfileFact{} }
        if existing.Categories[k].Confidence < v.Confidence || v.Confidence == 1.0 {
            existing.Categories[k] = v  // higher confidence wins
        }
    }
    existing.Version++
    existing.UpdatedAt = time.Now()
    return existing
}
```

## 2. Acceptance Criteria

- [ ] Profile updates idempotent
- [ ] Higher confidence facts overwrite lower
- [ ] Version counter increments on each merge

