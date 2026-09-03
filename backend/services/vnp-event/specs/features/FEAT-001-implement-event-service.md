---
id: FEAT-001
title: Implement vnp-event — Cross-Engine Event Timeline Service
service: vnp-event
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-002
linked_tdd: TDD-vnp-event
---

## Mục Tiêu

Implement vnp-event as a Go microservice providing cross-domain event timeline storage, semantic + temporal search, and event gist summarization. Subscribes to events from all 6 engines and builds a unified timeline per user.

## Bối Cảnh Nghiệp Vụ

vnp-event is the temporal memory layer — it collects events from Cognee, Graphiti, Memobase, OpenViking, Zep, and Supermemory into a unified timeline. The search-hub queries vnp-event for the `events` component of `memory.recall()`.

## Scope

### In Scope
- Domain entities: UserEvent, EventSource (enum), EventGist (from tdd.md §2)
- Usecase: CreateEvent, SearchEvents, SearchGists, GetTimeline, FilterByTags
- gRPC handlers: VnpEventService (from tdd.md §3)
- PostgreSQL + pgvector: events table with embedding column
- NATS subscribers: listen to all 6 engine completion events
- Bi-temporal support: valid_at / invalid_at for temporal reasoning
- Event gist generation: batch summarization via LLM

### Out of Scope
- LLM model selection (uses EmbeddingService port)
- Real-time streaming (Phase 2)

## Thiết Kế Kỹ Thuật

### API Contract (from tdd.md §3)
```protobuf
service VnpEventService {
  rpc CreateEvent(CreateEventRequest) returns (Event);
  rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
  rpc SearchGists(SearchGistsRequest) returns (SearchGistsResponse);
  rpc GetTimeline(GetTimelineRequest) returns (TimelineResponse);
  rpc FilterByTags(FilterByTagsRequest) returns (FilterResponse);
}
```

### Data Model
- `user_events`: id, user_id, tenant_id, source (enum), content, tags[], embedding (vector(1536)), created_at, valid_at, invalid_at
- `event_gists`: id, event_ids[], summary, embedding (vector(1536)), created_at

### NATS Subscriptions (from tdd.md §4)
| Subject | Source Engine | Mapping |
|---------|-------------|---------|
| `cognee.pipeline.completed` | Cognee | Create event(source=COGNEE) |
| `graphiti.pipeline.completed` | Graphiti | Create event(source=GRAPHITI) |
| `memobase.pipeline.flush` | Memobase | Create event(source=MEMOBASE) |
| `ov.storage.resource.parsed` | OpenViking | Create event(source=OPENVIKING) |
| `zep.core.memory.enriched` | Zep | Create event(source=ZEP) |
| `sm.engine.document.saved` | Supermemory | Create event(source=SUPERMEMORY) |

### Internal Architecture
```
services/vnp-event/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   ├── event.go        # UserEvent, EventSource enum
│   │   │   └── gist.go         # EventGist
│   │   ├── repository/
│   │   │   ├── event_repo.go
│   │   │   └── gist_repo.go
│   │   └── errors.go
│   ├── usecase/
│   │   ├── event_service.go    # CRUD + temporal search
│   │   ├── gist_service.go     # Batch summarization
│   │   └── timeline_service.go # Timeline assembly
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   └── nats/subscriber.go  # 6 engine event listeners
│   └── infra/
│       ├── config/config.go
│       └── persistence/
│           ├── pg_event.go     # pgvector queries
│           └── pg_gist.go
└── go.mod
```

## Acceptance Criteria

- [ ] AC-1: `go build ./cmd/server/` compiles without errors
- [ ] AC-2: CreateEvent → stores with embedding → searchable by semantic similarity
- [ ] AC-3: SearchEvents with temporal range → returns events where valid_at ∈ [start, end]
- [ ] AC-4: GetTimeline → returns events sorted by valid_at for a user
- [ ] AC-5: FilterByTags → intersects tag arrays correctly
- [ ] AC-6: NATS subscriber creates events from all 6 engine sources
- [ ] AC-7: EventGist summarizes batches of events (≥5 events per gist)

## Test Requirements
- **Unit tests:** EventService.Search (temporal + semantic), GistService.Summarize
- **Integration tests:** NATS event → CreateEvent → SearchEvents round-trip
- **Minimum coverage:** 80%

## Definition of Done
- [ ] Code implements all Acceptance Criteria
- [ ] Unit tests pass, coverage ≥ 80%
- [ ] `docs/changelog.md` updated
- [ ] No lint errors
