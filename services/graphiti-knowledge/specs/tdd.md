---
id: TDD-graphiti-knowledge
title: Technical Design — graphiti-knowledge
service: graphiti-knowledge
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-09
group: Graphiti
---

# Technical Design — graphiti-knowledge

> **Group**: Graphiti | **gRPC Port**: 9023 | **Health Port**: 9096

## 1. Service Overview

LLM-intensive processing engine: entity extraction, edge extraction, entity/edge resolution (deduplication), community detection, embedding generation, cross-encoder reranking. All AI interactions routed through Bifrost gateway.

## 2. Clean Architecture Layers

### Domain Layer
- ExtractedEntity, ExtractedEdge, Resolution, DuplicateDecision
- Stateless: no persistent storage, pure LLM processing
- Prompt templates: extract_entities, resolve_entities, extract_edges, summarize_community

### Usecase Layer
Business logic orchestration. Imports domain only.

### Adapter Layer
- gRPC handler (inbound): ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, GenerateEmbedding, Rerank, UpdateCommunity
- gRPC clients (outbound): downstream service calls
- Repository adapters: database implementations

### Infrastructure Layer
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel)

## 3. gRPC API

RPCs: ExtractEntities, ResolveEntities, ExtractEdges, ResolveEdges, GenerateEmbedding, Rerank, UpdateCommunity

## 4. Cross-Service Dependencies

Bifrost (LLM gateway), graphiti-store (graph reads for resolution), NATS (publish entity.resolved, community.rebuilt)

## 5. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs
- **Traces**: OTel spans for every usecase method
- **Logs**: Structured JSON via slog with request_id, tenant_id
- **Health**: gRPC health check + HTTP /healthz on port 9096

## 6. Multi-Tenancy

Tenant isolation via gRPC metadata `x-tenant-id` → propagated to all DB queries.

---

> **Next Steps**: Decompose this TDD into individual FEAT/ARCH specs in `specs/features/` and `specs/architecture/` before implementation.
