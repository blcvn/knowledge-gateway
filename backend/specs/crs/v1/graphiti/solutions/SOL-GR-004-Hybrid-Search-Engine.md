# SOL-GR-004 — Solution: Hybrid Search Engine

| Field | Value |
|---|---|
| **Solution ID** | SOL-GR-004 |
| **CR** | CR-GR-004 |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/graphiti-search` |

---

## 1. Phân tích

Multi-strategy search: BM25 + vector similarity + graph traversal + temporal filtering, fused with RRF.

### Key: `services/graphiti-search/internal/usecase/search.go` [MODIFY]

```go
func (u *SearchUseCase) HybridSearch(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
    ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
    defer cancel()

    bm25Results := make(chan []Entity, 1)
    vectorResults := make(chan []Entity, 1)
    graphResults := make(chan []Entity, 1)

    go func() { res, _ := u.bm25.Search(ctx, req); bm25Results <- res }()
    go func() { res, _ := u.vector.Search(ctx, req); vectorResults <- res }()
    go func() { res, _ := u.graph.TraversalSearch(ctx, req); graphResults <- res }()

    all := [][]Entity{<-bm25Results, <-vectorResults, <-graphResults}
    return &SearchResult{Entities: rrfFusion(all)[:req.Limit]}, nil
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/graphiti-search/internal/usecase/search.go` | MODIFY — add hybrid search |
| `services/graphiti-search/internal/usecase/temporal.go` | NEW — time range filter |

---

## 3. Acceptance Criteria

- [ ] p95 < 500ms for hybrid search
- [ ] Temporal filtering by date range
- [ ] RRF fusion scores entities from 3 sources
