---
id: TDD-zep-search
title: Technical Design — zep-search
service: zep-search
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Zep
---

# Technical Design — zep-search

> **Group**: Zep | **gRPC Port**: 9065 | **Health Port**: 12065

## 1. Service Overview

Semantic search across Temporal KG. Multi-scope search (edges/nodes/episodes) with 5 reranking strategies. Redis caching with 30s TTL. Called by zep-memory for context assembly.

## 2. Domain Model

- **GraphSearchQuery**: query, user_id, group_ids, scope, reranker, filters, limit
- **SessionSearchQuery**: query, user_id, session_ids, limit, max_facts
- **SearchScope**: edges | nodes | episodes | all
- **RerankerType**: rrf | mmr | cross_encoder | node_distance | episode_mentions
- **SearchResult**: Items[], Total, Query, Scope, Reranker, LatencyMs

## 3. Port Interfaces

```go
type SearchService interface {
  GraphSearch, SearchSessions, GetRelevantFacts
}
type GraphitiSearchClient interface { Search, GetMemory }
type SearchCacheStore interface { Get, Set, Invalidate }
```

## 4. NATS Events (Consumed)

| Subject | Action |
|---------|--------|
| `zep.graph.extraction.completed` | Invalidate cache for group |
| `zep.graph.fact.created` | Update cache |
| `zep.graph.fact.invalidated` | Remove from cache |

## 5. Reranker Strategies

| Strategy | Config | Best For |
|----------|--------|----------|
| RRF | K=60 | General-purpose |
| MMR | Lambda=0.5 | Diversity |
| CrossEncoder | ms-marco model | Accuracy |
| NodeDistance | MaxDepth=3 | Graph-aware |
| EpisodeMentions | Decay=0.95 | Recency |

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.
