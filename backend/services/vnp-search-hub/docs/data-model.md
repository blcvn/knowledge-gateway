---
id: DOC-S04
service: vnp-search-hub
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-search-hub — Data Model

> **Database**: Redis (cache) — no primary SQL database

## Redis Keys

### Result Cache

| Key Pattern | Type | TTL | Description |
|-------------|------|-----|-------------|
| `recall:{tenant_id}:{query_hash}` | STRING (JSON) | 60s | Cached recall results |
| `recall:meta:{tenant_id}:{query_hash}` | HASH | 60s | Recall metadata (engines used, latency) |

### Engine Health

| Key Pattern | Type | TTL | Description |
|-------------|------|-----|-------------|
| `engine:health:{engine_name}` | HASH | 10s | Engine health status |
| `engine:latency:{engine_name}` | SORTED SET | 5m | Recent latency samples |

## Internal State (In-Memory)

| Component | Structure | Description |
|-----------|----------|-------------|
| Engine Registry | `[]EngineEndpoint` | All search engine gRPC endpoints |
| Reranker Config | `RerankConfig` | Default + per-tenant reranking strategy |
| Token Budget | `int` | Max tokens for context assembly |

## Recall Pipeline Data Flow

```
RecallRequest → Query Hash → Cache Check
  → Cache Miss:
    → Fan-out to 7 engines (errgroup, 2s timeout)
    → Collect []EngineResult
    → Merge + Dedup (content hash)
    → Rerank (RRF / MMR / Cross-Encoder)
    → Token Budget Truncation
    → Cache Result (60s TTL)
  → Return RecallResponse
```

## No SQL Tables

The search hub is stateless — it orchestrates fan-out queries to engine search services. Redis caches recent results to avoid redundant fan-out queries.
