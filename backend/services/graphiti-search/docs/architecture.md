---
id: DOC-S03
service: graphiti-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-search — Service Architecture

> **Group**: Graphiti | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/graphiti-search/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # SearchQuery, SearchResult, SearchConfig
│   │   ├── value_object.go        # RerankerType, SearchMethod, ScoreThreshold
│   │   ├── event.go               # SearchExecuted, CacheInvalidated
│   │   └── errors.go              # ErrEmptyQuery, ErrInvalidConfig
│   ├── usecase/
│   │   ├── hybrid_search.go       # Orchestrates multi-strategy search + rerank
│   │   ├── node_search.go         # Entity node focused search
│   │   ├── edge_search.go         # Fact/relationship search
│   │   ├── community_search.go    # Community summary search
│   │   └── port/
│   │       ├── input.go           # SearchUseCase interfaces
│   │       └── output.go          # StoreSearcher, KnowledgeEmbedder, KnowledgeReranker
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC server: HybridSearch, NodeSearch, EdgeSearch, CommunitySearch
│   │   │   └── mapper.go          # Proto ↔ Domain mapping
│   │   ├── client/
│   │   │   ├── knowledge_client.go # Embedding + reranking via graphiti-knowledge
│   │   │   └── store_client.go     # Vector/fulltext/BFS search via graphiti-store
│   │   └── event/
│   │       └── nats_subscriber.go  # Subscribe to index invalidation events
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

## Design Decisions

- **Parallel fan-out**: cosine similarity, BM25, and BFS run concurrently via errgroup
- **Pluggable reranking**: Strategy pattern allows swapping RRF/MMR/Cross-Encoder per request
- **Result caching**: Redis-backed cache for repeated queries within same group_id (invalidated by NATS events)
- **Temporal filters**: Applied post-reranking to avoid filtering before relevance scoring

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| graphiti-knowledge (gRPC) | Embedding generation + Cross-Encoder reranking |
| graphiti-store (gRPC) | Vector, fulltext, BFS search primitives |
| Redis | Search result caching |
| NATS JetStream | Cache invalidation events |
