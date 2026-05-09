---
id: DOC-S03
service: sm-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-memory — Service Architecture

> **Group**: Supermemory | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/sm-memory/
├── cmd/server/main.go
├── internal/
│   ├── domain/           # Layer 1: entities, value objects, events, errors
│   ├── usecase/          # Layer 2: business logic + port interfaces
│   ├── adapter/          # Layer 3: gRPC handlers, DB repos, NATS pub/sub
│   └── infra/            # Layer 4: config, server, wire DI
```

## External Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL/SurrealDB | SQL | Memory + relation persistence |
| Bifrost (LLM) | HTTP | Fact extraction from documents |
| NATS JetStream | Publisher | `sm.memory.created` → sm-search, sm-profile |

## Storage

PostgreSQL/SurrealDB (memories, relations), pgvector (memory embeddings)
