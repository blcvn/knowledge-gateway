---
id: TASK-064
title: "[SOL-003 T10] vnp-search-hub — Unified Cross-Engine Search Handler"
service: vnp-search-hub
type: FEAT
priority: P1
status: Done
created: 2026-05-14
updated: 2026-05-14
linked_specs:
  - ui/specs/solutions/SOL-003-ui-gateway-hardening.md
  - gateway/specs/solutions/SOL-002-ux-console-api-upgrade.md
---

## Mục Tiêu
Expose HTTP handler trong `vnp-search-hub` service cho unified cross-engine search — fan-out query tới 6 memory engines và aggregate kết quả.

## Bối Cảnh Nghiệp Vụ
Gateway đã implement `console_search_usecase.go` (SOL-002 T12) với 6-engine fan-out search pattern. `vnp-search-hub` cần expose actual handler để nhận search requests và orchestrate queries tới: cognee, graphiti, zep, openviking, memobase, supermemory.

## Phạm Vi Công Việc (Scope)

### In Scope
1. **Search Handler**:
   - `POST /api/v1/search` — Unified search endpoint
   - Request: `{ query, engines[], filters, limit, offset, reranking }`
   - Response: `{ results[], total, facets: { byEngine, byType }, latencyMs }`
2. **Fan-out Orchestrator**: Concurrent queries to 6 engine search endpoints
3. **Result Merger**: De-duplicate, score-sort, facet aggregation
4. **Reranking Support**: Optional reranking mode (semantic, temporal, hybrid)

### Out of Scope
- Individual engine search optimization (each engine handles their own)
- Search analytics / query logging (future spec)

## Thiết Kế Kỹ Thuật

### API Contract
```
POST /api/v1/search
Body: {
  "query": "user preferences for dark mode",
  "engines": ["memobase", "graphiti", "zep"],  // optional, default all
  "filters": { "engine": "memobase", "type": "profile" },
  "limit": 20,
  "offset": 0,
  "reranking": "hybrid"   // semantic | temporal | hybrid | none
}
Response 200: {
  "results": [MemoryItem],
  "total": 47,
  "facets": {
    "byEngine": { "memobase": 20, "graphiti": 15, "zep": 12 },
    "byType": { "profile": 18, "episodic": 15, "semantic": 14 }
  },
  "latencyMs": 230
}
```

### Internal Architecture
```
handler/search_handler.go
  → usecase/search_orchestrator.go
    → clients/cognee_client.go      (gRPC)
    → clients/graphiti_client.go    (gRPC)
    → clients/zep_client.go         (gRPC)
    → clients/openviking_client.go  (gRPC)
    → clients/memobase_client.go    (gRPC)
    → clients/supermemory_client.go (gRPC)
    → merger/result_merger.go       (de-dup + score sort)
    → reranker/reranker.go          (optional)
```

## Acceptance Criteria
- [ ] AC-1: `POST /api/v1/search` returns merged results from specified engines
- [ ] AC-2: Results include facets grouped by engine and memory type
- [ ] AC-3: Latency ≤ 500ms for 3-engine fan-out (with 5s per-engine timeout)
- [ ] AC-4: Partial results returned if some engines timeout (graceful degradation)
- [ ] AC-5: Unit tests ≥ 80% coverage for orchestrator + merger logic

## Test Requirements
- Unit tests: Merger logic, reranking, timeout handling
- Integration tests: Mock gRPC servers for each engine
- Minimum coverage: 80%
