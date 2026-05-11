---
id: DOC-S03
service: sm-document
version: 1.1.0
status: Deprecated
created: 2026-05-09
updated: 2026-05-10
---

# sm-document — Service Architecture

> **Group**: Supermemory | **Pattern**: 4-layer Clean Architecture

> **🚨 DEPRECATION NOTICE**: This architecture is obsolete. The service has been merged into `sm-engine` (Ref: [ARCH-007-merge-sm-engine]).


## Layer Structure

```
services/sm-document/
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
| PostgreSQL/SurrealDB | SQL | Document + chunk persistence |
| NATS JetStream | Publisher | `sm.document.created/deleted` |

## Storage

PostgreSQL/SurrealDB (documents, chunks), pgvector (chunk embeddings)
