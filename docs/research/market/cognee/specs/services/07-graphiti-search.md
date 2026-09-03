# 07 — Graphiti Search Service

> **gRPC**: 9022 | **Health**: 9096

---

## 1. Purpose

Hybrid temporal search: combines embedding similarity, BM25 keyword, graph traversal, và community-aware ranking trên bi-temporal knowledge graph.

---

## 2. Clean Architecture

```
services/graphiti-search/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # SearchResult, GraphPath, TemporalFilter
│   │   ├── value_object.go     # SearchMode, TimeRange, ScoreFusion
│   │   └── errors.go
│   ├── usecase/
│   │   ├── hybrid_search.go    # Orchestrate multi-strategy search
│   │   ├── entity_search.go    # Entity-focused search
│   │   ├── community_search.go # Community-based search
│   │   ├── center_search.go    # Node centrality search
│   │   ├── port/
│   │   │   └── output.go       # GraphSearcher, EmbeddingSearcher, Reranker, StoreClient
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/handler.go     # graphiti.search.v1.SearchService impl
│   │   ├── searcher/
│   │   │   ├── embedding.go    # Vector similarity search
│   │   │   ├── bm25.go         # Full-text keyword search
│   │   │   ├── graph.go        # Cypher-based graph traversal
│   │   │   ├── temporal.go     # Bi-temporal aware search
│   │   │   └── community.go    # Community membership search
│   │   ├── client/
│   │   │   ├── store_client.go     # gRPC → graphiti-store for graph queries
│   │   │   └── knowledge_client.go # gRPC → graphiti-knowledge for reranking
│   │   └── event/subscriber.go     # graphiti.entity.resolved → reindex
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
```

---

## 3. Search Pipeline

```
SearchRequest(query, group_id, center_node_uuid?, time_range?)
     │
     ▼
┌── Phase 1: Parallel Retrieval ──────────────────┐
│  1a. Embedding search (cosine sim, Qdrant)       │
│  1b. BM25 keyword search (Neo4j fulltext)       │
│  1c. Graph traversal (Cypher)                    │
│  1d. Community search (community_ids)            │
└────────────┬─────────────────────────────────────┘
             ▼
┌── Phase 2: Temporal Filtering ───────────────────┐
│  - Filter by valid_at / invalid_at timestamps    │
│  - Apply bi-temporal constraints                  │
│  - group_id isolation                             │
└────────────┬─────────────────────────────────────┘
             ▼
┌── Phase 3: Score Fusion + Reranking ─────────────┐
│  - RRF (Reciprocal Rank Fusion) merge            │
│  - Cross-encoder reranking via knowledge-svc     │
│  - Episode distance decay (recency bias)         │
│  - Return top_k results                           │
└──────────────────────────────────────────────────┘
```

---

## 4. Bi-Temporal Search

```go
type TemporalFilter struct {
    // Transaction time: when fact was recorded
    CreatedAfter  *time.Time
    CreatedBefore *time.Time
    
    // Valid time: when fact is/was true
    ValidAt       *time.Time  // Point-in-time query
    ValidFrom     *time.Time  // Range query start
    ValidTo       *time.Time  // Range query end
    
    // Include invalidated edges?
    IncludeInvalid bool
}
```
