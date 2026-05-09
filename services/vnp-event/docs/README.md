---
id: DOC-S01
service: vnp-event
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Platform Team
---

# vnp-event

> **Group**: Platform | **gRPC Port**: 9041 | **Health Port**: 9101 | **Origin**: Cross-domain

## Purpose

Cross-domain event timeline service. Stores temporal events from **all engines** (Memobase profile events, Graphiti episodes, Cognee pipeline events, Supermemory memory events) with **semantic + temporal search** capabilities. Provides a unified event history with bi-temporal validity and tag-based filtering.

### Business Capability

- **Unified Event Storage**: Events from Memobase, Graphiti, Cognee, Supermemory, and OpenViking
- **Semantic Search**: Vector-based event search using pgvector embeddings
- **Temporal Search**: Time-range queries with valid_at/invalid_at bi-temporal windows
- **Event Gists**: Summarized event clusters for efficient context retrieval
- **Tag-Based Filtering**: Multi-tag filter for scoping events
- **Timeline View**: Chronological event stream per user/tenant

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC, NATS JetStream
- **Database**: PostgreSQL + pgvector (events + vector search), Redis (recent events cache)
- **Architecture**: 4-layer Clean Architecture

## API Surface

```protobuf
service VnpEventService {
  rpc CreateEvent(CreateEventRequest) returns (Event);
  rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
  rpc SearchGists(SearchGistsRequest) returns (SearchGistsResponse);
  rpc GetTimeline(GetTimelineRequest) returns (TimelineResponse);
  rpc FilterByTags(FilterByTagsRequest) returns (FilterResponse);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/events` | Create event |
| POST | `/v1/events/search` | Semantic event search |
| GET | `/v1/events/timeline` | Chronological timeline |
| POST | `/v1/events/gists` | Search event summaries |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL + pgvector | SQL | Event storage + vector search |
| Redis | Cache | Recent events (last 100 per user) |
| Bifrost | HTTP | Event embedding generation |
| memobase-engine | NATS sub | `memobase.event.created` |
| graphiti-ingestion | NATS sub | `graphiti.episode.ingested` |
| sm-memory | NATS sub | `sm.memory.created` |
| vnp-search-hub | gRPC | Fan-out search target |

## Links

- [API](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md) · [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — Platform
