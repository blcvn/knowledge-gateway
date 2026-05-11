---
id: FEAT-SEA-006
title: Reranker Implementations — 5 Strategies
service: graphiti-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement 5 reranking strategies as pluggable Reranker interface implementations.

## Scope

- `internal/usecase/reranker/interface.go` — Reranker interface
- `internal/usecase/reranker/rrf.go` — Reciprocal Rank Fusion
- `internal/usecase/reranker/mmr.go` — Maximal Marginal Relevance
- `internal/usecase/reranker/cross_encoder.go` — Neural reranking via graphiti-pipeline
- `internal/usecase/reranker/node_distance.go` — Graph proximity weighting
- `internal/usecase/reranker/episode_mentions.go` — Frequency-based relevance

### Reranker Interface

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, results []SearchResult) ([]RankedResult, error)
    Type() RerankerType
}
```

### Strategy Details

| Strategy | Algorithm | Use Case |
|----------|-----------|----------|
| **RRF** | `1/(k+rank)` fusion across ranked lists | Default combiner for multi-method results |
| **MMR** | λ·sim(q,d) - (1-λ)·max(sim(d,d')) | Diversity-promoting, avoids redundant results |
| **Cross-Encoder** | Neural scoring via graphiti-pipeline Rerank RPC | Highest quality, highest latency |
| **Node Distance** | Weight by BFS distance from query-related entities | Graph-topology-aware ranking |
| **Episode Mentions** | Weight by # episodes mentioning entity | Frequency/importance proxy |

## Acceptance Criteria

- [ ] AC-1: RRF combines multiple ranked lists with k=60 default
- [ ] AC-2: MMR promotes diversity with configurable λ (default 0.7)
- [ ] AC-3: Cross-Encoder delegates to graphiti-pipeline Rerank RPC
- [ ] AC-4: Node Distance uses BFS results from store to weight proximity
- [ ] AC-5: Episode Mentions counts entity occurrence across episodes
- [ ] AC-6: All rerankers implement Reranker interface (pluggable)
- [ ] AC-7: Rerankers can be chained (sequential application)

## Test Requirements
- **Unit tests**: Each reranker with known inputs and expected outputs
- **Minimum coverage**: 85% (core algorithm code)
