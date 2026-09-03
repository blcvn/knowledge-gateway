---
id: DOC-S07
service: graphiti-search
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-search — Changelog

All notable changes to this service will be documented in this file.

## [0.1.0] — 2026-05-10 — Initial Release

### Added

- **Domain Layer**: SearchQuery, SearchResult, RankedResult, SearchFilter types
- **Domain Layer**: SearchMethod (cosine, BM25, BFS) and RerankerType (RRF, MMR, CrossEncoder, NodeDistance, EpisodeMentions) enums
- **Usecase Layer**: Hybrid search orchestrator (parallel search → merge → rerank → cache)
- **Usecase Layer**: 5 pluggable reranker implementations (Reranker interface)
- **Adapter Layer**: graphiti-store gRPC client for search primitives delegation
- **Adapter Layer**: Redis cache with TTL + NATS-driven cache invalidation
- **gRPC Service**: HybridSearch, NodeSearch, EdgeSearch, CommunitySearch RPCs on :9022
- **Infrastructure**: Viper config, Wire DI, OTel tracing, Prometheus metrics
- **Operations**: Health checks on :9095, Dockerfile

### Architecture

- Stateless query service — no own persistent storage
- Multi-strategy reranking pipeline (chainable)
- Graceful degradation: search works without Redis (uncached)
