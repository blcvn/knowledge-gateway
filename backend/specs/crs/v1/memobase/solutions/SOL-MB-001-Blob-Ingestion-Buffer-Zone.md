# Solution: SOL-MB-001 — Blob Ingestion & Buffer Zone Service

**CR:** [CR-MB-001](../CR-MB-001-Blob-Ingestion-Buffer-Zone.md)  
**Wave:** 1 (Data Layer)  
**Priority:** Critical  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/memobase-ingestion` hoàn chỉnh theo Clean Architecture trong monorepo `vnp-memory`. Service này là **hot-path** duy nhất nhận raw blobs từ client, quản lý Buffer Zone FSM, và kích hoạt LLM pipeline qua NATS.

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Buffer Zone FSM thiếu | Implement `buffer_zones` table + status enum (`idle→processing→done/failed`) |
| Không có token counting | Tích hợp `pkg/tokenizer` (tiktoken-go, gpt-4o encoder) |
| Auto-flush thiếu | Background `AutoFlushScheduler` goroutine + token threshold check trong `InsertBlobUseCase` |
| Multi-blob type | Polymorphic `BlobData` JSONB với type discriminator |
| Concurrency race | Status-based lock: `UPDATE buffer_zones SET status='processing' WHERE status='idle'` |
| NATS integration | Publish `memobase.buffer.ready`, subscribe `engine.completed/failed` |

---

## 2. Vị trí trong Codebase

```
vnp-memory/
└── services/
    └── memobase-ingestion/          ← [NEW] Toàn bộ service mới
        ├── cmd/server/main.go
        ├── internal/
        │   ├── domain/
        │   ├── usecase/
        │   ├── adapter/
        │   └── infra/
        ├── api/proto/
        └── Makefile
```

### Đăng ký vào Monolith Bootstrap

```go
// apps/memory/internal/bootstrap/memobase.go  ← MODIFY

func bootstrapMemobaseIngestion(ctx context.Context, cfg *config.Config, registry *bus.InProcessRegistry) error {
    // 1. Khởi tạo dependencies
    db := infra.NewPostgresDB(cfg.Database)
    natsConn := infra.GetNATSConn(ctx)
    tokenizer := tokenizer.NewTiktokenTokenizer("gpt-4o")

    // 2. Khởi tạo repositories
    blobRepo := postgres.NewBlobRepository(db)
    bufferRepo := postgres.NewBufferRepository(db)

    // 3. Khởi tạo use cases
    insertBlobUC := usecase.NewInsertBlobUseCase(blobRepo, bufferRepo, tokenizer, nats.NewPublisher(natsConn), cfg.Ingestion)
    flushBufferUC := usecase.NewFlushBufferUseCase(bufferRepo, nats.NewPublisher(natsConn))
    autoFlushSched := usecase.NewAutoFlushScheduler(bufferRepo, flushBufferUC, cfg.Ingestion.Buffer)

    // 4. Khởi tạo gRPC handler
    handler := grpchandler.NewIngestionHandler(insertBlobUC, flushBufferUC, /* ... */)

    // 5. Đăng ký vào InProcessRegistry (bufconn)
    server := grpc.NewServer()
    ingestionv1.RegisterIngestionServiceServer(server, handler)
    registry.Register("memobase-ingestion", server, bufconn.Listen(1024*1024))

    // 6. Khởi động scheduler
    go autoFlushSched.Run(ctx)

    return nil
}
```

---

## 3. Database Migration

**File:** `services/memobase-ingestion/internal/infra/migrations/001_init.up.sql`

```sql
-- Bảng lưu raw blobs (ChatBlob, DocBlob, SummaryBlob)
CREATE TABLE IF NOT EXISTS general_blobs (
    id                UUID        NOT NULL,
    project_id        VARCHAR     NOT NULL,
    user_id           UUID        NOT NULL,
    blob_type         VARCHAR     NOT NULL CHECK (blob_type IN ('chat', 'doc', 'summary')),
    blob_data         JSONB       NOT NULL,
    additional_fields JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_general_blobs_user ON general_blobs(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_general_blobs_type ON general_blobs(user_id, project_id, blob_type);

-- Bảng quản lý Buffer Zone FSM
CREATE TABLE IF NOT EXISTS buffer_zones (
    id         UUID        NOT NULL,
    project_id VARCHAR     NOT NULL,
    user_id    UUID        NOT NULL,
    blob_id    UUID        NOT NULL,
    blob_type  VARCHAR     NOT NULL,
    token_size INTEGER     NOT NULL DEFAULT 0,
    status     VARCHAR     NOT NULL DEFAULT 'idle'
                           CHECK (status IN ('idle', 'processing', 'done', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (blob_id, project_id) REFERENCES general_blobs(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_buffer_zones_user_status ON buffer_zones(user_id, project_id, blob_type, status);
CREATE INDEX IF NOT EXISTS idx_buffer_zones_stale ON buffer_zones(updated_at) WHERE status = 'idle';
```

---

## 4. Domain Layer

### 4.1 Blob Domain (`internal/domain/blob.go`)

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type BlobType string

const (
    BlobTypeChat    BlobType = "chat"
    BlobTypeDoc     BlobType = "doc"
    BlobTypeSummary BlobType = "summary"
)

// ChatMessage tuân theo OpenAI message format
type ChatMessage struct {
    Role    string `json:"role"`    // user | assistant | system
    Content string `json:"content"`
}

// BlobData là interface cho từng loại blob
type BlobData interface {
    BlobType() BlobType
    Validate() error
}

type ChatBlobData struct {
    Messages []ChatMessage `json:"messages"`
}

func (c *ChatBlobData) BlobType() BlobType { return BlobTypeChat }
func (c *ChatBlobData) Validate() error {
    for _, m := range c.Messages {
        switch m.Role {
        case "user", "assistant", "system":
        default:
            return ErrInvalidBlobRole
        }
    }
    return nil
}

type DocBlobData struct {
    Content string `json:"content"`
}

func (d *DocBlobData) BlobType() BlobType { return BlobTypeDoc }
func (d *DocBlobData) Validate() error {
    if d.Content == "" {
        return ErrEmptyBlobContent
    }
    return nil
}

type SummaryBlobData struct {
    Summary string `json:"summary"`
}

func (s *SummaryBlobData) BlobType() BlobType { return BlobTypeSummary }
func (s *SummaryBlobData) Validate() error {
    if s.Summary == "" {
        return ErrEmptyBlobContent
    }
    return nil
}

type Blob struct {
    ID               uuid.UUID
    UserID           uuid.UUID
    ProjectID        string
    BlobType         BlobType
    BlobData         BlobData
    AdditionalFields map[string]any
    CreatedAt        time.Time
}
```

### 4.2 Buffer Domain (`internal/domain/buffer.go`)

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type BufferStatus string

const (
    BufferStatusIdle       BufferStatus = "idle"
    BufferStatusProcessing BufferStatus = "processing"
    BufferStatusDone       BufferStatus = "done"
    BufferStatusFailed     BufferStatus = "failed"
)

// FSM transitions:
// idle → processing (FlushBuffer acquires lock)
// processing → done (engine completed)
// processing → failed (engine failed, retryable)
// failed → idle (explicit retry reset)

type BufferZone struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    ProjectID string
    BlobID    uuid.UUID
    BlobType  BlobType
    TokenSize int
    Status    BufferStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (b *BufferZone) CanFlush() bool {
    return b.Status == BufferStatusIdle
}
```

---

## 5. Use Case Layer

### 5.1 Insert Blob (`internal/usecase/insert_blob.go`)

Luồng xử lý:
1. Validate blob format (role check cho ChatBlob)
2. Persist blob vào `general_blobs` (PostgreSQL JSONB)
3. Đếm tokens qua `pkg/tokenizer` (tiktoken gpt-4o)
4. Insert `buffer_zones` record với `status=idle`
5. Query tổng idle tokens của user
6. Nếu vượt threshold (default 1024): trigger async auto-flush

**Điểm quan trọng:** Auto-flush là non-blocking (`go uc.autoFlush(...)`), không ảnh hưởng latency của InsertBlob response.

### 5.2 Flush Buffer (`internal/usecase/flush_buffer.go`)

Luồng xử lý với concurrency control:
1. Query tất cả `idle` buffer entries cho user/project
2. **Atomic status lock**: `UPDATE SET status='processing' WHERE status='idle' RETURNING id`
   - PostgreSQL atomic UPDATE đảm bảo chỉ 1 goroutine acquire lock
   - Nếu không có rows updated → đã có flush khác đang chạy → return nil
3. Nếu `sync=true`: gRPC call trực tiếp tới `memobase-engine.ProcessBlobs()`
4. Nếu `sync=false`: publish `memobase.buffer.ready` NATS event
5. Subscribe `memobase.engine.completed` → update status `done`
6. Subscribe `memobase.engine.failed` → update status `failed`

### 5.3 Auto-Flush Scheduler (`internal/usecase/auto_flush.go`)

```go
// Chạy trong background goroutine, kiểm tra mỗi 5 phút
// Query: SELECT DISTINCT user_id, project_id FROM buffer_zones
//        WHERE status='idle' AND updated_at < NOW() - interval '1 hour'
// → trigger FlushBuffer cho mỗi user
```

---

## 6. Adapter Layer

### 6.1 PostgreSQL Repository

**BlobRepository:**
- `Save(ctx, blob) → (Blob, error)` — INSERT với JSONB serialization
- `GetByID(ctx, blobID, projectID) → (Blob, error)` — SELECT by composite PK
- `Delete(ctx, blobID, projectID) → error` — DELETE (cascade xóa buffer entry)
- `DeleteByUser(ctx, userID, projectID) → error` — bulk delete (triggered bởi user deletion)

**BufferRepository:**
- `Save(ctx, entry) → (BufferZone, error)` — INSERT idle entry
- `AcquireProcessingLock(ctx, userID, projectID, blobType) → ([]BufferZone, error)` — atomic UPDATE SET status='processing' WHERE status='idle'
- `GetTotalIdleTokens(ctx, userID, projectID) → (int, error)` — SUM token_size WHERE status='idle'
- `MarkDone(ctx, bufferIDs []uuid.UUID, projectID string) → error`
- `MarkFailed(ctx, bufferIDs []uuid.UUID, projectID string, errMsg string) → error`
- `GetUsersWithStaleIdleBuffers(ctx, idleTimeout time.Duration) → ([]UserProject, error)` — cho auto-flush scheduler
- `GetCapacity(ctx, userID, projectID, blobType string) → (CapacityInfo, error)` — cho API

### 6.2 NATS Event Handler

```go
// subscriber.go — lắng nghe events từ engine
func (s *Subscriber) HandleEngineCompleted(msg *nats.Msg) {
    var payload EngineCompletedPayload
    json.Unmarshal(msg.Data, &payload)
    
    // Cập nhật status → done
    s.bufferRepo.MarkDone(ctx, payload.BufferIDs, payload.ProjectID)
    
    // Nếu persistent_chat_blobs=false: xóa raw blobs
    if !s.config.PersistentChatBlobs {
        blobIDs := extractBlobIDs(payload.BufferIDs)
        s.blobRepo.DeleteBatch(ctx, blobIDs, payload.ProjectID)
    }
    
    msg.Ack()
}

func (s *Subscriber) HandleEngineFailed(msg *nats.Msg) {
    var payload EngineFailedPayload
    json.Unmarshal(msg.Data, &payload)
    s.bufferRepo.MarkFailed(ctx, payload.BufferIDs, payload.ProjectID, payload.Error)
    msg.Ack()
}
```

---

## 7. gRPC Server Registration

Service đăng ký vào `InProcessRegistry` với `bufconn` transport. Trong monolith mode, gateway giao tiếp với ingestion service qua in-memory gRPC (zero network latency).

**Proto file:** `api/proto/memobase/ingestion/v1/ingestion.proto`

```protobuf
syntax = "proto3";
package memobase.ingestion.v1;
option go_package = "github.com/vnp/memory/services/memobase-ingestion/api/proto/memobase/ingestion/v1";

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
```

---

## 8. Configuration

**File:** `services/memobase-ingestion/configs/config.yaml`

```yaml
server:
  grpc_port: 9041
  health_port: 9091

buffer:
  max_chat_blob_token_size: 1024  # MEMOBASE_MAX_CHAT_BLOB_BUFFER_TOKEN_SIZE
  flush_interval: 3600s            # idle timeout → auto-flush
  max_concurrent_flush: 5          # bulkhead semaphore
  persistent_chat_blobs: false     # MEMOBASE_PERSISTENT_CHAT_BLOBS

tokenizer:
  model: "gpt-4o"

database:
  url: "${DATABASE_URL}"
  pool_size: 25
  max_overflow: 10

nats:
  url: "${NATS_URL}"
  stream: "memobase"
  subjects:
    publish_buffer_ready: "memobase.buffer.ready"
    subscribe_engine_completed: "memobase.engine.completed"
    subscribe_engine_failed: "memobase.engine.failed"
    subscribe_user_deleted: "memobase.admin.user.deleted"
```

---

## 9. Phụ thuộc mới

| Package | Mục đích | Go Module |
|---|---|---|
| `tiktoken-go` | Token counting | `github.com/pkoukk/tiktoken-go` |
| `nats.go` | NATS JetStream client | `github.com/nats-io/nats.go` |
| `pgx/v5` | PostgreSQL driver | `github.com/jackc/pgx/v5` |
| `google/uuid` | UUID generation | `github.com/google/uuid` |
| `sony/gobreaker` | Circuit breaker | `github.com/sony/gobreaker` |

---

## 10. Testing Strategy

### Unit Tests
- `TestInsertBlobUseCase_ValidChatBlob` — happy path
- `TestInsertBlobUseCase_InvalidRole` → `ErrInvalidBlobRole`
- `TestInsertBlobUseCase_AutoFlushTriggered` — khi total tokens > threshold
- `TestFlushBufferUseCase_ConcurrencyLock` — 2 goroutines flush cùng user, chỉ 1 wins
- `TestFlushBufferUseCase_NothingToFlush` — không có idle entries → return nil
- `TestAutoFlushScheduler_StaleBuffers` — mock clock, verify flush triggered

### Integration Tests
- `TestBlobInsertAndFlushE2E` — insert blob → check buffer_zones → flush → verify NATS event
- `TestPersistentChatBlobsMode` — `persistent_chat_blobs=false` → blob deleted after flush

---

## 11. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| tiktoken-go licensing | Thấp | MIT license, production-ready |
| Database lock contention khi nhiều users flush đồng thời | Trung bình | Index trên `(user_id, project_id, status)` + connection pooling |
| NATS message loss khi engine restart | Thấp | JetStream WorkQueue retention + Ack-based delivery |
| Auto-flush scheduler overhead | Thấp | Query chỉ `WHERE status='idle'` với index → fast |
| `persistent_chat_blobs=false` + engine failure | Trung bình | Chỉ xóa blobs SAU KHI `memobase.engine.completed` (không phải failed) |
