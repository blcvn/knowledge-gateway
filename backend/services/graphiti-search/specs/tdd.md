---
id: TDD-graphiti-search
title: Technical Design — graphiti-search
service: graphiti-search
version: 2.0.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Graphiti
linked_sol: SOL-001
---

# Technical Design — graphiti-search

> **Group**: Graphiti | **gRPC Port**: 9022 | **Health Port**: 9095

## 1. Service Overview

Hybrid search engine over the Graphiti temporal knowledge graph. Combines vector similarity (cosine), full-text (BM25), and BFS graph traversal with configurable multi-strategy reranking. Results cached in Redis with NATS-driven cache invalidation.

**Key Characteristics:**
- 3 search methods: cosine similarity, BM25 fulltext, BFS graph traversal
- 5 reranking strategies: RRF, MMR, Cross-Encoder, Node Distance, Episode Mentions
- Redis-backed result caching with TTL + NATS invalidation
- Temporal filtering (valid_at, invalid_at window queries)
- Multi-tenant isolation by group_id scoping

## 2. Clean Architecture Layers

### 2.1 Domain Layer

```
internal/domain/
├── entity.go          # SearchQuery, SearchResult, RankedResult, SearchFilter
├── value_object.go    # SearchMethod, RerankerType, ScoreWeight, TemporalWindow
├── config.go          # SearchConfig, RerankerConfig, CacheConfig
└── errors.go          # ErrNoResults, ErrInvalidQuery, ErrCacheUnavailable
```

**Key Types:**
```go
type SearchMethod string
const (
    MethodCosine  SearchMethod = "cosine_similarity"
    MethodBM25    SearchMethod = "bm25"
    MethodBFS     SearchMethod = "breadth_first_search"
)

type RerankerType string
const (
    RerankerRRF            RerankerType = "rrf"
    RerankerMMR            RerankerType = "mmr"
    RerankerCrossEncoder   RerankerType = "cross_encoder"
    RerankerNodeDistance    RerankerType = "node_distance"
    RerankerEpisodeMentions RerankerType = "episode_mentions"
)
```

### 2.2 Usecase Layer

```
internal/usecase/
├── hybrid_search.go       # Main orchestrator: parallel search → merge → rerank → cache
├── node_search.go         # Entity-specific search (name, summary, embedding)
├── edge_search.go         # Edge-specific search (fact, temporal range)
├── community_search.go    # Community-level search
├── reranker/
│   ├── interface.go       # Reranker interface
│   ├── rrf.go             # Reciprocal Rank Fusion
│   ├── mmr.go             # Maximal Marginal Relevance
│   ├── cross_encoder.go   # Neural reranking (delegates to graphiti-pipeline)
│   ├── node_distance.go   # Graph proximity weighting
│   └── episode_mentions.go # Frequency-based relevance
├── port/
│   ├── input.go           # SearchUseCase, NodeSearchUseCase, EdgeSearchUseCase
│   └── output.go          # StoreSearchClient, EmbedderClient, CacheRepo, KnowledgeClient
└── dto/
    ├── request.go
    └── response.go
```

### 2.3 Adapter Layer

```
internal/adapter/
├── grpc/
│   ├── handler.go         # HybridSearch, NodeSearch, EdgeSearch, CommunitySearch
│   └── mapper.go          # Proto ↔ Domain mapping
├── client/
│   ├── store_client.go    # gRPC → graphiti-store (search primitives)
│   └── pipeline_client.go # gRPC → graphiti-pipeline (cross-encoder rerank)
├── cache/
│   └── redis_cache.go     # Search result caching (TTL + key hashing)
└── event/
    └── nats_subscriber.go # Cache invalidation on graphiti.episode.ingested
```

### 2.4 Infrastructure Layer

```
internal/infra/
├── config/config.go       # Search-specific config (cache TTL, reranker weights)
├── server/grpc.go
├── telemetry/
└── wire/wire.go
```

## 3. gRPC API

```protobuf
service GraphitiSearchService {
  rpc HybridSearch(HybridSearchRequest) returns (HybridSearchResponse);
  rpc NodeSearch(NodeSearchRequest) returns (NodeSearchResponse);
  rpc EdgeSearch(EdgeSearchRequest) returns (EdgeSearchResponse);
  rpc CommunitySearch(CommunitySearchRequest) returns (CommunitySearchResponse);
}
```

## 4. Hybrid Search Pipeline

```
Query → Embed → [parallel] Cosine + BM25 + BFS → Merge → Rerank → Cache → Return
```

## 5. Cross-Service Dependencies

| Service | Direction | Protocol | Operations |
|---------|-----------|----------|-----------|
| graphiti-store | Outbound gRPC | :9024 | CosineSimilaritySearch, FulltextSearch, BFSSearch |
| graphiti-pipeline | Outbound gRPC | :9021 | GenerateEmbedding (query embedding), Rerank (cross-encoder) |
| Redis | Outbound | TCP :6379 | Result caching (GET/SET with TTL) |
| NATS | Inbound | JetStream | Subscribe `graphiti.episode.ingested` → invalidate cache |

## 6. Observability

- **Metrics**: `graphiti_search_duration_seconds{method,reranker}`, `graphiti_search_cache_hit_ratio`, `graphiti_search_results_count{method}`, `graphiti_search_rerank_duration_seconds{strategy}`
- **Traces**: OTel spans for search pipeline stages (embed, search, merge, rerank, cache)
- **Logs**: Structured JSON: request_id, tenant_id, query_hash, method, result_count, cache_hit
- **Health**: gRPC health + HTTP /healthz, /readyz on :9095

## 7. Multi-Tenancy

- `x-tenant-id` → `group_id` scoping on all store queries
- Cache keys prefixed with `group_id` for isolation
- NATS invalidation filters by `group_id`

## Feature Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| FEAT-SEA-001 | Domain layer | ⏳ Draft | P0 |
| FEAT-SEA-002 | Usecase layer | ⏳ Draft | P0 |
| FEAT-SEA-003 | gRPC handlers | ⏳ Draft | P0 |
| FEAT-SEA-004 | Store client adapter | ⏳ Draft | P0 |
| FEAT-SEA-005 | Redis cache + NATS | ⏳ Draft | P0 |
| FEAT-SEA-006 | Rerankers (5 strategies) | ⏳ Draft | P0 |
| FEAT-SEA-007 | Infrastructure | ⏳ Draft | P0 |

---

> **Next Steps**: Implement FEAT specs from SOL-001 in dependency order.
