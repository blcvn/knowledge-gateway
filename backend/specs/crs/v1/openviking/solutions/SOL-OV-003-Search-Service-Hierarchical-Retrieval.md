# SOL-OV-003 — Solution: Search Service (Hierarchical Retrieval)

| Field | Value |
|---|---|
| **Solution ID** | SOL-OV-003 |
| **CR** | CR-OV-003 |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/ov-search` |

---

## 1. Giải pháp

Hierarchical retrieval: exact match → fuzzy → semantic vector search, with BM25 reranking.

### `services/ov-search/internal/usecase/search.go` [MODIFY]

```go
func (u *SearchUseCase) Search(ctx context.Context, req *SearchRequest) ([]*FileChunk, error) {
    // Tier 1: Exact match (fast path)
    if results, _ := u.bm25.ExactMatch(ctx, req.TenantID, req.Query); len(results) >= req.Limit {
        return results[:req.Limit], nil
    }
    // Tier 2: Semantic similarity
    embedding, _ := u.embedder.Embed(ctx, req.Query)
    results, _ := u.vectorRepo.CosineSimilarity(ctx, req.TenantID, embedding, req.Limit*3)
    // Tier 3: BM25 reranking of vector results
    return u.bm25.Rerank(ctx, results, req.Query)[:req.Limit], nil
}
```

## 2. Acceptance Criteria

- [ ] 3-tier hierarchical search
- [ ] BM25 reranking improves precision
- [ ] grep mode (regex) returns line-level matches

