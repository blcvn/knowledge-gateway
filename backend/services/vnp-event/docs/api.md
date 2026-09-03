---
id: DOC-S02
service: vnp-event
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-event — API Reference

## gRPC Service Definition

```protobuf
service VNPEventService {
  rpc CreateEvent(CreateEventRequest) returns (Event);
  rpc QueryTimeline(TimelineQuery) returns (TimelineResponse);
  rpc GetEntityEvents(EntityEventsRequest) returns (TimelineResponse);
  rpc SearchEvents(SearchEventsRequest) returns (SearchEventsResponse);
}
```

## RPC: QueryTimeline

```protobuf
message TimelineQuery {
  string group_id = 1;
  google.protobuf.Timestamp from = 2;
  google.protobuf.Timestamp to = 3;
  repeated string engines = 4;   // Filter by source engine
  string entity_ref = 5;         // Filter by entity
  int32 limit = 6;
  string cursor = 7;
}

message TimelineResponse {
  repeated Event events = 1;
  string next_cursor = 2;
}

message Event {
  string id = 1;
  string entity_ref = 2;
  string action = 3;
  string details = 4;
  string source_engine = 5;
  google.protobuf.Timestamp created_at = 6;
}
```

## NATS Events Subscribed

| Subject | Source | Action |
|---------|--------|--------|
| `graphiti.episode.ingested` | graphiti-ingestion | Create timeline event |
| `cognee.cognify.completed` | cognee-cognify | Create timeline event |
| `memobase.memory.flushed` | memobase-ingestion | Create timeline event |
| `zep.memory.stored` | zep-memory | Create timeline event |
| `sm.document.created` | sm-document | Create timeline event |
| `ov.session.committed` | ov-session | Create timeline event |
