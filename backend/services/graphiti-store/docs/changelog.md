---
id: DOC-S07
service: graphiti-store
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-store — Changelog

All notable changes to this service will be documented in this file.

## [0.1.0] — 2026-05-10 — Initial Release

### Added

- **Domain Layer**: EntityNode, EpisodicNode, CommunityNode, SagaNode types
- **Domain Layer**: EntityEdge with bi-temporal model (valid_at, invalid_at, expired_at)
- **Domain Layer**: GraphDriver composite interface (Strategy pattern)
- **Domain Layer**: 7 repository interfaces (Node, Edge, Community, Search, Index, Bulk, Transaction)
- **Neo4j Driver**: Full CRUD for all 4 node types
- **Neo4j Driver**: Bi-temporal edge CRUD with InvalidateEdge and time range queries
- **Neo4j Driver**: Cosine similarity search, BM25 fulltext search, BFS graph traversal
- **Neo4j Driver**: Atomic SaveBulk with transaction rollback support
- **Neo4j Driver**: Index management (vector, fulltext, composite, range)
- **gRPC Service**: 15 RPCs on port :9024
- **Infrastructure**: Viper config, Wire DI, OTel tracing, Prometheus metrics
- **Operations**: Health checks (/healthz, /readyz) on :9097, Dockerfile

### Architecture

- Clean Architecture 4-layer: domain → usecase → adapter → infra
- Driver Factory pattern for pluggable graph backends
- Multi-tenant isolation via group_id scoping on all queries
