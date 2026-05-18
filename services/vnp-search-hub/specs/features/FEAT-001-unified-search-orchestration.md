---
id: FEAT-001
title: Unified Search Orchestration
service: vnp-search-hub
version: 1.0.0
status: Draft
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: gateway/SOL-002 (T12)
linked_ux: "ux_spec.md §6.2 Memory Explorer, §6.6 Context Debugger"
---

## Mục Tiêu

Orchestrate cross-engine search: fan-out queries to all 6 engines, merge + rerank results, return unified response với facets.

## Scope

### In Scope
- gRPC `SearchHub.UnifiedSearch(query, filters, engines)` — fan-out search
- gRPC `SearchHub.GetMemoryDetail(id, engine)` — single memory detail
- gRPC `SearchHub.GetNeighbors(id, engine, depth)` — graph neighbors
- Reranking strategies: RRF, MMR, Cross-Encoder
- Faceted aggregation by engine, memory type, time range

### Out of Scope
- LLM-based query rewrite (future)
- Embedding generation (engines do this internally)

## Thiết Kế Kỹ Thuật

### Business Logic

1. Parse filters from query (engine list, memory_type, time range)
2. Fan-out search concurrently to selected engines:
   - `cognee-search.Search(query, strategy)`
   - `graphiti-search.Search(query, temporal_filter)`
   - `zep-search.SearchGraph(query)` + `zep-search.SearchSession(query)`
   - `ov-search.Search(query, levels)`
   - `memobase-context.SearchEvents(user_id, query)` + profiles
   - `sm-search.Search(query, filters)`
3. Normalize scores across engines (min-max normalization)
4. Apply reranking strategy (RRF default)
5. Compute facets (count by engine, by type)
6. Return merged results with provenance metadata

### Reranking Strategies

```go
type RerankStrategy interface {
    Rerank(results []EngineResult) []RankedResult
}

// RRF: Reciprocal Rank Fusion
// MMR: Maximal Marginal Relevance (diversity)
// CrossEncoder: Neural reranking (requires LLM call)
```

## Acceptance Criteria
- [ ] AC-1: Fan-out to 6 engines concurrently, timeout per engine 5s
- [ ] AC-2: Failed engine excluded from results, others still return
- [ ] AC-3: RRF reranking produces consistent ordering
- [ ] AC-4: Facets correctly count results per engine and memory type
- [ ] AC-5: Score normalization produces 0.0-1.0 range across engines
- [ ] AC-6: GetMemoryDetail returns full provenance + raw payload
- [ ] AC-7: p95 latency < 500ms for 6-engine search

## Test Requirements
- Unit tests: Score normalization, RRF, facet computation
- Integration tests: Multi-engine mock search
- Minimum coverage: 80%
