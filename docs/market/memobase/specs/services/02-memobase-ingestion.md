# 02 — Memobase Ingestion Service

> **gRPC**: 9041 | **Health**: 9091

---

## 1. Purpose

Xử lý data ingestion pipeline: nhận blobs (chat/doc/summary), quản lý buffer zone, trigger flush khi buffer đầy. Tách biệt hot-path ingestion khỏi cold-path LLM processing.

---

## 2. Clean Architecture

```
services/memobase-ingestion/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Blob, BufferZone, BufferStatus
│   │   ├── value_object.go     # BlobType, TokenSize, BufferThreshold
│   │   ├── event.go            # BufferReadyEvent, BlobInsertedEvent
│   │   └── errors.go           # ErrInvalidBlobType, ErrBufferProcessing
│   ├── usecase/
│   │   ├── insert_blob.go      # Store blob + add to buffer + check capacity
│   │   ├── flush_buffer.go     # Mark buffers processing + emit NATS event
│   │   ├── get_buffer_capacity.go
│   │   ├── get_blob.go
│   │   ├── delete_blob.go
│   │   ├── mark_buffer_done.go # Called by engine completion event
│   │   ├── port/
│   │   │   ├── input.go        # InsertBlobUseCase, FlushBufferUseCase interfaces
│   │   │   └── output.go       # BlobRepository, BufferRepository, EventPublisher
│   │   └── dto/
│   │       ├── request.go      # InsertBlobRequest, FlushBufferRequest
│   │       └── response.go     # BlobResponse, BufferCapacityResponse
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go      # memobase.ingestion.v1.IngestionService impl
│   │   │   └── mapper.go       # Proto ↔ Domain
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── blob_repo.go      # general_blobs table
│   │   │       └── buffer_repo.go    # buffer_zones table
│   │   ├── event/
│   │   │   ├── publisher.go    # NATS: memobase.buffer.ready
│   │   │   └── subscriber.go   # NATS: memobase.engine.completed/failed
│   │   └── tokenizer/
│   │       └── tiktoken.go     # Token counting adapter
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Entities

```go
type BlobType string
const (
    BlobTypeChat    BlobType = "chat"
    BlobTypeDoc     BlobType = "doc"
    BlobTypeSummary BlobType = "summary"
)

type Blob struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    ProjectID       string
    BlobType        BlobType
    BlobData        json.RawMessage  // OpenAI-compatible messages
    AdditionalFields json.RawMessage
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type BufferStatus string
const (
    BufferIdle       BufferStatus = "idle"
    BufferProcessing BufferStatus = "processing"
    BufferDone       BufferStatus = "done"
    BufferFailed     BufferStatus = "failed"
)

type BufferZone struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    ProjectID  string
    BlobID     uuid.UUID
    BlobType   BlobType
    TokenSize  int
    Status     BufferStatus
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

---

## 4. Use Case Flow: InsertBlob

```
Client → Gateway → gRPC InsertBlob(user_id, blob_type, blob_data)
                        │
                        ▼
              ┌──── InsertBlobUseCase ────────┐
              │ 1. Validate blob_type         │
              │ 2. Calculate token_size       │
              │    (tiktoken gpt-4o encoder)  │
              │ 3. Store Blob in DB           │
              │ 4. Insert BufferZone entry    │
              │    (status = "idle")          │
              │ 5. Check buffer capacity:     │
              │    SUM(token_size) WHERE idle  │
              │    > max_buffer_token_size?    │
              │ 6. If full → emit BufferReady │
              └──────────────────────────────┘
                        │
                        ▼ (if buffer full)
              NATS: memobase.buffer.ready
              {user_id, project_id, buffer_ids[], blob_type}
                        │
                        ▼
              memobase-engine (subscriber)
```

## 5. Use Case Flow: FlushBuffer

```
Client → Gateway → gRPC FlushBuffer(user_id, buffer_type)
                        │
                        ▼
              ┌──── FlushBufferUseCase ──────┐
              │ 1. Query idle buffers        │
              │ 2. If sync=true:             │
              │    a. Mark "processing"       │
              │    b. gRPC → engine.Process   │
              │    c. Await result            │
              │    d. Mark "done" / "failed"  │
              │ 3. If sync=false:            │
              │    a. Mark "processing"       │
              │    b. Emit BufferReady (NATS) │
              └─────────────────────────────┘
```

---

## 6. NATS Events

| Subject | Payload | Publisher | Subscriber |
|---------|---------|-----------|------------|
| `memobase.buffer.ready` | `{user_id, project_id, buffer_ids[], blob_type}` | ingestion | engine |
| `memobase.engine.completed` | `{user_id, project_id, buffer_ids[], event_id}` | engine | ingestion |
| `memobase.engine.failed` | `{user_id, project_id, buffer_ids[], error}` | engine | ingestion |

---

## 7. Buffer State Machine

```
        InsertBlob
            │
            ▼
    ┌──────────────┐
    │     idle     │ ← Initial state
    └──────┬───────┘
           │ FlushBuffer / auto-flush
           ▼
    ┌──────────────┐
    │  processing  │ ← Engine working
    └──┬───────┬───┘
       │       │
       ▼       ▼
  ┌────────┐ ┌────────┐
  │  done  │ │ failed │ ← Retry: reset to idle
  └────────┘ └────────┘
```

---

## 8. Configuration

```yaml
ingestion:
  grpc:
    port: 9041
  health:
    port: 9091
  buffer:
    max_token_size: 1024          # Flush threshold
    auto_flush_interval: 3600s    # Idle auto-flush
    persistent_blobs: false       # Delete after processing
    max_process_token_size: 16384 # Max tokens per flush batch
  tokenizer:
    model: "gpt-4o"
  nats:
    url: "nats://nats:4222"
    stream: "memobase"
  database:
    url: "${DATABASE_URL}"
    pool_size: 25
    max_overflow: 15
```
