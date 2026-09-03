---
id: TDD-vnp-search-hub
title: Technical Design — vnp-search-hub
service: vnp-search-hub
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Platform
---

# Technical Design — vnp-search-hub

> **Group**: Platform | **gRPC Port**: 9042 | **Health Port**: 9102

## 1. Service Overview

Cross-engine search orchestration. Fan-out queries to 7 engine search services in parallel, merge + dedup + rerank results. This is the `memory.recall()` backend — stateless orchestrator.

## 2. Domain Layer

- **RecallRequest**: query, tenant_id, scope (all|semantic|episodic|profile|procedural|adaptive|events), max_results, rerank_strategy, token_budget
- **RecallResponse**: profiles[], facts[], events[], documents[], context (pre-formatted string), metadata (latency_ms, engines_used[], total_results)
- **RerankStrategy**: enum — RRF (default) | MMR (diversity) | CrossEncoder (quality)
- **EngineResult**: engine_name, results[], latency_ms, error

## 3. gRPC API

```protobuf
service VnpSearchHubService {
  rpc Recall(RecallRequest) returns (RecallResponse);
  rpc MultiSearch(MultiSearchRequest) returns (MultiSearchResponse);
}
```

## 4. Recall Pipeline

```
RecallRequest → embed query (shared)
  │
  ├─ errgroup.Go: cognee-search.Search() (2s timeout)
  ├─ errgroup.Go: graphiti-search.HybridSearch() (2s timeout)
  ├─ errgroup.Go: memobase-context.GetContext() (2s timeout)
  ├─ errgroup.Go: ov-search.HierarchicalSearch() (2s timeout)
  ├─ errgroup.Go: zep-search.GraphSearch() (2s timeout)
  ├─ errgroup.Go: sm-search.HybridSearch() (2s timeout)
  └─ errgroup.Go: vnp-event.SearchEvents() (2s timeout)
  │
  ▼ [Collect results — partial success OK if ≥1 engine responds]
  │
  ├─ Merge: normalize scores across engines (0-1 scale)
  ├─ Dedup: content hash to remove cross-engine duplicates
  ├─ Rerank:
  │   ├─ RRF: 1/(k+rank) per engine, sum across engines (fast, default)
  │   ├─ MMR: maximize relevance while minimizing redundancy
  │   └─ CrossEncoder: Bifrost cross-encoder model (slow, highest quality)
  ├─ Token budget: truncate to fit max_tokens (default 4096)
  │
  ▼ AssembleResponse
```

## 5. Reranking Strategies

| Strategy | Latency | Quality | Use Case |
|----------|---------|---------|----------|
| RRF (Reciprocal Rank Fusion) | < 5ms | Good | Default, fast responses |
| MMR (Maximal Marginal Relevance) | < 10ms | Good | Diversity-focused results |
| Cross-Encoder | 200-500ms | Best | Highest quality (via Bifrost) |

## 6. Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| cognee-search | gRPC | Semantic KG results |
| graphiti-search | gRPC | Temporal graph results |
| memobase-context | gRPC | User profile + events |
| ov-search | gRPC | Tiered context (L0/L1/L2) |
| zep-search | gRPC | Temporal facts |
| sm-search | gRPC | Adaptive KG + memory results |
| vnp-event | gRPC | Cross-domain events |
| Bifrost | HTTP | Cross-encoder reranking |

## 7. Observability

- **Metrics**: recall_total, recall_latency_seconds (histogram), engine_latency_per_engine, engine_errors_total, rerank_latency
- **Traces**: OTel spans for fan-out per engine, merge, rerank
- **Health**: gRPC + HTTP /healthz on port 9102

## 8. SLA Targets

| Metric | Target |
|--------|--------|
| Recall latency (p95) | < 500ms |
| Profile retrieval (p95) | < 100ms |
| Fan-out timeout | 2s max per engine |

---

> **Next Steps**: FEAT-001 (Recall API with fan-out), FEAT-002 (RRF Reranking), FEAT-003 (Token Budget Truncation), QA-001 (SLA Benchmark)
