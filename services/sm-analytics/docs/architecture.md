---
id: DOC-S03
service: sm-analytics
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-analytics — Service Architecture

> **Group**: Supermemory | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/sm-analytics/
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
| PostgreSQL/SurrealDB | SQL | Usage metrics, token tracking |

## Storage

PostgreSQL/SurrealDB (usage_metrics, token_usage, storage_metrics)
