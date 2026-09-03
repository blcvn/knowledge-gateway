# Change Request: CR-MB-001 — Blob Ingestion & Buffer Zone Service

**CR ID:** CR-MB-001  
**Component:** `services/memobase-ingestion` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** memobase PRD §5.2 (F-2), SRS §3.3–3.4, specs/services/02-memobase-ingestion.md  
**Maps to Python:** `api_layer/blob.py`, `controllers/blob.py`, `controllers/buffer.py`

---

## 1. Mô tả

Xây dựng **memobase-ingestion** service — hot-path service chịu trách nhiệm nhận raw data blobs từ client, quản lý Buffer Zone (processing queue), và trigger pipeline xử lý bộ nhớ khi buffer đầy.

**3 loại blob:**
- `ChatBlob`: Hội thoại user/assistant theo format OpenAI Compatible
- `DocBlob`: Tài liệu văn bản
- `SummaryBlob`: Tóm tắt người dùng đã có

---

## 2. Vấn đề hiện tại

VNP Memory hiện tại (`services/memory-service`, `apps/memory`) có basic blob ingestion nhưng:
- ❌ Không có **Buffer Zone FSM** (`idle → processing → done/failed`) với concurrency control.
- ❌ Không có **auto-flush** khi buffer vượt token threshold (1024 tokens).
- ❌ Không có **idle-time flush** (auto-flush sau 1 giờ không có data mới).
- ❌ Không có **token-size counting** (tiktoken) để trigger flush.
- ❌ Không có **multi-type blob** support (chỉ có chat, không có doc/summary).
- ❌ Không có **`persistent_chat_blobs`** mode (raw blobs xóa sau processing).
- ❌ Không có **NATS event** `memobase.buffer.ready` để notify engine service.
- ❌ Không có **buffer capacity API** cho client monitoring.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/memobase-ingestion/`

**Port:** `9041` (gRPC internal), **Health:** `9091`

**Cấu trúc (Clean Architecture):**
```
services/memobase-ingestion/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── blob.go            # Blob, ChatBlob, DocBlob, SummaryBlob
│   │   ├── buffer.go          # BufferZone, BufferStatus FSM
│   │   ├── project.go         # Project (tenant context)
│   │   └── errors.go          # ErrInvalidBlob, ErrBufferFull
│   ├── usecase/
│   │   ├── insert_blob.go     # Validate + Store blob + queue in buffer
│   │   ├── flush_buffer.go    # Manual flush trigger
│   │   ├── auto_flush.go      # Token threshold + idle timeout checker
│   │   ├── get_buffer_capacity.go
│   │   ├── get_blob.go        # Fetch blob for engine processing
│   │   ├── delete_blob.go     # Delete raw blob (post-processing)
│   │   └── port/
│   │       ├── input.go       # InsertBlobUseCase, FlushBufferUseCase interfaces
│   │       └── output.go      # BlobRepository, BufferRepository, EventPublisher,
│   │                          #   TokenizerPort, EngineClient
│   ├── adapter/
│   │   ├── grpc/handler.go    # memobase.ingestion.v1.IngestionService impl
│   │   ├── repository/postgres/
│   │   │   ├── blob_repo.go   # general_blobs table CRUD
│   │   │   └── buffer_repo.go # buffer_zones table FSM operations
│   │   ├── client/
│   │   │   └── engine_client.go  # gRPC call to memobase-engine (sync flush)
│   │   └── event/
│   │       ├── publisher.go   # NATS: memobase.buffer.ready
│   │       └── subscriber.go  # NATS: memobase.engine.completed/failed
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
├── api/proto/memobase/ingestion/v1/ingestion.proto
└── Makefile
```

### 3.2. Domain Models

```go
// internal/domain/blob.go

type BlobType string
const (
    BlobTypeChat    BlobType = "chat"
    BlobTypeDoc     BlobType = "doc"
    BlobTypeSummary BlobType = "summary"
)

type Blob struct {
    ID               uuid.UUID
    UserID           uuid.UUID
    ProjectID        string
    BlobType         BlobType
    BlobData         BlobData      // JSONB
    AdditionalFields map[string]any
    CreatedAt        time.Time
}

type ChatMessage struct {
    Role    string `json:"role"`    // user | assistant | system
    Content string `json:"content"`
}

type ChatBlobData struct {
    Messages []ChatMessage `json:"messages"`
}

// DocBlobData: raw text content
// SummaryBlobData: pre-made summary string

// internal/domain/buffer.go

type BufferStatus string
const (
    BufferStatusIdle       BufferStatus = "idle"
    BufferStatusProcessing BufferStatus = "processing"
    BufferStatusDone       BufferStatus = "done"
    BufferStatusFailed     BufferStatus = "failed"
)

type BufferZone struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    BlobID    uuid.UUID
    ProjectID string
    BlobType  BlobType
    TokenSize int
    Status    BufferStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 3.3. Insert Blob Use Case

```go
// internal/usecase/insert_blob.go

func (uc *InsertBlobUseCase) Execute(ctx context.Context, req InsertBlobRequest) (*InsertBlobResult, error) {
    // 1. Validate blob format (role validation for ChatBlob)
    if err := validateBlob(req); err != nil {
        return nil, domain.ErrInvalidBlob
    }

    // 2. Store blob in general_blobs table
    blob, err := uc.blobRepo.Save(ctx, req.toDomain())

    // 3. Count tokens (tiktoken gpt-4o encoder)
    tokenSize, err := uc.tokenizer.CountBlobTokens(blob)

    // 4. Insert BufferZone entry (status=idle)
    bufferEntry, err := uc.bufferRepo.Save(ctx, BufferZone{
        BlobID:    blob.ID,
        UserID:    blob.UserID,
        ProjectID: blob.ProjectID,
        BlobType:  blob.BlobType,
        TokenSize: tokenSize,
        Status:    BufferStatusIdle,
    })

    // 5. Check auto-flush threshold
    totalTokens, _ := uc.bufferRepo.GetTotalIdleTokens(ctx, blob.UserID, blob.ProjectID)
    if totalTokens > uc.config.MaxChatBlobBufferTokenSize {
        // Async background flush (non-blocking)
        go uc.autoFlush(blob.UserID, blob.ProjectID)
    }

    return &InsertBlobResult{BlobID: blob.ID}, nil
}
```

### 3.4. Buffer Zone FSM

```go
// internal/usecase/flush_buffer.go

func (uc *FlushBufferUseCase) Execute(ctx context.Context, userID, projectID string, sync bool) error {
    // 1. Fetch all idle buffer entries for this user
    entries, err := uc.bufferRepo.GetByStatus(ctx, userID, projectID, BufferStatusIdle)
    if len(entries) == 0 {
        return nil  // nothing to flush
    }

    // 2. Status-based concurrency lock: set to "processing"
    //    Prevents duplicate parallel flush
    bufferIDs := extractIDs(entries)
    if err := uc.bufferRepo.UpdateStatus(ctx, bufferIDs, BufferStatusProcessing); err != nil {
        return err  // already processing → skip
    }

    // 3. Publish NATS event OR call engine directly (sync mode)
    if sync {
        // gRPC call to engine (used for test/manual flush)
        err = uc.engineClient.ProcessBlobs(ctx, ProcessBlobsRequest{
            UserID:    userID,
            ProjectID: projectID,
            BufferIDs: bufferIDs,
        })
    } else {
        // Async NATS publish (normal flow)
        err = uc.eventPublisher.PublishBufferReady(ctx, userID, projectID, bufferIDs)
    }

    // 4. On NATS engine.completed: set status "done", optionally delete blobs
    // 5. On NATS engine.failed: set status "failed" (retry-able)
    return err
}
```

### 3.5. Auto-Flush Mechanisms

```go
// internal/usecase/auto_flush.go

// Trigger 1: Token threshold (via insert_blob)
// Trigger 2: Idle-time scheduler (background goroutine)

type AutoFlushScheduler struct {
    interval      time.Duration  // default: 3600s
    flushUseCase  FlushBufferUseCase
}

// Every interval: query all users with idle buffers not updated in >interval
// → trigger async flush for each
func (s *AutoFlushScheduler) Run(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    for {
        select {
        case <-ticker.C:
            users, _ := s.bufferRepo.GetUsersWithStaleIdleBuffers(ctx, s.interval)
            for _, u := range users {
                go s.flushUseCase.Execute(ctx, u.UserID, u.ProjectID, false)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### 3.6. gRPC API

```protobuf
syntax = "proto3";
package memobase.ingestion.v1;

service IngestionService {
    rpc InsertBlob(InsertBlobRequest) returns (InsertBlobResponse);
    rpc GetBlob(GetBlobRequest) returns (GetBlobResponse);
    rpc DeleteBlob(DeleteBlobRequest) returns (DeleteBlobResponse);
    rpc FlushBuffer(FlushBufferRequest) returns (FlushBufferResponse);
    rpc GetBufferCapacity(GetBufferCapacityRequest) returns (GetBufferCapacityResponse);
    rpc GetBlobsForProcessing(GetBlobsForProcessingRequest) returns (GetBlobsForProcessingResponse);
    rpc MarkBufferDone(MarkBufferDoneRequest) returns (MarkBufferDoneResponse);
    rpc MarkBufferFailed(MarkBufferFailedRequest) returns (MarkBufferFailedResponse);
}

message InsertBlobRequest {
    string user_id = 1;
    string project_id = 2;
    string blob_type = 3;      // chat | doc | summary
    bytes blob_data = 4;       // JSON serialized blob content
    map<string, string> additional_fields = 5;
}

message InsertBlobResponse {
    string blob_id = 1;
    int32 token_size = 2;
    bool auto_flush_triggered = 3;
}

message FlushBufferRequest {
    string user_id = 1;
    string project_id = 2;
    string buffer_type = 3;    // chat | doc | summary | all
    bool sync = 4;             // wait for completion
}

message GetBufferCapacityResponse {
    int32 idle_count = 1;
    int32 processing_count = 2;
    int32 total_token_size = 3;
}
```

### 3.7. NATS Events

| Subject | Direction | Payload |
|---|---|---|
| `memobase.buffer.ready` | Publish | `{user_id, project_id, buffer_ids[], blob_type}` |
| `memobase.engine.completed` | Subscribe | `{user_id, project_id, buffer_ids[], event_id}` → mark done |
| `memobase.engine.failed` | Subscribe | `{user_id, project_id, buffer_ids[], error}` → mark failed |
| `memobase.admin.user.deleted` | Subscribe | `{user_id, project_id}` → cascade delete blobs |

### 3.8. REST Endpoints (via Gateway)

```
POST   /api/v1/blobs/insert/{user_id}
  Body: { blob_type, blob_data: { messages: [...] } }
  Response: { data: { id: blob_id, chat_results: [] }, errno: 0 }

GET    /api/v1/blobs/{user_id}/{blob_id}
DELETE /api/v1/blobs/{user_id}/{blob_id}

POST   /api/v1/users/buffer/{user_id}/{buffer_type}   # flush
GET    /api/v1/users/buffer/capacity/{user_id}/{buffer_type}
```

---

## 4. Database Schema

```sql
-- general_blobs
CREATE TABLE general_blobs (
    id          UUID NOT NULL,
    project_id  VARCHAR NOT NULL,
    user_id     UUID NOT NULL,
    blob_type   VARCHAR NOT NULL,     -- chat | doc | summary
    blob_data   JSONB NOT NULL,
    additional_fields JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX idx_general_blobs_user_id_project_id ON general_blobs(user_id, project_id);
CREATE INDEX idx_general_blobs_user_id_blob_type ON general_blobs(user_id, project_id, blob_type);

-- buffer_zones
CREATE TABLE buffer_zones (
    id          UUID NOT NULL,
    project_id  VARCHAR NOT NULL,
    user_id     UUID NOT NULL,
    blob_id     UUID NOT NULL,
    blob_type   VARCHAR NOT NULL,
    token_size  INTEGER NOT NULL DEFAULT 0,
    status      VARCHAR NOT NULL DEFAULT 'idle',  -- idle|processing|done|failed
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (blob_id, project_id) REFERENCES general_blobs(id, project_id) ON DELETE CASCADE
);
CREATE INDEX idx_buffer_zones_user_id_blob_type ON buffer_zones(user_id, project_id, blob_type, status);
```

---

## 5. Configuration

```yaml
ingestion:
  grpc:
    port: 9041
  health:
    port: 9091
  buffer:
    max_chat_blob_token_size: 1024    # auto-flush threshold
    flush_interval: 3600s             # idle-time flush
    max_concurrent_flush: 5           # bulkhead
    persistent_chat_blobs: false      # delete raw after processing
  tokenizer:
    model: "gpt-4o"                   # tiktoken encoder
  nats:
    url: "nats://nats:4222"
    stream: "memobase"
  database:
    url: "${DATABASE_URL}"
    pool_size: 25
```

---

## 6. Acceptance Criteria

- [ ] `POST /api/v1/blobs/insert/{user_id}` với ChatBlob (10 messages) → trả về `blob_id`, `token_size > 0`.
- [ ] Blob với `role` không phải `user|assistant|system` → HTTP 400 Bad Request.
- [ ] Insert blob thứ n khi `total_idle_tokens > 1024` → `auto_flush_triggered: true` trong response.
- [ ] `POST /api/v1/users/buffer/{user_id}/chat` → flush executed, blobs processed.
- [ ] 2 concurrent flush requests cùng user → chỉ 1 thực sự chạy (status-based lock).
- [ ] Engine fail → buffer_zones.status = "failed" (có thể retry lại).
- [ ] `GET /api/v1/users/buffer/capacity/{user_id}/chat` → `{idle_count: N, total_token_size: M}`.
- [ ] `persistent_chat_blobs: false` → sau flush thành công, `general_blobs` entries bị xóa.
- [ ] `DELETE /api/v1/blobs/{user_id}/{blob_id}` → blob removed, buffer entry removed.
- [ ] `DocBlob` và `SummaryBlob` insert → accepted, queued in buffer, processed similarly.
- [ ] Auto-flush scheduler: blob idle > 1 giờ → automatic flush triggered.
