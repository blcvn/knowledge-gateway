---
id: TDD-graphiti-search
title: Technical Design — graphiti-search
service: graphiti-search
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-09
group: Graphiti
---

# Technical Design — graphiti-search

> **Group**: Graphiti | **gRPC Port**: 9022 | **Health Port**: 9095

## 1. Service Overview

Hybrid search over temporal knowledge graph combining vector similarity (cosine), full-text (BM25), BFS graph traversal, and configurable reranking (RRF/MMR/Cross-Encoder/Node Distance/Episode Mentions).

## 2. Clean Architecture Layers

### Domain Layer
- SearchQuery, SearchConfig, SearchResult, SearchFilter
- SearchMethod: cosine_similarity, bm25, breadth_first_search
- Reranker: rrf, mmr, cross_encoder, node_distance, episode_mentions

### Usecase Layer
Business logic orchestration. Imports domain only.

### Adapter Layer
- gRPC handler (inbound): HybridSearch, NodeSearch, EdgeSearch, CommunitySearch
- gRPC clients (outbound): downstream service calls
- Repository adapters: database implementations

### Infrastructure Layer
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel)

## 3. gRPC API

RPCs: HybridSearch, NodeSearch, EdgeSearch, CommunitySearch

## 4. Cross-Service Dependencies

graphiti-knowledge (embedding, reranking), graphiti-store (search primitives), Redis (cache), NATS (cache invalidation)

## 5. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs
- **Traces**: OTel spans for every usecase method
- **Logs**: Structured JSON via slog with request_id, tenant_id
- **Health**: gRPC health check + HTTP /healthz on port 9095

## 6. Multi-Tenancy

Tenant isolation via gRPC metadata `x-tenant-id` → propagated to all DB queries.

---

> **Next Steps**: Decompose this TDD into individual FEAT/ARCH specs in `specs/features/` and `specs/architecture/` before implementation.
