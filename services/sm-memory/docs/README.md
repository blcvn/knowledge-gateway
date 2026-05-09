---
id: DOC-S01
service: sm-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-memory

> **Group**: Supermemory (Adaptive KG Memory) | **gRPC Port**: 9072 | **Health Port**: 12072 | **Origin**: Supermemory

## Purpose

Memory engine with **Ebbinghaus forgetting curve** decay. Fact extraction from documents, knowledge graph construction, and adaptive memory retention based on access patterns.

### Business Capability

- **Memory CRUD**: Create/read/delete memories with relevance scoring
- **Forgetting Curve**: Ebbinghaus-inspired decay based on `access_count × relevance_score`
- **Fact Extraction**: LLM-powered fact extraction from documents
- **Relation Management**: Create semantic relationships between memories

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream
- **Database**: PostgreSQL/SurrealDB (memories, relations), pgvector (memory embeddings)
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire

## Quick Start

```bash
make build-sm-memory
make run-sm-memory
docker compose up sm-memory
```

## API Surface

### gRPC Service

```protobuf
service SmMemoryService {
  rpc CreateMemory(CreateMemoryRequest) returns (Memory);
  rpc GetMemory(GetMemoryRequest) returns (Memory);
  rpc ForgetMemory(ForgetMemoryRequest) returns (Empty);
  rpc ListMemories(ListMemoriesRequest) returns (ListMemoriesResponse);
  rpc CreateRelation(CreateRelationRequest) returns (Relation);
}
```

## NATS Events

Subscribed: `sm.document.created` → extract facts from new documents.
Published: `sm.memory.created` → sm-search (index), sm-profile (update traits).

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL/SurrealDB | SQL | Memory + relation persistence |
| Bifrost (LLM) | HTTP | Fact extraction from documents |
| NATS JetStream | Publisher | `sm.memory.created` → sm-search, sm-profile |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/07-supermemory-services.md)

## Owner

- **Team**: VNP Memory — Supermemory
