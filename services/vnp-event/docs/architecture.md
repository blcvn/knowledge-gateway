---
id: DOC-S03
service: vnp-event
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-event — Service Architecture

> **Group**: Platform | **Pattern**: 4-layer Clean Architecture | **Event Consumer/Provider**

## Layer Structure

```
services/vnp-event/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # TimelineEvent, EventType, EntityRef
│   │   ├── value_object.go        # TimeRange, EventSource, EventAction
│   │   ├── event.go               # EventCreated
│   │   └── errors.go              # ErrEventNotFound
│   ├── usecase/
│   │   ├── create_event.go        # Store new timeline event
│   │   ├── query_timeline.go      # Temporal + semantic timeline query
│   │   ├── entity_events.go       # Entity-scoped event query
│   │   ├── search_events.go       # Semantic search via pgvector
│   │   └── port/
│   │       ├── input.go           # EventUseCase, TimelineUseCase
│   │       └── output.go          # EventRepo, EmbedderClient, CacheClient
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC server
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       └── event_repo.go  # pgvector-enabled event storage
│   │   ├── cache/
│   │   │   └── redis_cache.go     # Recent events hot cache
│   │   └── event/
│   │       └── nats_subscriber.go # Subscribe to all engine events (6 subjects)
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

## Design Decisions

- **NATS fan-in**: Single service subscribes to events from all 6 engines, creating a unified timeline
- **pgvector**: Enables semantic event search without requiring a separate vector database
- **Redis hot cache**: Most recent events (last 1h per tenant) cached for fast timeline queries
- **Append-only**: Events are immutable after creation — no updates, only inserts
