# TASK-ZEP-005 — services/zep-thread: Domain Model & PostgreSQL Schema

**Task ID:** TASK-ZEP-005  
**Wave:** 2 (Core CRUD)  
**Solution:** [SOL-ZEP-001](../solutions/SOL-ZEP-001-Thread-Session-Management.md)  
**Depends on:** TASK-ZEP-002 (pkg/metadata)  
**Ước tính:** 2h  
**Priority:** Critical

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-thread: 6 .go - thread/session domain + schema  
---

## Mục tiêu

Thiết lập domain layer và database schema cho `services/zep-thread/`:
1. `Session` entity với đầy đủ fields (UUID, SessionID, UserID, ProjectUUID, Metadata, EndedAt, DeletedAt)
2. `SessionRepository` interface (port)
3. PostgreSQL schema migration
4. `SessionRepo` implementation (PostgreSQL)

---

## Input Context

- **Clean Architecture:** domain → infra (không import ngược)
- **Uses:** `pkg/metadata` (advisory lock cho MergePatch)
- **Target path:** `services/zep-thread/`
- **Database:** PostgreSQL (same instance as rest of VNP Memory)

---

## Công việc cụ thể

### 1. Tạo `services/zep-thread/internal/domain/session.go`

```go
package domain

import (
    "context"
    "time"
)

// Session represents a conversation thread với lifecycle management
type Session struct {
    UUID        string
    SessionID   string         // user-provided unique ID (e.g. "chat_session_abc")
    UserID      *string        // optional: link to a ZepUser
    ProjectUUID string         // multi-tenant scope
    Metadata    map[string]any // JSONB free-form metadata
    EndedAt     *time.Time     // nil=active; non-nil=session closed
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time     // soft delete
}

// IsEnded returns true nếu session đã được kết thúc
func (s *Session) IsEnded() bool { return s.EndedAt != nil }

// IsDeleted returns true nếu session đã bị soft-delete
func (s *Session) IsDeleted() bool { return s.DeletedAt != nil }

// SessionRepository là port (interface) cho data access
type SessionRepository interface {
    Create(ctx context.Context, s *Session) error
    GetBySessionID(ctx context.Context, sessionID, projectUUID string) (*Session, error)
    GetByUUID(ctx context.Context, uuid string) (*Session, error)
    Update(ctx context.Context, s *Session) error
    MergePatchMetadata(ctx context.Context, sessionID string, patch map[string]any) error
    SoftDelete(ctx context.Context, sessionID, projectUUID string) error
    List(ctx context.Context, projectUUID string, limit, offset int) ([]*Session, int, error)
    ListByUser(ctx context.Context, userID, projectUUID string) ([]*Session, error)
    Search(ctx context.Context, projectUUID, query string) ([]*Session, error)
    Upsert(ctx context.Context, sessionID, projectUUID string, userID *string) (*Session, error)
}

// Sentinel errors
var (
    ErrSessionNotFound     = errors.New("session not found")
    ErrSessionAlreadyEnded = errors.New("session already ended")
    ErrSessionEnded        = errors.New("session has been ended")
    ErrSessionDeleted      = errors.New("session has been deleted")
)
```

### 2. Tạo SQL migration `services/zep-thread/migrations/001_create_sessions.sql`

```sql
CREATE TABLE sessions (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   VARCHAR NOT NULL,
    user_id      VARCHAR,
    project_uuid UUID NOT NULL,
    metadata     JSONB DEFAULT '{}',
    ended_at     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at   TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX session_user_id_idx ON sessions(user_id) WHERE deleted_at IS NULL;
CREATE INDEX session_project_idx ON sessions(project_uuid) WHERE deleted_at IS NULL;
CREATE INDEX session_metadata_gin ON sessions USING GIN(metadata);
CREATE UNIQUE INDEX sessions_session_project_uidx 
    ON sessions(session_id, project_uuid) WHERE deleted_at IS NULL;
CREATE INDEX sessions_fts ON sessions USING GIN(to_tsvector('english', metadata::text));
```

### 3. Tạo `services/zep-thread/internal/infra/postgres/session_repo.go`

Implement `SessionRepository` với:
- `Create`: INSERT với gen_random_uuid()
- `GetBySessionID`: SELECT WHERE session_id=$1 AND project_uuid=$2 AND deleted_at IS NULL
- `Update`: UPDATE SET updated_at=NOW()
- `MergePatchMetadata`: dùng `metadata.MergeJSONBMetadata()` từ pkg/metadata
- `SoftDelete`: UPDATE SET deleted_at=NOW()
- `List`: SELECT với LIMIT/OFFSET, trả về total count
- `ListByUser`: SELECT WHERE user_id=$1
- `Search`: Full-text search trên metadata JSONB
- `Upsert`: INSERT ... ON CONFLICT DO UPDATE SET updated_at=NOW()

### 4. Tạo `services/zep-thread/internal/infra/postgres/session_repo_test.go`

Integration test (dùng testcontainers/postgres hoặc real DB):
- CRUD operations
- Concurrent MergePatch (10 goroutines) → no race condition
- Upsert idempotency
- Search by metadata content

---

## Acceptance Criteria

- [ ] `go build ./services/zep-thread/...` không có lỗi
- [ ] Migration SQL chạy không có lỗi trên PostgreSQL
- [ ] `Session.IsEnded()` trả về đúng
- [ ] `MergePatchMetadata` dùng advisory lock từ `pkg/metadata`
- [ ] `Upsert` với cùng sessionID+projectUUID → không tạo duplicate
- [ ] `SoftDelete` → `GetBySessionID` trả về ErrSessionNotFound

---

## Files tạo ra

```
services/zep-thread/
├── internal/
│   ├── domain/
│   │   └── session.go
│   └── infra/
│       └── postgres/
│           ├── session_repo.go
│           └── session_repo_test.go
└── migrations/
    └── 001_create_sessions.sql
```

## Sau khi hoàn thành

Chạy: `go build ./services/zep-thread/...`
