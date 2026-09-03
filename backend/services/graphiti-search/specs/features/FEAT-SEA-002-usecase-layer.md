---
id: FEAT-SEA-002
title: Usecase Layer — Hybrid Search Orchestrator
service: graphiti-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement usecase layer: hybrid search orchestrator (parallel search → merge → rerank → cache), individual search methods (node, edge, community), và port interfaces.

## Scope

- `internal/usecase/hybrid_search.go` — Main orchestrator
- `internal/usecase/node_search.go` — Entity-specific search
- `internal/usecase/edge_search.go` — Edge search with temporal filter
- `internal/usecase/community_search.go` — Community-level search
- Port interfaces: StoreSearchClient, EmbedderClient, CacheRepo, Reranker

### Hybrid Search Pipeline

```go
func (uc *HybridSearchUseCase) Execute(ctx context.Context, query SearchQuery) ([]RankedResult, error) {
    // 1. Check cache (hash of query + group_id)
    // 2. Generate query embedding via EmbedderClient
    // 3. Execute search methods in parallel (cosine, BM25, BFS)
    // 4. Merge results (union with dedup by entity UUID)
    // 5. Apply reranking pipeline (sequential: RRF → MMR → ...)
    // 6. Cache results with TTL
    // 7. Return top-K ranked results
}
```

## Acceptance Criteria

- [ ] AC-1: HybridSearch executes configured methods in parallel (goroutines)
- [ ] AC-2: Results merged with deduplication by entity UUID
- [ ] AC-3: Reranking pipeline applies strategies in order
- [ ] AC-4: Cache hit returns results without executing search
- [ ] AC-5: All usecases depend only on port interfaces

## Test Requirements
- **Unit tests**: Orchestrator with mock ports, parallel execution
- **Minimum coverage**: 80%
