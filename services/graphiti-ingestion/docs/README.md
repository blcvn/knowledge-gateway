---
id: DOC-S01
service: graphiti-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Graphiti Team
---

# graphiti-ingestion

> **Group**: Graphiti (Episodic KG) | **gRPC Port**: 9021 | **Health Port**: 9094 | **Origin**: Graphiti

## Purpose

Episode lifecycle management and pipeline orchestration via the **Saga pattern**. This service is the entry point for all episodic knowledge ingestion into the Graphiti temporal knowledge graph. It serializes per-group ingestion for consistency and coordinates a multi-step pipeline across `graphiti-knowledge` and `graphiti-store` services.

### Business Capability

- **Episode Ingestion**: Accepts episodic data (messages, JSON, text, fact triples) and orchestrates the full extraction-resolution-persistence pipeline
- **Bulk Ingestion**: Streaming bulk episode ingestion for batch processing scenarios
- **Saga Orchestration**: Multi-step pipeline with compensating actions for failure recovery
- **Per-Group Serialization**: Ensures consistency within a `group_id` by serializing concurrent ingestion requests
- **Status Tracking**: Provides episode processing status queries for async pipeline monitoring

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream consumer/publisher
- **Database**: PostgreSQL (job state, saga state tracking)
- **Graph Database**: Neo4j 5.x (via graphiti-store, indirect)
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire
- **Observability**: OTel traces + Prometheus metrics + slog JSON
- **Resilience**: Circuit breaker (per downstream gRPC client), retry with exponential backoff

## Quick Start

```bash
# From monorepo root
make build-graphiti-ingestion
make run-graphiti-ingestion

# Or with Docker Compose
docker compose up graphiti-ingestion

# Health check
grpcurl -plaintext localhost:9094 grpc.health.v1.Health/Check
curl http://localhost:9094/healthz
```

## Pipeline (Saga) Flow

```
1. Validate + enqueue (per group_id serialization)
2. → graphiti-knowledge.ExtractEntities(content)
3. → graphiti-knowledge.ResolveEntities(extracted, group_id)
4. → graphiti-knowledge.ExtractEdges(episode, resolved_nodes)
5. → graphiti-knowledge.ResolveEdges(extracted_edges, group_id)
6. → graphiti-store.SaveBulk(nodes, edges, episode)
7. → graphiti-knowledge.UpdateCommunity(affected_entities)
8. Emit NATS: graphiti.episode.ingested
```

### Compensating Actions

| Step Failed | Compensation |
|-------------|-------------|
| ExtractEntities | Mark episode FAILED, retry with backoff |
| SaveBulk | Rollback partial writes via graphiti-store |
| UpdateCommunity | Queue for async retry (non-blocking) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| `graphiti-knowledge` | gRPC (9023) | Entity/edge extraction, resolution, community update |
| `graphiti-store` | gRPC (9024) | Bulk graph persistence (nodes, edges, episodes) |
| NATS JetStream | Async | Publish `graphiti.episode.ingested` events |

## Links

- [API Reference](./api.md) (DOC-S02)
- [Architecture](./architecture.md) (DOC-S03)
- [Data Model](./data-model.md) (DOC-S04)
- [Configuration](./configuration.md) (DOC-S05)
- [Runbook](./runbook.md) (DOC-S06)
- [Changelog](./changelog.md) (DOC-S07)
- [Architecture Spec](../../../specs/architecture/03-graphiti-services.md)
- [TDD Spec](../specs/tdd.md)

## Owner

- **Team**: VNP Memory — Graphiti
- **Contact**: TBD
