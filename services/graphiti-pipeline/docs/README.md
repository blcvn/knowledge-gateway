# graphiti-pipeline — Episodic KG Ingestion + Knowledge Service

> **Service**: `graphiti-pipeline` | **gRPC Port**: 9021 | **Health**: 9094  
> **Origin**: Consolidated from graphiti-ingestion + graphiti-knowledge  
> **Status**: Proposed | **Version**: 0.1.0

---

## Purpose

Unified ingestion and knowledge extraction service for the Graphiti episodic knowledge graph engine. Orchestrates a multi-step saga pipeline: episode ingestion → entity extraction → entity resolution → edge extraction → edge resolution → embedding generation → community update. Features bi-temporal data model (valid_at/invalid_at) and compensating actions for saga rollback.

## Business Capability

- **Episode Ingestion**: Ingest conversational episodes with speaker attribution
- **Saga Pipeline**: 6-step LLM-powered knowledge extraction with compensating rollback
- **Entity/Edge Management**: Extract, resolve, deduplicate entities and relationships
- **Bi-temporal Model**: Track entity/edge validity windows (valid_at, invalid_at)
- **Community Detection**: Automatic graph community structure updates
- **Bulk Ingestion**: Parallel episode processing with per-group_id serialization

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| RPC | gRPC (2 services: GraphitiIngestionService + GraphitiKnowledgeService) |
| Database | PostgreSQL 17 + pgvector |
| Graph DB | Neo4j 5+ (entities, edges, communities) |
| Async | NATS JetStream |
| LLM | Bifrost multi-provider (entity/edge extraction, resolution) |

## Quick Start

```bash
cd services/graphiti-pipeline
go run cmd/server/main.go
# gRPC: :9021 | Health: :9094
```

## Links

- [Architecture](./architecture.md)
- [Changelog](./changelog.md)

## Owner

Graphiti Engine Team
