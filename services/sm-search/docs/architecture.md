---
id: DOC-S03
service: sm-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-search — Service Architecture

> **Group**: Supermemory | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/sm-search/
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
| PostgreSQL + pgvector | SQL | Vector + fulltext search |
| Bifrost (LLM) | HTTP | RAG completion |
| Redis | Cache | Search result cache |

## Storage

PostgreSQL + pgvector (vector + fulltext search), Redis (result cache)
