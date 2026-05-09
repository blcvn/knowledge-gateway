---
id: DOC-S03
service: sm-auth
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-auth — Service Architecture

> **Group**: Supermemory | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/sm-auth/
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
| PostgreSQL/SurrealDB | SQL | API keys (SHA-256), orgs, roles |

## Storage

PostgreSQL/SurrealDB (API keys hashed, organizations, roles)
