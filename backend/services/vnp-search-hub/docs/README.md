---
id: DOC-S01
service: vnp-search-hub
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Platform Team
---

# vnp-search-hub

> **Group**: Platform | **gRPC Port**: 9042 | **Health Port**: 9102 | **Origin**: Unified

## Purpose

Cross-engine search orchestration — the **`memory.recall()`** backend. Fan-out queries to all 7 engine search services in parallel, merge + deduplicate results, apply reranking (RRF/MMR/Cross-Encoder), and assemble unified recall responses within strict SLA budgets.

### Business Capability

- **Unified Recall**: Single API → parallel fan-out to cognee-search, graphiti-search, memobase-context, ov-search, zep-search, sm-search, vnp-event
- **Multi-Search**: Query specific engines with custom parameters
- **Result Merging**: Content-hash deduplication across engines
- **Reranking Strategies**: RRF (default, fast), MMR (diversity), Cross-Encoder (highest quality)
- **Token Budget**: Truncate results to fit LLM context window
- **Latency SLA**: p95 recall < 500ms, per-engine timeout 2s

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC (errgroup for parallel fan-out)
- **Database**: None (stateless orchestrator)
- **Architecture**: 4-layer Clean Architecture

## API Surface

```protobuf
service VnpSearchHubService {
  rpc Recall(RecallRequest) returns (RecallResponse);
  rpc MultiSearch(MultiSearchRequest) returns (MultiSearchResponse);
}
```

### Recall Pipeline

```
RecallRequest{query, tenant_id, scope, max_results}
  │
  ├── Parallel fan-out (errgroup, 2s timeout per engine):
  │   ├── cognee-search.Search(query)          → []DocumentChunk
  │   ├── graphiti-search.HybridSearch(query)   → []GraphFact
  │   ├── memobase-context.GetContext(user_id)  → ContextString
  │   ├── ov-search.HierarchicalSearch(query)   → []TieredContext
  │   ├── zep-search.GraphSearch(query)         → []TemporalFact
  │   ├── sm-search.HybridSearch(query)         → []MemoryEntry
  │   └── vnp-event.SearchEvents(query)         → []UserEvent
  │
  ├── Merge + Dedup (content hash)
  ├── Rerank (RRF / MMR / Cross-Encoder)
  ├── Token budget truncation
  │
  ▼
RecallResponse {
  profiles:  []ProfileSection,
  facts:     []GraphFact,
  events:    []TemporalEvent,
  documents: []DocumentChunk,
  context:   string  // pre-formatted for LLM prompt
  metadata:  RecallMetadata{latency_ms, engines_used[], total_results}
}
```

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| cognee-search | gRPC | Semantic KG search |
| graphiti-search | gRPC | Temporal graph search |
| memobase-context | gRPC | User profile + event context |
| ov-search | gRPC | Tiered hierarchical search |
| zep-search | gRPC | Temporal facts + context |
| sm-search | gRPC | Adaptive KG search |
| vnp-event | gRPC | Cross-domain event search |

## SLA Targets

| Metric | Target |
|--------|--------|
| Recall latency (p95) | < 500ms |
| Profile retrieval (p95) | < 100ms |
| Fan-out timeout | 2s max per engine |

## Links

- [API](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md) · [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — Platform
