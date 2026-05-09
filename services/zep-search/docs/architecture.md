---
id: DOC-S03
service: zep-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-search — Service Architecture

> **Group**: Zep | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/zep-search/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── search_query.go        # GraphSearchQuery, SessionSearchQuery
│   │   ├── search_result.go       # SearchResult, ranked items
│   │   ├── reranker.go            # RerankerType enum + strategy interface
│   │   ├── search_scope.go        # SearchScope (edges|nodes|episodes)
│   │   ├── search_filter.go       # Filters (labels, types, rating, lambda)
│   │   └── errors.go
│   ├── usecase/
│   │   ├── graph_search.go        # Multi-scope search + cache
│   │   ├── session_search.go      # Cross-session search
│   │   ├── get_relevant_facts.go  # Context assembly support
│   │   └── port/ + dto/
│   ├── adapter/
│   │   ├── grpc/handler.go, mapper.go
│   │   ├── client/graphiti_client.go  # HTTP → Graphiti search
│   │   ├── repository/redis/cache.go  # Search result cache
│   │   └── event/subscriber.go        # NATS cache invalidation
│   └── infra/
```

## Component Diagram

```mermaid
graph TD
    GW[vnp-gateway] -->|gRPC| H[gRPC Handler]
    ZM[zep-memory] -->|gRPC GetRelevantFacts| H
    H --> UC[Use Cases]
    UC --> CACHE[Redis Cache]
    UC --> GC[Graphiti Client]
    GC -->|HTTP POST /search| GRAPHITI[Graphiti]
    NATS[NATS JetStream] -->|graph events| SUB[NATS Subscriber]
    SUB -->|invalidate| CACHE
    CACHE -->|GET/SET| REDIS[(Redis)]
```

## External Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| Graphiti | HTTP | Search and get-memory endpoints |
| Redis | TCP | Search result caching (30s TTL) |
| NATS JetStream | Sub | Cache invalidation from zep-graph |
