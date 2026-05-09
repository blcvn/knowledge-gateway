# 03 — Thread Service (zep-thread)

> **gRPC**: 9042 | **Health**: 9142  
> **Origin**: L3 — Business Logic Layer (Session DAO)

---

## 1. Purpose

Quản lý lifecycle của Session/Thread entities — đơn vị tổ chức conversation giữa user và AI agent. Cung cấp:
- Thread CRUD với metadata management (JSONB merge-patch)
- Session state management (`ended_at` — blocks future message ingestion)
- Advisory lock-based concurrency control cho concurrent metadata updates
- User-thread association (optional `user_id` FK)

---

## 2. Clean Architecture Layout

```
services/zep-thread/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── session.go             # Session/Thread entity
│   │   ├── session_id.go          # SessionID value object
│   │   ├── metadata.go            # Metadata value object (JSONB merge)
│   │   ├── advisory_lock.go       # AdvisoryLockKey (SHA-256 hash of session_id)
│   │   ├── event.go               # SessionCreated, SessionEnded, SessionUpdated
│   │   └── errors.go              # ErrSessionNotFound, ErrSessionEnded, ErrLockTimeout
│   │
│   ├── usecase/
│   │   ├── create_session.go      # Create session, optional user_id link
│   │   ├── get_session.go         # Get by session_id
│   │   ├── update_session.go      # Patch metadata with advisory lock
│   │   ├── list_sessions.go       # List with pagination
│   │   ├── list_ordered.go        # Ordered list (created_at desc)
│   │   ├── list_user_sessions.go  # List sessions for a user
│   │   ├── upsert_session.go      # Create-or-update (used by PutMemory flow)
│   │   ├── end_session.go         # Set ended_at, block future ingestion
│   │   ├── port/
│   │   │   ├── input.go           # ThreadService interface
│   │   │   └── output.go          # SessionRepository, LockManager, EventPublisher
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── session_repo.go    # bun ORM session store
│   │   │       ├── lock_manager.go    # PostgreSQL advisory lock implementation
│   │   │       └── model.go
│   │   └── event/
│   │       └── publisher.go
│   │
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Layer

### 3.1 Session Entity

```go
package domain

type Session struct {
    UUID        string
    SessionID   SessionID         // unique, human-readable
    UserID      *string           // nullable FK → User.user_id
    ProjectUUID string            // multi-tenant isolation
    Metadata    Metadata          // JSONB arbitrary data
    EndedAt     *time.Time        // marks session closed
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time        // soft delete
}

// IsEnded checks if session has been explicitly ended
func (s *Session) IsEnded() bool {
    return s.EndedAt != nil
}

// CanAcceptMessages returns true if session can receive new messages
func (s *Session) CanAcceptMessages() bool {
    return !s.IsEnded() && s.DeletedAt == nil
}
```

### 3.2 Advisory Lock

```go
// AdvisoryLockKey generates a PostgreSQL advisory lock key from session_id
type AdvisoryLockKey int64

func NewAdvisoryLockKey(sessionID string) AdvisoryLockKey {
    h := sha256.Sum256([]byte(sessionID))
    return AdvisoryLockKey(binary.BigEndian.Int64(h[:8]))
}
```

### 3.3 Domain Errors

```go
var (
    ErrSessionNotFound    = errors.New("session not found")
    ErrSessionEnded       = errors.New("session has been ended; no new messages can be added")
    ErrSessionAlreadyExists = errors.New("session already exists")
    ErrLockTimeout        = errors.New("advisory lock acquisition timed out")
    ErrInvalidSessionID   = errors.New("session_id must be alphanumeric with underscores")
)
```

---

## 4. Use Case Layer

### 4.1 Port Interfaces

```go
package port

type ThreadService interface {
    CreateSession(ctx context.Context, req dto.CreateSessionRequest) (*dto.SessionResponse, error)
    GetSession(ctx context.Context, sessionID string) (*dto.SessionResponse, error)
    UpdateSession(ctx context.Context, req dto.UpdateSessionRequest) (*dto.SessionResponse, error)
    UpsertSession(ctx context.Context, req dto.UpsertSessionRequest) (*dto.SessionResponse, error)
    EndSession(ctx context.Context, sessionID string) error
    ListSessions(ctx context.Context, req dto.ListSessionsRequest) (*dto.SessionListResponse, error)
    ListOrderedSessions(ctx context.Context, req dto.ListSessionsRequest) (*dto.SessionListResponse, error)
    ListUserSessions(ctx context.Context, userID string, req dto.ListSessionsRequest) (*dto.SessionListResponse, error)
}

type SessionRepository interface {
    Create(ctx context.Context, session *domain.Session) error
    GetBySessionID(ctx context.Context, sessionID string, projectUUID string) (*domain.Session, error)
    Update(ctx context.Context, session *domain.Session) error
    Upsert(ctx context.Context, session *domain.Session) error
    SetEndedAt(ctx context.Context, sessionID string, projectUUID string) error
    SoftDelete(ctx context.Context, sessionID string, projectUUID string) error
    List(ctx context.Context, projectUUID string, limit, offset int) ([]*domain.Session, int, error)
    ListOrdered(ctx context.Context, projectUUID string, limit, offset int) ([]*domain.Session, int, error)
    ListByUserID(ctx context.Context, userID string, projectUUID string, limit, offset int) ([]*domain.Session, int, error)
}

type LockManager interface {
    AcquireAdvisoryLock(ctx context.Context, key domain.AdvisoryLockKey) error
    ReleaseAdvisoryLock(ctx context.Context, key domain.AdvisoryLockKey) error
}
```

### 4.2 UpdateSession Use Case (with Advisory Lock)

```go
func (uc *UpdateSessionUseCase) Execute(ctx context.Context, req dto.UpdateSessionRequest) (*dto.SessionResponse, error) {
    projectUUID := tenant.FromContext(ctx).ProjectUUID
    
    // 1. Acquire advisory lock (prevent concurrent metadata updates)
    lockKey := domain.NewAdvisoryLockKey(req.SessionID)
    if err := uc.lockManager.AcquireAdvisoryLock(ctx, lockKey); err != nil {
        return nil, domain.ErrLockTimeout
    }
    defer uc.lockManager.ReleaseAdvisoryLock(ctx, lockKey)
    
    // 2. Get existing session
    session, err := uc.repo.GetBySessionID(ctx, req.SessionID, projectUUID)
    if err != nil {
        return nil, err
    }
    
    // 3. Merge metadata (JSONB merge-patch)
    if req.Metadata != nil {
        session.Metadata = session.Metadata.Merge(domain.Metadata(req.Metadata))
    }
    session.UpdatedAt = time.Now()
    
    // 4. Persist
    if err := uc.repo.Update(ctx, session); err != nil {
        return nil, err
    }
    
    return dto.FromSession(session), nil
}
```

### 4.3 Retry Policy for Lock Acquisition

```go
// Exponential backoff: 200ms → 30s, max 15 retries
type RetryPolicy struct {
    InitialInterval time.Duration  // 200ms
    MaxInterval     time.Duration  // 30s
    MaxRetries      int            // 15
    Multiplier      float64        // 2.0
}
```

---

## 5. gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.thread.v1;

service ThreadService {
  rpc CreateSession(CreateSessionRequest) returns (SessionResponse);
  rpc GetSession(GetSessionRequest) returns (SessionResponse);
  rpc UpdateSession(UpdateSessionRequest) returns (SessionResponse);
  rpc UpsertSession(UpsertSessionRequest) returns (SessionResponse);
  rpc EndSession(EndSessionRequest) returns (google.protobuf.Empty);
  rpc ListSessions(ListSessionsRequest) returns (SessionListResponse);
  rpc ListOrderedSessions(ListSessionsRequest) returns (SessionListResponse);
  rpc ListUserSessions(ListUserSessionsRequest) returns (SessionListResponse);
}

message SessionResponse {
  string uuid = 1;
  string session_id = 2;
  optional string user_id = 3;
  string project_uuid = 4;
  google.protobuf.Struct metadata = 5;
  optional google.protobuf.Timestamp ended_at = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
}

message CreateSessionRequest {
  string session_id = 1;           // required
  optional string user_id = 2;    // optional FK
  google.protobuf.Struct metadata = 3;
}

message UpsertSessionRequest {
  string session_id = 1;           // required
  optional string user_id = 2;
  google.protobuf.Struct metadata = 3;
}

message UpdateSessionRequest {
  string session_id = 1;
  google.protobuf.Struct metadata = 2;  // merge-patch
}
```

---

## 6. PostgreSQL Schema

```sql
CREATE TABLE sessions (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   TEXT NOT NULL,
    user_id      TEXT,                      -- FK → users.user_id (nullable)
    project_uuid UUID NOT NULL,
    metadata     JSONB DEFAULT '{}',
    ended_at     TIMESTAMPTZ,              -- session closure marker
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    
    UNIQUE (session_id, project_uuid)
);

CREATE INDEX session_user_id_idx ON sessions(user_id) WHERE deleted_at IS NULL;
CREATE INDEX session_project_uuid_idx ON sessions(project_uuid) WHERE deleted_at IS NULL;
CREATE INDEX session_composite_idx ON sessions(session_id, project_uuid, deleted_at);
CREATE INDEX session_created_at_idx ON sessions(created_at DESC) WHERE deleted_at IS NULL;
```

---

## 7. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.thread.session.created` | `{session_id, user_id, project_uuid}` | zep-graph (init session context) |
| `zep.thread.session.ended` | `{session_id, project_uuid, ended_at}` | zep-memory (cleanup pending ops) |
| `zep.thread.session.deleted` | `{session_id, project_uuid}` | zep-memory (cascade messages), zep-graph (cascade graph data) |

---

## 8. Configuration

```yaml
thread:
  grpc:
    port: 9042
  health:
    port: 9142
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 10
  advisory_lock:
    retry_initial: 200ms
    retry_max: 30s
    retry_max_attempts: 15
    retry_multiplier: 2.0
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  telemetry:
    service_name: "zep-thread"
    otel_endpoint: "otel-collector:4317"
```
