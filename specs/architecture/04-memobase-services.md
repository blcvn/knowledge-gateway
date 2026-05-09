# Memobase Services

---

# Memobase Ingestion Service

> **Service**: `memobase-ingestion` | **gRPC Port**: 9031

## 1. Responsibility

Blob ingestion (ChatBlob, DocBlob, SummaryBlob), Buffer Zone FSM, flush trigger.

## 2. gRPC API

```protobuf
service MemobaseIngestionService {
  rpc InsertBlob(InsertBlobRequest) returns (InsertBlobResponse);
  rpc GetBufferStatus(BufferStatusRequest) returns (BufferStatus);
  rpc FlushBuffer(FlushBufferRequest) returns (FlushResponse);
  rpc DeleteBlob(DeleteBlobRequest) returns (Empty);
}
```

## 3. Buffer Zone FSM

```
IDLE → (token_sum >= 1024 OR idle > 1h) → PROCESSING → DONE → IDLE
PROCESSING → (fail) → FAILED → (retry) → PROCESSING
```

## 4. NATS: `memobase.buffer.ready` → memobase-engine

---

# Memobase Engine Service

> **Service**: `memobase-engine` | **gRPC Port**: 9032

## 1. Responsibility

Core LLM processing: profile extraction (YOLO merge), event summarization. **Fixed 3 LLM calls per flush**.

## 2. Pipeline

```
LLM #1: entry_chat_summary → LLM #2: extract_topics → LLM #3: merge_yolo
Output: updated_profiles[] + new_events[]
```

## 3. Profile Schema

```go
type Profile struct {
    ID, UserID, ProjectID string
    Topic, SubTopic, Content string
    UpdatedAt time.Time
}
```

## 4. NATS Events

- `memobase.engine.completed` → memobase-context
- `memobase.profile.changed` → memobase-context (cache invalidate)
- `memobase.event.created` → vnp-event

---

# Memobase Context Service

> **Service**: `memobase-context` | **gRPC Port**: 9033

## 1. Responsibility

Read-path context assembly. Prompt-ready context < 100ms p95.

## 2. gRPC API

```protobuf
service MemobaseContextService {
  rpc GetContext(GetContextRequest) returns (ContextResponse);
  rpc GetProfiles(GetProfilesRequest) returns (ProfilesResponse);
  rpc SearchProfiles(SearchProfilesRequest) returns (SearchProfilesResponse);
}
```

## 3. Caching: Redis TTL 20min, invalidated on `memobase.profile.changed`.
