# Solution: SOL-ZEP-001 — Conversation Thread & Session Management

**CR ID:** CR-ZEP-001  
**Solution ID:** SOL-ZEP-001  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/zep-thread/` (port gRPC 9042) với Clean Architecture 4 lớp. Service này thay thế `zep-thread` service hiện có trong `services/memory-service/domain/zep/` (hiện chỉ có `ZepSession` entity cơ bản). Implement đầy đủ lifecycle management với `ended_at` guard, advisory lock cho concurrent updates, và soft delete pattern.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `ZepSession` entity | `services/memory-service/internal/domain/zep/` | Có: SessionID, UserID, Metadata — thiếu EndedAt, DeletedAt |
| `zep-thread` bootstrap | `apps/memory/internal/bootstrap/` | Có: service đăng ký nhưng chỉ CRUD cơ bản |
| `zep-user` service | `apps/memory/internal/bootstrap/` | Có: UserService |
| `/v1/zep/*` routes | `gateway/adapter/handler/` | Có: 9 routes, cần thêm session-specific routes |

### Gap phân tích

- `ZepSession` thiếu `EndedAt *time.Time` và `DeletedAt *time.Time`
- Không có `ended_at` lifecycle guard cho PutMemory
- Không có advisory lock cho concurrent PATCH metadata
- Thiếu `UpsertSession` (create-or-get, dùng bởi Memory Service)
- Chưa có `ListUserThreads`, `SearchThreads`

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service

```
services/zep-thread/
├── internal/
│   ├── domain/
│   │   ├── session.go         # Session entity đầy đủ
│   │   └── repository.go      # SessionRepository port
│   ├── usecase/
│   │   ├── create_thread.go   # Tạo session mới
│   │   ├── get_thread.go      # Get by sessionID (respect soft delete)
│   │   ├── update_thread.go   # JSONB merge-patch với advisory lock
│   │   ├── end_thread.go      # Set ended_at = now()
│   │   ├── delete_thread.go   # Soft delete: set deleted_at
│   │   ├── list_threads.go    # List by project với pagination
│   │   ├── list_user_threads.go # List threads của một user
│   │   ├── search_threads.go  # Full-text + metadata search
│   │   └── upsert_session.go  # Create-or-get (dùng bởi Memory Service)
│   ├── adapter/
│   │   └── grpc/
│   │       └── thread_server.go   # gRPC ThreadService server
│   └── infra/
│       └── postgres/
│           └── session_repo.go    # PostgreSQL repository
```

### 3.2. Domain Model (Go)

```go
// services/zep-thread/internal/domain/session.go

package domain

import "time"

type Session struct {
    UUID        string
    SessionID   string          // user-provided unique ID (e.g. "chat_session_abc")
    UserID      *string         // optional: link to a ZepUser
    ProjectUUID string          // multi-tenant scope (project_uuid)
    Metadata    map[string]any  // JSONB free-form metadata
    EndedAt     *time.Time      // nil = active; non-nil = session closed
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time      // soft delete; nil = not deleted
}

// IsEnded returns true if the session has been explicitly ended
func (s *Session) IsEnded() bool {
    return s.EndedAt != nil
}

// IsDeleted returns true if the session is soft-deleted
func (s *Session) IsDeleted() bool {
    return s.DeletedAt != nil
}

// SessionRepository defines the data access contract
type SessionRepository interface {
    Create(ctx context.Context, s *Session) error
    GetBySessionID(ctx context.Context, sessionID, projectUUID string) (*Session, error)
    GetByUUID(ctx context.Context, uuid string) (*Session, error)
    Update(ctx context.Context, s *Session) error
    MergePatchMetadata(ctx context.Context, sessionID string, patch map[string]any) error // advisory lock
    SoftDelete(ctx context.Context, sessionID, projectUUID string) error
    List(ctx context.Context, projectUUID string, limit, offset int) ([]*Session, error)
    ListByUser(ctx context.Context, userID, projectUUID string) ([]*Session, error)
    Search(ctx context.Context, projectUUID, query string) ([]*Session, error)
    Upsert(ctx context.Context, sessionID, projectUUID string, userID *string) (*Session, error)
}
```

### 3.3. Advisory Lock Implementation

```go
// services/zep-thread/internal/infra/postgres/session_repo.go

package postgres

import (
    "crypto/sha256"
    "encoding/binary"
    "fmt"
)

// advisoryLockKey converts a session ID into a stable int64 for PostgreSQL advisory lock
func advisoryLockKey(sessionID string) int64 {
    hash := sha256.Sum256([]byte(sessionID))
    return int64(binary.BigEndian.Uint64(hash[:8]))
}

// MergePatchMetadata implements JSONB merge-patch with advisory lock
// Retry: exponential backoff 200ms → 30s, max 15 retries (from CR-ZEP-009)
func (r *SessionRepo) MergePatchMetadata(ctx context.Context, sessionID string, patch map[string]any) error {
    lockKey := advisoryLockKey(sessionID)

    return r.withAdvisoryLock(ctx, lockKey, func(tx *sql.Tx) error {
        patchJSON, err := json.Marshal(patch)
        if err != nil { return err }

        _, err = tx.ExecContext(ctx, `
            UPDATE sessions
            SET metadata = metadata || $1::jsonb,
                updated_at = NOW()
            WHERE session_id = $2
              AND deleted_at IS NULL
        `, patchJSON, sessionID)
        return err
    })
}

// withAdvisoryLock acquires a PostgreSQL session-level advisory lock,
// runs fn within a transaction, then releases the lock
func (r *SessionRepo) withAdvisoryLock(ctx context.Context, lockKey int64, fn func(*sql.Tx) error) error {
    conn, err := r.db.Conn(ctx)
    if err != nil { return err }
    defer conn.Close()

    // Acquire advisory lock
    if _, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
        return fmt.Errorf("acquire advisory lock: %w", err)
    }
    defer conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockKey)

    tx, err := conn.BeginTx(ctx, nil)
    if err != nil { return err }
    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }
    return tx.Commit()
}
```

### 3.4. Use Case Implementations

```go
// services/zep-thread/internal/usecase/end_thread.go

type EndThreadUseCase struct {
    repo SessionRepository
}

func (uc *EndThreadUseCase) Execute(ctx context.Context, sessionID, projectUUID string) error {
    session, err := uc.repo.GetBySessionID(ctx, sessionID, projectUUID)
    if err != nil { return err }
    if session.IsDeleted() { return ErrSessionNotFound }
    if session.IsEnded() { return ErrSessionAlreadyEnded }

    now := time.Now()
    session.EndedAt = &now
    session.UpdatedAt = now
    return uc.repo.Update(ctx, session)
}

// services/zep-thread/internal/usecase/upsert_session.go

type UpsertSessionUseCase struct {
    repo SessionRepository
}

// UpsertSession: Get existing session or create new one.
// Used by Memory Service's PutMemory to auto-create sessions.
func (uc *UpsertSessionUseCase) Execute(ctx context.Context, req UpsertSessionRequest) (*Session, error) {
    return uc.repo.Upsert(ctx, req.SessionID, req.ProjectUUID, req.UserID)
}
```

```go
// SQL for Upsert (atomic create-or-get)
const upsertSessionSQL = `
    INSERT INTO sessions (session_id, project_uuid, user_id, metadata)
    VALUES ($1, $2, $3, '{}')
    ON CONFLICT (session_id, project_uuid)
    DO UPDATE SET updated_at = NOW()
    RETURNING *
`
```

### 3.5. Session Lifecycle Guard (dùng bởi Memory Service)

```go
// Memory Service gọi Thread Service trước khi insert messages:
func (ms *MemoryService) PutMemory(ctx context.Context, req PutMemoryRequest) error {
    // 1. Upsert session via Thread Service gRPC
    session, err := ms.threadClient.UpsertSession(ctx, &threadpb.UpsertSessionRequest{
        SessionId:   req.SessionID,
        ProjectUuid: req.ProjectUUID,
    })
    if err != nil { return err }

    // 2. Lifecycle guard: từ chối nếu ended
    if session.EndedAt != nil {
        return ErrSessionEnded  // 400: "session has been ended"
    }

    // 3. Proceed with message insert
    // ...
}
```

---

## 4. Database Schema

```sql
-- Nâng cấp bảng sessions (từ zep_sessions hiện tại)
CREATE TABLE sessions (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   VARCHAR NOT NULL,
    user_id      VARCHAR,                                  -- optional: links to users.user_id
    project_uuid UUID NOT NULL,
    metadata     JSONB DEFAULT '{}',
    ended_at     TIMESTAMPTZ,                              -- non-nil = session closed
    created_at   TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at   TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    deleted_at   TIMESTAMPTZ                               -- soft delete
);

-- Indexes
CREATE INDEX session_user_id_idx ON sessions(user_id) WHERE deleted_at IS NULL;
CREATE INDEX session_project_idx ON sessions(project_uuid) WHERE deleted_at IS NULL;
CREATE INDEX session_metadata_gin ON sessions USING GIN(metadata);
CREATE UNIQUE INDEX sessions_session_project_uidx
    ON sessions(session_id, project_uuid)
    WHERE deleted_at IS NULL;

-- Full-text search index on metadata
CREATE INDEX sessions_fts ON sessions USING GIN(to_tsvector('english', metadata::text));
```

---

## 5. gRPC Protocol

```protobuf
// proto/thread/v1/thread.proto

service ThreadService {
    rpc CreateThread (CreateThreadRequest) returns (Session);
    rpc GetThread (GetThreadRequest) returns (Session);
    rpc UpdateThread (UpdateThreadRequest) returns (Session);
    rpc EndThread (EndThreadRequest) returns (google.protobuf.Empty);
    rpc DeleteThread (DeleteThreadRequest) returns (google.protobuf.Empty);
    rpc ListThreads (ListThreadsRequest) returns (ListThreadsResponse);
    rpc ListUserThreads (ListUserThreadsRequest) returns (ListThreadsResponse);
    rpc SearchThreads (SearchThreadsRequest) returns (ListThreadsResponse);
    rpc UpsertSession (UpsertSessionRequest) returns (Session);
}

message Session {
    string uuid = 1;
    string session_id = 2;
    optional string user_id = 3;
    string project_uuid = 4;
    google.protobuf.Struct metadata = 5;
    optional google.protobuf.Timestamp ended_at = 6;
    google.protobuf.Timestamp created_at = 7;
    google.protobuf.Timestamp updated_at = 8;
    optional google.protobuf.Timestamp deleted_at = 9;
}
```

---

## 6. API Endpoints (Gateway Integration)

```go
// gateway/adapter/handler/zep_thread_handler.go

func (h *ZepThreadHandler) Register(mux *http.ServeMux) {
    mux.HandleFunc("GET /api/v2/sessions",               h.ListSessions)
    mux.HandleFunc("POST /api/v2/sessions",              h.CreateSession)
    mux.HandleFunc("GET /api/v2/sessions-ordered",       h.ListSessionsOrdered)
    mux.HandleFunc("POST /api/v2/sessions/search",       h.SearchSessions)
    mux.HandleFunc("GET /api/v2/sessions/{id}",          h.GetSession)
    mux.HandleFunc("PATCH /api/v2/sessions/{id}",        h.UpdateSession)  // merge-patch + advisory lock
    mux.HandleFunc("DELETE /api/v2/sessions/{id}",       h.DeleteSession)  // soft delete
    mux.HandleFunc("POST /api/v2/sessions/{id}/end",     h.EndSession)     // set ended_at
    mux.HandleFunc("GET /api/v2/users/{id}/sessions",    h.ListUserSessions)
}
```

---

## 7. Bootstrap Integration (Monolith)

```go
// apps/memory/internal/bootstrap/zep_thread.go

func initZepThreadService(reg *bus.InProcessRegistry, cfg *config.Config) {
    db := postgres.New(cfg.Postgres)
    repo := postgres.NewSessionRepo(db)

    // Wire use cases
    svc := &ThreadService{
        createThread:     usecase.NewCreateThread(repo),
        getThread:        usecase.NewGetThread(repo),
        updateThread:     usecase.NewUpdateThread(repo),
        endThread:        usecase.NewEndThread(repo),
        deleteThread:     usecase.NewDeleteThread(repo),
        listThreads:      usecase.NewListThreads(repo),
        listUserThreads:  usecase.NewListUserThreads(repo),
        searchThreads:    usecase.NewSearchThreads(repo),
        upsertSession:    usecase.NewUpsertSession(repo),
    }

    grpcSrv := grpc.NewThreadServiceServer(svc)
    reg.Register("zep-thread", grpcSrv)
}
```

---

## 8. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + PostgreSQL schema migration | 1 ngày |
| **P2** | Advisory lock infra + MergePatch implementation | 1 ngày |
| **P3** | Create, Get, Delete, End use cases | 1 ngày |
| **P4** | List, ListUser, Search, Upsert use cases | 1 ngày |
| **P5** | gRPC server + proto definitions | 1 ngày |
| **P6** | Gateway REST handler integration | 1 ngày |
| **P7** | Bootstrap integration + tests | 1 ngày |

**Tổng:** ~7 ngày (Wave 2)

---

## 9. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| EndThread → thêm message mới bị từ chối | EndedAt guard trong Memory Service PutMemory |
| user_id → GET /users/:id/sessions trả về | ListUserThreads query + index |
| Concurrent PATCH → advisory lock | SHA-256 → int64 → pg_advisory_lock |
| Soft delete → GET trả về 404 | deleted_at IS NULL WHERE clause |
| PATCH nhiều lần → JSONB merge (không replace) | `metadata || $1::jsonb` SQL operator |
