---
id: TDD-graphiti-store
title: Technical Design — graphiti-store
service: graphiti-store
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-09
group: Graphiti
---

# Technical Design — graphiti-store

> **Group**: Graphiti | **gRPC Port**: 9024 | **Health Port**: 9097

## 1. Service Overview

Graph database abstraction layer with pluggable backends (Neo4j, FalkorDB, Kuzu, Neptune). All graph CRUD operations, transactions, index management, and search primitives.

## 2. Clean Architecture Layers

### Domain Layer
- EntityNode: uuid, name, group_id, summary, name_embedding, labels, attributes
- EpisodicNode: uuid, name, group_id, content, source, valid_at, entity_edges
- CommunityNode: uuid, name, group_id, summary, name_embedding
- SagaNode: uuid, name, group_id, summary, first/last_episode_uuid
- EntityEdge: uuid, name, fact, fact_embedding, valid_at, invalid_at, expired_at, attributes
- GraphDriver interface: Strategy pattern for backend selection

### Usecase Layer
Business logic orchestration. Imports domain only.

### Adapter Layer
- gRPC handler (inbound): SaveNode, GetNode, DeleteNode, SaveEdge, GetEdge, DeleteEdge, SaveBulk, CosineSimilaritySearch, FulltextSearch, BFSSearch, DeleteByGroupID, BuildIndices
- gRPC clients (outbound): downstream service calls
- Repository adapters: database implementations

### Infrastructure Layer
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel)

## 3. gRPC API

RPCs: SaveNode, GetNode, DeleteNode, SaveEdge, GetEdge, DeleteEdge, SaveBulk, CosineSimilaritySearch, FulltextSearch, BFSSearch, DeleteByGroupID, BuildIndices

## 4. Cross-Service Dependencies

Neo4j 5.x (primary), FalkorDB (pluggable), Kuzu (pluggable), Neptune (pluggable)

## 5. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs
- **Traces**: OTel spans for every usecase method
- **Logs**: Structured JSON via slog with request_id, tenant_id
- **Health**: gRPC health check + HTTP /healthz on port 9097

## 6. Multi-Tenancy

Tenant isolation via gRPC metadata `x-tenant-id` → propagated to all DB queries.

---

> **Next Steps**: Decompose this TDD into individual FEAT/ARCH specs in `specs/features/` and `specs/architecture/` before implementation.
