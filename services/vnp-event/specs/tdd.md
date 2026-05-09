---
id: TDD-vnp-event
title: Technical Design — vnp-event
service: vnp-event
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Platform
---

# Technical Design — vnp-event

> **Group**: Platform | **gRPC Port**: 9041 | **Health Port**: 9101

## 1. Service Overview

Cross-domain event timeline: stores temporal events from all 6 engines (Cognee, Graphiti, Memobase, OpenViking, Zep, Supermemory). Semantic + temporal search with bi-temporal validity and event gist summarization.

## 2. Domain Layer

- **UserEvent**: id (UUID), user_id, tenant_id, source (MEMOBASE|GRAPHITI|COGNEE|OPENVIKING|ZEP|SUPERMEMORY), content (TEXT), tags (TEXT[]), embedding (VECTOR(1536)), created_at, valid_at, invalid_at
- **EventSource**: enum — MEMOBASE, GRAPHITI, COGNEE, OPENVIKING, ZEP, SUPERMEMORY
- **EventGist**: id, events_ids[], summary, embedding, created_at

## 3. gRPC API

```protobuf
service VnpEventService {
  rpc CreateEvent(CreateEventRequest) returns (Event);
  rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
  rpc SearchGists(SearchGistsRequest) returns (SearchGistsResponse);
  rpc GetTimeline(GetTimelineRequest) returns (TimelineResponse);
  rpc FilterByTags(FilterByTagsRequest) returns (FilterResponse);
}
```

## 4. NATS Events (Subscriptions)

| Subject | Source Engine | Mapping |
|---------|-------------|---------|
| `memobase.event.created` | Memobase | Profile change events |
| `graphiti.episode.ingested` | Graphiti | Episode timeline events |
| `cognee.pipeline.completed` | Cognee | KG build completion |
| `ov.session.committed` | OpenViking | Session commit events |
| `zep.graph.fact.created` | Zep | Fact extraction events |
| `sm.memory.created` | Supermemory | Memory creation events |

## 5. Data Model

### Tables
- `user_events`: id(PK UUID), user_id, tenant_id, source, content(TEXT), tags(TEXT[]), embedding(VECTOR(1536)), created_at, valid_at, invalid_at
- `event_gists`: id(PK), event_ids(UUID[]), summary(TEXT), embedding(VECTOR(1536)), created_at

### Key Indexes
- `idx_event_tenant_user` (tenant_id, user_id, created_at DESC) — timeline queries
- `idx_event_source` (tenant_id, source, created_at) — per-engine filter
- `idx_event_tags` GIN (tags) — tag filter
- `idx_event_embedding` HNSW — semantic search
- `idx_event_temporal` (valid_at, invalid_at) — bi-temporal queries

## 6. Storage

- **PostgreSQL + pgvector**: Primary event storage + vector search
- **Redis**: Last 100 events per (tenant_id, user_id) for fast timeline

## 7. Observability

- **Metrics**: events_created_total (by source), search_latency, timeline_latency
- **Traces**: OTel spans for CreateEvent, SearchEvents, GetTimeline
- **Health**: gRPC + HTTP /healthz on port 9101

---

> **Next Steps**: FEAT-001 (Event CRUD), FEAT-002 (Semantic Search), FEAT-003 (Event Gist Summarization), ARCH-001 (Multi-engine NATS subscriptions)
