---
id: DOC-S01
service: zep-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Zep Team
---

# zep-search

> **Group**: Zep (Context Engineering) | **gRPC Port**: 9065 | **Health Port**: 12065 | **Origin**: Zep

## Purpose

Semantic search across Zep's Temporal Knowledge Graph. Cung cấp multi-scope search (edges/nodes/episodes) with 5 reranking strategies, session-level search, context retrieval for memory assembly (called by zep-memory), và Redis-backed search result caching.

### Business Capability

- **Graph Search**: Multi-scope search with 5 reranking strategies (RRF, MMR, Cross-Encoder, Node Distance, Episode Mentions)
- **Session Search**: Cross-session fact discovery
- **Context Retrieval**: GetRelevantFacts called by zep-memory for context assembly
- **Fact Filtering**: Label, type, rating, temporal filters
- **Caching**: Redis-backed with short TTL (30s) for temporal data freshness

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream consumer
- **Cache**: Redis (search result caching, 30s TTL)
- **External**: Graphiti HTTP (search/get-memory endpoints)
- **Architecture**: 4-layer Clean Architecture

## Quick Start

```bash
make build-zep-search
make run-zep-search
docker compose up zep-search
```

## API Surface

### gRPC Service

```protobuf
service SearchService {
  rpc GraphSearch(GraphSearchRequest) returns (SearchResponse);
  rpc SearchSessions(SessionSearchRequest) returns (SessionSearchResponse);
  rpc GetRelevantFacts(GetRelevantFactsRequest) returns (FactListResponse);
}
```

## Reranking Strategies

| Reranker | Algorithm | Latency | Best For |
|----------|-----------|---------|----------|
| `rrf` | Reciprocal Rank Fusion | Low | General-purpose, balanced |
| `mmr` | Maximal Marginal Relevance | Low | Diverse results |
| `cross_encoder` | Neural cross-encoder | High | Best accuracy |
| `node_distance` | Graph shortest path | Medium | Relationship-aware |
| `episode_mentions` | Episode frequency | Low | Recency-aware |

## NATS Events

### Consumed

| Subject | Source | Action |
|---------|--------|--------|
| `zep.graph.extraction.completed` | zep-graph | Invalidate search cache |
| `zep.graph.fact.created` | zep-graph | Update cache with new fact |
| `zep.graph.fact.invalidated` | zep-graph | Remove invalidated fact |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| Graphiti | HTTP | Search and get-memory endpoints |
| Redis | TCP | Search result caching |
| NATS JetStream | Sub | Cache invalidation events |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Reference Design](../../../references/zep/specs/services/06-search-service.md)

## Owner

- **Team**: VNP Memory — Zep
