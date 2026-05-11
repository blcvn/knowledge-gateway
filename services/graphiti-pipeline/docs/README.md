---
id: DOC-S01
service: graphiti-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
owner: VNP Memory — Graphiti Team
---

# graphiti-pipeline

> **Group**: Graphiti (Episodic KG) | **gRPC Port**: 9021 | **Health Port**: 9094  
> **Origin**: Consolidated from graphiti-ingestion + graphiti-knowledge  
> **Status**: Active | **Version**: 0.1.0

## Purpose

Unified episodic knowledge ingestion and LLM-powered extraction service for the Graphiti temporal knowledge graph engine. This single binary consolidates the ingestion pipeline orchestrator and knowledge processing engine, converting cross-service gRPC hops into local function calls for reduced latency and simplified deployment.

### Business Capability

- **Episode Ingestion**: Accepts episodic data (messages, JSON, text, fact triples) and orchestrates the full extraction → resolution → persistence pipeline
- **Saga Orchestration**: 6-step pipeline with compensating actions for failure recovery, per-group serialization for consistency
- **Entity Extraction & Resolution**: LLM-powered entity identification and semantic deduplication via Bifrost AI gateway
- **Edge Extraction & Resolution**: Temporal fact triple extraction with contradiction detection and bi-temporal invalidation
- **Embedding Generation**: Vector embeddings for entities, edges, and queries via configurable embedding providers
- **Community Detection**: Label propagation + hierarchical LLM summarization for entity clusters
- **Cross-Encoder Reranking**: Neural reranking support for search result quality improvement
- **Bulk Ingestion**: Streaming bulk episode processing with cross-episode deduplication
- **Bi-temporal Data Model**: Track entity/edge validity windows (`valid_at`, `invalid_at`, `expired_at`, `created_at`)

## Tech Stack

| Component | Technology |
|-----------|-----------|
| **Language** | Go 1.23+ |
| **Framework** | gRPC server (2 services: IngestionService + KnowledgeService) |
| **Database** | PostgreSQL 17 + pgvector (saga state, episode metadata, embeddings) |
| **Graph DB** | Neo4j 5.x (entities, edges, communities) via graphiti-store |
| **Async** | NATS JetStream (event publishing) |
| **LLM** | Bifrost multi-provider gateway (entity/edge extraction, resolution, summarization) |
| **Architecture** | 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra) |
| **DI** | Google Wire (compile-time dependency injection) |
| **Observability** | OTel traces + Prometheus metrics + slog JSON |
| **Resilience** | Circuit breaker (gobreaker), retry with exponential backoff, bulkhead |

## Quick Start

```bash
# From monorepo root
cd services/graphiti-pipeline
go run cmd/server/main.go

# Or with Docker Compose
docker compose up graphiti-pipeline

# Health check
grpcurl -plaintext localhost:9094 grpc.health.v1.Health/Check
curl http://localhost:9094/healthz
```

## Pipeline (Saga) Flow

```
IngestEpisode(content, group_id)
  │
  ├─1─► [LOCAL] ExtractEntities(content)        — LLM via Bifrost
  ├─2─► [LOCAL] ResolveEntities(extracted)       — Search + LLM dedup
  ├─3─► [LOCAL] ExtractEdges(episode, nodes)     — LLM fact extraction
  ├─4─► [LOCAL] ResolveEdges(extracted_edges)    — Contradiction detection
  ├─5─► [LOCAL] GenerateEmbeddings(nodes, edges) — Vector generation
  ├─6─► [gRPC]  graphiti-store.SaveBulk(...)     — Graph persistence
  ├─7─► [LOCAL] UpdateCommunity(affected)        — Community recalculation
  └─8─► [NATS]  Publish graphiti.episode.ingested
```

### Compensating Actions

| Step Failed | Compensation |
|-------------|-------------|
| ExtractEntities | Mark episode FAILED, retry with backoff |
| ResolveEntities | Discard resolution, keep extracted entities |
| SaveBulk | Rollback partial writes via graphiti-store |
| UpdateCommunity | Queue for async retry (non-blocking) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| `graphiti-store` | gRPC (9024) | Graph database CRUD operations, bulk persistence |
| Bifrost | HTTP | LLM inference (entity/edge extraction, resolution, summarization) |
| NATS JetStream | Async | Publish `graphiti.episode.ingested`, `entity.resolved`, `community.rebuilt` |
| PostgreSQL | SQL | Saga state tracking, episode metadata, deduplication |
| Neo4j | Bolt (via graphiti-store) | Entity/edge/community graph storage |

## Links

- [API Reference](./api.md) (DOC-S02)
- [Architecture](./architecture.md) (DOC-S03)
- [Data Model](./data-model.md) (DOC-S04)
- [Configuration](./configuration.md) (DOC-S05)
- [Runbook](./runbook.md) (DOC-S06)
- [Changelog](./changelog.md) (DOC-S07)
- [TDD Spec](../specs/tdd.md)

## Owner

- **Team**: VNP Memory — Graphiti Engine
- **Contact**: TBD
