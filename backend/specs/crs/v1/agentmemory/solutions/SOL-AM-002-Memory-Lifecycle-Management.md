# SOL-AM-002 — Solution: Memory Lifecycle Management

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-002 |
| **CR** | CR-AM-002 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/memory-service` |

---

## 1. Giải pháp

Memory lifecycle: create → read → decay → evict → archive. Jaccard versioning for deduplication.

### Jaccard Versioning

```go
// services/memory-service/internal/usecase/versioning.go [NEW]
func JaccardSimilarity(a, b []string) float64 {
    setA := map[string]bool{}; for _, t := range a { setA[t] = true }
    intersect := 0
    for _, t := range b { if setA[t] { intersect++ } }
    return float64(intersect) / float64(len(setA)+len(b)-intersect)
}

func (u *VersioningUseCase) StoreWithVersioning(ctx context.Context, mem *Memory) error {
    tokens := tokenize(mem.Content)
    candidates, _ := u.memRepo.FindRecent(ctx, mem.TenantID, mem.UserID, mem.Type, 20)
    
    for _, existing := range candidates {
        sim := JaccardSimilarity(tokens, tokenize(existing.Content))
        if sim > 0.85 {
            // Same content: create new version of existing
            existing.Version++
            existing.Content = mem.Content
            existing.UpdatedAt = time.Now()
            return u.memRepo.Update(ctx, existing)
        }
    }
    // New content: store as v1
    mem.Version = 1
    return u.memRepo.Store(ctx, mem)
}
```

## 2. File Changes

| File | Action |
|---|---|
| `services/memory-service/internal/usecase/versioning.go` | NEW |
| `services/memory-service/internal/usecase/decay.go` | NEW (see SOL-INTEL-004) |
| `deployment/dev/migrations/0XX_memory_service.sql` | VERIFY |

## 3. Acceptance Criteria

- [ ] Jaccard > 0.85 → new version (no duplicate)
- [ ] Jaccard < 0.85 → new independent memory
- [ ] Memory CRUD: GET, LIST, UPDATE, DELETE via agent memory API
- [ ] Decay + eviction per SOL-INTEL-004

