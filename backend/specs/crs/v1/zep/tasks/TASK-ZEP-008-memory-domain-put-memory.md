# TASK-ZEP-008 — services/memory-service: Domain Upgrade & PutMemory

**Task ID:** TASK-ZEP-008  
**Wave:** 3 (Memory Core)  
**Solution:** [SOL-ZEP-002](../solutions/SOL-ZEP-002-Memory-Message-Context-Assembly.md)  
**Depends on:** TASK-ZEP-006 (thread use cases — UpsertSession)  
**Ước tính:** 4h  
**Priority:** Critical — sub-200ms SLA

---

## Mục tiêu

Nâng cấp `services/memory-service/` (Zep domain) với:
1. `Message` entity (thêm `RoleType` enum 6 values, `TokenCount`, `Metadata`)
2. PostgreSQL schema migration cho `messages` table
3. `PutMemory` use case (batch insert + NATS publish, sub-200ms)
4. Session lifecycle guard (gọi Thread Service UpsertSession)

---

## Input Context

- **Existing code:** `services/memory-service/internal/domain/zep/` có `ZepMessage` cơ bản — sẽ UPGRADE
- **Không xóa code cũ** — extend existing entities
- **NATS topic:** `zep.memory.messages.ingested` (publish sau INSERT)
- **Target:** p95 latency < 200ms cho PutMemory

---

## Công việc cụ thể

### 1. Nâng cấp `services/memory-service/internal/domain/zep/message.go`

```go
// RoleType định nghĩa 6 valid role types cho message
type RoleType string

const (
    RoleTypeNone      RoleType = "norole"    // default cho unknown
    RoleTypeSystem    RoleType = "system"
    RoleTypeAssistant RoleType = "assistant"
    RoleTypeUser      RoleType = "user"
    RoleTypeFunction  RoleType = "function"
    RoleTypeTool      RoleType = "tool"
)

// ValidRoleTypes map dùng để validate incoming role_type
var ValidRoleTypes = map[RoleType]bool{
    RoleTypeNone: true, RoleTypeSystem: true, RoleTypeAssistant: true,
    RoleTypeUser: true, RoleTypeFunction: true, RoleTypeTool: true,
}

// Message entity (UPGRADE từ ZepMessage)
type Message struct {
    UUID        string
    SessionID   string
    ProjectUUID string
    Role        string         // free text (e.g. "gpt-4o", "human")
    RoleType    RoleType       // typed enum — validation + routing
    Content     string
    TokenCount  int            // NEW: token count of Content
    Metadata    map[string]any // NEW: JSONB metadata
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time     // soft delete
}

// Sentinel errors
var (
    ErrSessionEnded   = errors.New("session has been ended")
    ErrMessageNotFound = errors.New("message not found")
)
```

### 2. Tạo SQL Migration `services/memory-service/migrations/002_upgrade_messages_table.sql`

```sql
-- Thêm các cột mới vào messages table (không xóa cột cũ)
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS role_type   VARCHAR NOT NULL DEFAULT 'norole',
    ADD COLUMN IF NOT EXISTS token_count INT     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS metadata    JSONB   DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS deleted_at  TIMESTAMPTZ;

-- role_type validation constraint
ALTER TABLE messages ADD CONSTRAINT messages_role_type_check
    CHECK (role_type IN ('norole','system','assistant','user','function','tool'));

-- Indexes
CREATE INDEX IF NOT EXISTS msg_session_idx ON messages(session_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS msg_role_type_idx ON messages(role_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS msg_metadata_gin ON messages USING GIN(metadata);
```

### 3. Tạo `MessageRepository` interface

```go
// services/memory-service/internal/domain/zep/message_repository.go
type MessageRepository interface {
    BatchInsert(ctx context.Context, messages []Message) error
    GetLastN(ctx context.Context, sessionID, projectUUID string, n int) ([]Message, error)
    GetByUUID(ctx context.Context, uuid string) (*Message, error)
    List(ctx context.Context, sessionID, projectUUID string, limit, offset int) ([]Message, int, error)
    SoftDelete(ctx context.Context, uuid string) error
    MergePatchMetadata(ctx context.Context, uuid string, patch map[string]any) error
    DeleteBySession(ctx context.Context, sessionID, projectUUID string) error
}
```

### 4. Implement `BatchInsert` trong PostgreSQL repo

```go
// services/memory-service/internal/infra/postgres/message_repo.go
// BatchInsert: dùng unnest() hoặc pgx.CopyFrom để bulk insert
// Target: < 50ms cho 50 messages
func (r *MessageRepo) BatchInsert(ctx context.Context, msgs []Message) error {
    // Single transaction, unnest() approach:
    // INSERT INTO messages (uuid, session_id, project_uuid, role, role_type, content, token_count, metadata)
    // SELECT unnest($1::uuid[]), unnest($2::varchar[]), ...
}
```

### 5. Tạo `PutMemory` Use Case

**`services/memory-service/internal/usecase/zep/put_memory.go`**

```go
type PutMemoryUseCase struct {
    threadClient ThreadServiceClient // gRPC → zep-thread (UpsertSession)
    msgRepo      MessageRepository
    publisher    EventPublisher      // NATS
    tokenizer    TokenizerPort       // count tokens
}

// Execute MUST complete in sub-200ms:
// 1. UpsertSession via Thread Service gRPC (in-process bufconn ~2ms)
// 2. Lifecycle guard: EndedAt != nil → return ErrSessionEnded
// 3. Validate + normalize RoleType (default "norole" if unknown)
// 4. Batch INSERT messages (PostgreSQL, single tx ~10ms)
// 5. Publish NATS event (non-blocking goroutine, ~1ms to dispatch)
// 6. Return immediately
func (uc *PutMemoryUseCase) Execute(ctx context.Context, req PutMemoryRequest) error { ... }
```

### 6. Tests

- `TestPutMemory_Success_Sub200ms`: ingest 10 messages, assert latency < 200ms
- `TestPutMemory_EndedSession_Returns400`: put to ended session → ErrSessionEnded
- `TestPutMemory_InvalidRoleType_DefaultsToNorole`: unknown role_type → "norole"
- `TestPutMemory_PublishesNATSEvent`: after insert, NATS event published
- `TestBatchInsert_50Messages_Under50ms`: benchmark test

---

## Acceptance Criteria

- [ ] `go build ./services/memory-service/...` không có lỗi
- [ ] Migration không có lỗi, không drop existing data
- [ ] `RoleType` validation: "tool" → valid; "robot" → default "norole"
- [ ] PutMemory vào ended session → HTTP 400
- [ ] BatchInsert 10 messages → single PostgreSQL transaction
- [ ] NATS event published sau INSERT (không phải trước)
- [ ] PutMemory p95 < 200ms (benchmark test)

---

## Files tạo ra

```
services/memory-service/
├── internal/
│   ├── domain/zep/
│   │   ├── message.go              (UPGRADE)
│   │   └── message_repository.go   (NEW interface)
│   ├── usecase/zep/
│   │   └── put_memory.go           (NEW)
│   └── infra/postgres/
│       ├── message_repo.go         (UPGRADE — thêm BatchInsert, GetLastN)
│       └── message_repo_test.go
└── migrations/
    └── 002_upgrade_messages_table.sql
```

## Sau khi hoàn thành

Chạy: `go build ./services/memory-service/... && go test ./services/memory-service/...`
