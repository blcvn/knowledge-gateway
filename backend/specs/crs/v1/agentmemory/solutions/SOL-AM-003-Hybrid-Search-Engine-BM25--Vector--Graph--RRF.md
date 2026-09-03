# SOL-AM-003 — Solution: Hybrid Search Engine (BM25 + Vector + Graph + RRF)

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-003 |
| **CR** | CR-AM-003 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/observe-service` |

---

## 1. Giải pháp

Observe-service BM25 search (already implemented) + vector search (step 10) + graph entities (Graphiti).

### `services/observe-service/internal/usecase/search.go` [MODIFY]

```go
func (u *SearchUseCase) HybridSearch(ctx context.Context, req *ObserveSearchRequest) (*SearchResult, error) {
    ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
    defer cancel()

    bm25Ch := make(chan []Observation, 1)
    vecCh := make(chan []Observation, 1)

    go func() { res, _ := u.bm25.Search(ctx, req.TenantID, req.SessionID, req.Query); bm25Ch <- res }()
    go func() {
        embedding, _ := u.embedder.Embed(ctx, req.Query)
        res, _ := u.vectorRepo.SearchObservations(ctx, req.TenantID, embedding, 20)
        vecCh <- res
    }()

    bm25Results := <-bm25Ch
    vecResults := <-vecCh
    fused := rrfFusion(bm25Results, vecResults)
    return &SearchResult{Observations: fused[:min(req.Limit, len(fused))]}, nil
}
```

## 2. Acceptance Criteria

- [ ] BM25 + vector fusion via RRF
- [ ] Session-scoped search (optional session_id filter)
- [ ] p95 < 300ms

