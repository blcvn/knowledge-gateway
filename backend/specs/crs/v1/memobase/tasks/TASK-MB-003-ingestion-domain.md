# TASK-MB-003 — `services/memobase-ingestion` Domain & DB Setup

**Wave:** 1 (Data Layer — song song với TASK-MB-002)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-MB-001 (pkg/tokenizer, pkg/config), TASK-MB-002 (DB foundation migrations phải chạy trước)  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-MB-001 §3, §4](../solutions/SOL-MB-001-Blob-Ingestion-Buffer-Zone.md)  
**Trạng thái:** ✅ Implemented  
**Ghi chú:** memobase-ingestion: 20 .go, buffer domain + pipeline
**Port gRPC:** 9041

---

## Mục tiêu

Tạo `services/memobase-ingestion/` phần đầu: database migrations, domain models (Blob, Buffer Zone FSM), PostgreSQL repositories, và ports/interfaces. Phần này là nền tảng để TASK-MB-004 implement use cases và gRPC.

---

## Cấu trúc thư mục

```
services/memobase-ingestion/
├── cmd/server/main.go               ← Stub (wire up ở TASK-MB-004)
├── api/proto/memobase/ingestion/v1/
│   └── ingestion.proto
├── internal/
│   ├── domain/
│   │   ├── blob.go                  # BlobType, BlobData interface, ChatBlobData, DocBlobData, SummaryBlobData, Blob
│   │   ├── buffer.go                # BufferStatus FSM, BufferZone
│   │   └── errors.go                # ErrInvalidBlobRole, ErrEmptyBlobContent, ErrNothingToFlush
│   ├── usecase/
│   │   └── port/
│   │       ├── input.go             # InsertBlobRequest, FlushBufferRequest
│   │       └── output.go            # BlobRepository, BufferRepository, EventPublisher interfaces
│   └── adapter/
│       └── repository/postgres/
│           ├── blob_repo.go
│           └── buffer_repo.go
└── internal/infra/
    └── migrations/
        ├── 002_ingestion.up.sql
        └── 002_ingestion.down.sql
```

---

## 1. Database Migration

**File: `internal/infra/migrations/002_ingestion.up.sql`**

```sql
-- Runs AFTER 001_foundation.up.sql (requires users table)

-- Raw blobs storage
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
CREATE INDEX IF NOT EXISTS idx_general_blobs_user  ON general_blobs(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_general_blobs_type  ON general_blobs(user_id, project_id, blob_type);
CREATE INDEX IF NOT EXISTS idx_general_blobs_time  ON general_blobs(created_at DESC);

-- Buffer Zone FSM
CREATE TABLE IF NOT EXISTS buffer_zones (
    id         UUID        NOT NULL,
    project_id VARCHAR     NOT NULL,
    user_id    UUID        NOT NULL,
    blob_id    UUID        NOT NULL,
    blob_type  VARCHAR     NOT NULL,
    token_size INTEGER     NOT NULL DEFAULT 0,
    status     VARCHAR     NOT NULL DEFAULT 'idle'
                           CHECK (status IN ('idle', 'processing', 'done', 'failed')),
    error_msg  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (blob_id, project_id) REFERENCES general_blobs(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_buffer_zones_user_status ON buffer_zones(user_id, project_id, blob_type, status);
CREATE INDEX IF NOT EXISTS idx_buffer_zones_stale       ON buffer_zones(updated_at) WHERE status = 'idle';
CREATE INDEX IF NOT EXISTS idx_buffer_zones_user_tokens ON buffer_zones(user_id, project_id, status, token_size);
```

---

## 2. Domain Layer

**File: `internal/domain/blob.go`**

```go
package domain

type BlobType string
const (
    BlobTypeChat    BlobType = "chat"
    BlobTypeDoc     BlobType = "doc"
    BlobTypeSummary BlobType = "summary"
)

type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type BlobData interface {
    BlobType() BlobType
    Validate() error
}

type ChatBlobData struct {
    Messages []ChatMessage `json:"messages"`
}

func (c *ChatBlobData) BlobType() BlobType { return BlobTypeChat }
func (c *ChatBlobData) Validate() error {
    if len(c.Messages) == 0 { return ErrEmptyBlobContent }
    for _, m := range c.Messages {
        switch m.Role {
        case "user", "assistant", "system": // valid
        default: return ErrInvalidBlobRole
        }
    }
    return nil
}

type DocBlobData struct {
    Content string `json:"content"`
}
func (d *DocBlobData) BlobType() BlobType { return BlobTypeDoc }
func (d *DocBlobData) Validate() error {
    if d.Content == "" { return ErrEmptyBlobContent }
    return nil
}

type SummaryBlobData struct {
    Summary string `json:"summary"`
}
func (s *SummaryBlobData) BlobType() BlobType { return BlobTypeSummary }
func (s *SummaryBlobData) Validate() error {
    if s.Summary == "" { return ErrEmptyBlobContent }
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

// DeserializeBlobData — từ JSONB raw bytes → typed BlobData
func DeserializeBlobData(raw json.RawMessage, blobType BlobType) (BlobData, error)
```

**File: `internal/domain/buffer.go`**

```go
package domain

type BufferStatus string
const (
    BufferStatusIdle       BufferStatus = "idle"
    BufferStatusProcessing BufferStatus = "processing"
    BufferStatusDone       BufferStatus = "done"
    BufferStatusFailed     BufferStatus = "failed"
)

// FSM valid transitions:
// idle → processing (FlushBuffer acquires lock)
// processing → done (engine completed)
// processing → failed (engine failed, retryable)
// failed → idle (manual retry reset)

type BufferZone struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    ProjectID string
    BlobID    uuid.UUID
    BlobType  BlobType
    TokenSize int
    Status    BufferStatus
    ErrorMsg  string
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (b *BufferZone) CanFlush() bool { return b.Status == BufferStatusIdle }
func (b *BufferZone) IsTerminal() bool {
    return b.Status == BufferStatusDone
}
```

---

## 3. Port Interfaces

**File: `internal/usecase/port/output.go`**

```go
package port

type BlobRepository interface {
    Save(ctx context.Context, blob *domain.Blob) (*domain.Blob, error)
    GetByID(ctx context.Context, blobID uuid.UUID, projectID string) (*domain.Blob, error)
    Delete(ctx context.Context, blobID uuid.UUID, projectID string) error
    DeleteByUser(ctx context.Context, userID uuid.UUID, projectID string) error
    GetForProcessing(ctx context.Context, bufferIDs []uuid.UUID, projectID string) ([]*domain.Blob, error)
}

type BufferRepository interface {
    Save(ctx context.Context, entry *domain.BufferZone) (*domain.BufferZone, error)

    // Atomic status lock: UPDATE SET status='processing' WHERE status='idle' RETURNING *
    AcquireProcessingLock(ctx context.Context, userID uuid.UUID, projectID string, blobType domain.BlobType) ([]*domain.BufferZone, error)

    GetTotalIdleTokens(ctx context.Context, userID uuid.UUID, projectID string) (int, error)
    GetBufferCapacity(ctx context.Context, userID uuid.UUID, projectID string, blobType domain.BlobType) (*CapacityInfo, error)
    MarkDone(ctx context.Context, bufferIDs []uuid.UUID, projectID string) error
    MarkFailed(ctx context.Context, bufferIDs []uuid.UUID, projectID string, errMsg string) error
    GetUsersWithStaleIdleBuffers(ctx context.Context, idleTimeout time.Duration) ([]*UserProject, error)
}

type CapacityInfo struct {
    NumBlobs  int
    NumTokens int
}

type UserProject struct {
    UserID    uuid.UUID
    ProjectID string
}

type EventPublisher interface {
    PublishBufferReady(ctx context.Context, payload BufferReadyPayload) error
}

type BufferReadyPayload struct {
    UserID    string   `json:"user_id"`
    ProjectID string   `json:"project_id"`
    BufferIDs []string `json:"buffer_ids"`
    BlobType  string   `json:"blob_type"`
}
```

---

## 4. PostgreSQL Repository Implementations

**File: `internal/adapter/repository/postgres/blob_repo.go`**

```go
type PostgresBlobRepository struct {
    db *pgx.Conn  // hoặc *pgxpool.Pool
}

func (r *PostgresBlobRepository) Save(ctx context.Context, blob *domain.Blob) (*domain.Blob, error) {
    // Serialize BlobData to JSONB
    blobDataJSON, err := json.Marshal(blob.BlobData)
    if err != nil { return nil, err }

    if blob.ID == uuid.Nil { blob.ID = uuid.New() }

    const q = `INSERT INTO general_blobs (id, project_id, user_id, blob_type, blob_data, additional_fields)
               VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb)
               ON CONFLICT (id, project_id) DO NOTHING
               RETURNING created_at`
    row := r.db.QueryRow(ctx, q, blob.ID, blob.ProjectID, blob.UserID, string(blob.BlobType), blobDataJSON, blob.AdditionalFields)
    row.Scan(&blob.CreatedAt)
    return blob, nil
}

func (r *PostgresBlobRepository) GetForProcessing(ctx context.Context, bufferIDs []uuid.UUID, projectID string) ([]*domain.Blob, error) {
    // JOIN general_blobs WITH buffer_zones WHERE buffer_zones.id = ANY($1) AND buffer_zones.project_id = $2
    // Returns blobs ordered by created_at ASC
}
```

**File: `internal/adapter/repository/postgres/buffer_repo.go`**

```go
func (r *PostgresBufferRepository) AcquireProcessingLock(ctx context.Context, userID uuid.UUID, projectID string, blobType domain.BlobType) ([]*domain.BufferZone, error) {
    // CRITICAL: Atomic UPDATE với RETURNING — đảm bảo concurrency safety
    const q = `
        WITH acquired AS (
            UPDATE buffer_zones
            SET status = 'processing', updated_at = NOW()
            WHERE (user_id, project_id, blob_type, status) = ($1, $2, $3, 'idle')
            RETURNING id, project_id, user_id, blob_id, blob_type, token_size, status, created_at, updated_at
        )
        SELECT * FROM acquired`
    // Nếu không có rows → đã có flush khác đang chạy → return empty slice (not error)
}

func (r *PostgresBufferRepository) GetTotalIdleTokens(ctx context.Context, userID uuid.UUID, projectID string) (int, error) {
    const q = `SELECT COALESCE(SUM(token_size), 0)
               FROM buffer_zones
               WHERE user_id=$1 AND project_id=$2 AND status='idle'`
    var total int
    r.db.QueryRow(ctx, q, userID, projectID).Scan(&total)
    return total, nil
}

func (r *PostgresBufferRepository) GetUsersWithStaleIdleBuffers(ctx context.Context, idleTimeout time.Duration) ([]*port.UserProject, error) {
    const q = `SELECT DISTINCT user_id, project_id FROM buffer_zones
               WHERE status='idle' AND updated_at < NOW() - $1::interval`
    rows, _ := r.db.Query(ctx, q, idleTimeout.String())
    // Scan into []UserProject
}
```

---

## 5. Proto Definition

**File: `api/proto/memobase/ingestion/v1/ingestion.proto`**

```protobuf
syntax = "proto3";
package memobase.ingestion.v1;
option go_package = "vnp-memory/services/memobase-ingestion/api/gen/ingestion/v1;ingestionv1";

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
  string blob_type        = 1;  // "chat" | "doc" | "summary"
  string user_id          = 2;
  string project_id       = 3;
  bytes  blob_data        = 4;  // JSON-encoded BlobData
  map<string, string> additional_fields = 5;
}

message InsertBlobResponse {
  string blob_id = 1;
  bool   flush_triggered = 2;
}

message FlushBufferRequest {
  string user_id    = 1;
  string project_id = 2;
  string blob_type  = 3;
  bool   sync       = 4;  // if true: gRPC call to engine; if false: NATS publish
}

message FlushBufferResponse {
  int32 blobs_flushed = 1;
  bool  skipped = 2;   // true if another flush already running
}

message GetBufferCapacityResponse {
  int32 num_blobs  = 1;
  int32 num_tokens = 2;
}
```

---

## Unit Tests

```
TestChatBlobData_Validate_ValidRoles    → user/assistant/system → no error
TestChatBlobData_Validate_InvalidRole   → "model" role → ErrInvalidBlobRole
TestChatBlobData_Validate_Empty         → [] → ErrEmptyBlobContent
TestDocBlobData_Validate_Valid          → non-empty → no error
TestDocBlobData_Validate_Empty          → "" → ErrEmptyBlobContent
TestSummaryBlobData_Validate_Empty      → "" → ErrEmptyBlobContent
TestBufferZone_CanFlush_Idle            → status=idle → true
TestBufferZone_CanFlush_Processing      → status=processing → false
TestDeserializeBlobData_Chat            → {"messages":[{"role":"user","content":"hi"}]} → ChatBlobData
TestDeserializeBlobData_UnknownType     → "image" → error
TestBlobRepository_Save                 → mock DB → INSERT called with correct JSON
TestBlobRepository_Save_SetsUUID        → blob.ID == uuid.Nil → new UUID assigned
TestBufferRepository_AcquireLock_Atomic → 2 concurrent calls → only 1 gets rows
TestBufferRepository_GetTotalIdleTokens → 3 idle entries 100+200+300 → 600
TestBufferRepository_GetStaleBuffers    → 2 users with stale idle → both returned
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

buf generate services/memobase-ingestion/
go build ./services/memobase-ingestion/...
go test ./services/memobase-ingestion/internal/domain/... -v -count=1
go test ./services/memobase-ingestion/internal/adapter/... -v -count=1
```

---

## Ghi chú triển khai

- Migration `002_ingestion` chạy SAU `001_foundation` (TASK-MB-002) — cần `users` table
- `AcquireProcessingLock` là single SQL statement atomic — không cần transaction wrapper
- `pgxpool.Pool` (không phải `pgx.Conn`) cho production để connection pooling
- `DeserializeBlobData`: switch trên blobType → unmarshal vào concrete type
