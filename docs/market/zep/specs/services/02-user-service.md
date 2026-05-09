# 02 — User Service (zep-user)

> **gRPC**: 9041 | **Health**: 9141  
> **Origin**: L3 — Business Logic Layer (User DAO)

---

## 1. Purpose

Quản lý toàn bộ lifecycle của User entities. Cung cấp:
- User CRUD với metadata management (JSONB merge-patch)
- Multi-tenant isolation qua `project_uuid`
- Soft delete pattern (audit trail preservation)
- User-to-graph association cho Temporal Knowledge Graph

---

## 2. Clean Architecture Layout

```
services/zep-user/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── user.go                # User entity
│   │   ├── user_id.go             # UserID value object (unique, alphanumeric_with_underscores)
│   │   ├── project.go             # ProjectUUID value object
│   │   ├── metadata.go            # Metadata value object (JSONB map)
│   │   ├── event.go               # UserCreated, UserUpdated, UserDeleted events
│   │   └── errors.go              # ErrUserNotFound, ErrUserAlreadyExists, ErrInvalidUserID
│   │
│   ├── usecase/
│   │   ├── create_user.go         # Create user with validation
│   │   ├── get_user.go            # Get user by user_id
│   │   ├── update_user.go         # Patch metadata via JSONB merge
│   │   ├── delete_user.go         # Soft delete (set deleted_at)
│   │   ├── list_users.go          # List with pagination + ordering
│   │   ├── port/
│   │   │   ├── input.go           # UserService interface
│   │   │   └── output.go          # UserRepository, EventPublisher
│   │   └── dto/
│   │       ├── request.go         # CreateUserRequest, UpdateUserRequest, ListUsersRequest
│   │       └── response.go        # UserResponse, UserListResponse
│   │
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC UserServiceServer implementation
│   │   │   └── mapper.go          # Proto ↔ Domain mapping
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── user_repo.go   # PostgreSQL CRUD (uptrace/bun)
│   │   │       └── model.go       # bun table model
│   │   └── event/
│   │       └── publisher.go       # NATS event publisher
│   │
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Layer

### 3.1 User Entity

```go
package domain

import (
    "time"
)

type User struct {
    UUID        string            // PK, auto-generated UUID
    UserID      UserID            // unique, human-readable identifier
    Email       string            // optional
    FirstName   string            // optional
    LastName    string            // optional
    ProjectUUID string            // FK, multi-tenant isolation
    Metadata    Metadata          // JSONB arbitrary data
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time        // soft delete marker
}

// UserID is a value object enforcing alphanumeric_with_underscores
type UserID string

func NewUserID(raw string) (UserID, error) {
    if !isAlphanumericWithUnderscores(raw) {
        return "", ErrInvalidUserID
    }
    return UserID(raw), nil
}

// Metadata is a JSONB map supporting merge-patch updates
type Metadata map[string]any

func (m Metadata) Merge(patch Metadata) Metadata {
    result := make(Metadata)
    for k, v := range m {
        result[k] = v
    }
    for k, v := range patch {
        if v == nil {
            delete(result, k)
        } else {
            result[k] = v
        }
    }
    return result
}
```

### 3.2 Domain Events

```go
type UserCreated struct {
    UserID      string
    ProjectUUID string
    Timestamp   time.Time
}

type UserUpdated struct {
    UserID      string
    ProjectUUID string
    Fields      []string   // changed fields
    Timestamp   time.Time
}

type UserDeleted struct {
    UserID      string
    ProjectUUID string
    Timestamp   time.Time
}
```

### 3.3 Domain Errors

```go
var (
    ErrUserNotFound      = errors.New("user not found")
    ErrUserAlreadyExists = errors.New("user already exists")
    ErrInvalidUserID     = errors.New("user_id must be alphanumeric with underscores")
    ErrProjectRequired   = errors.New("project_uuid is required")
)
```

---

## 4. Use Case Layer

### 4.1 Port Interfaces

```go
package port

// Input Port — use case interface
type UserService interface {
    CreateUser(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error)
    GetUser(ctx context.Context, userID string, projectUUID string) (*dto.UserResponse, error)
    UpdateUser(ctx context.Context, req dto.UpdateUserRequest) (*dto.UserResponse, error)
    DeleteUser(ctx context.Context, userID string, projectUUID string) error
    ListUsers(ctx context.Context, req dto.ListUsersRequest) (*dto.UserListResponse, error)
    ListOrderedUsers(ctx context.Context, req dto.ListUsersRequest) (*dto.UserListResponse, error)
}

// Output Port — repository interface
type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
    GetByUserID(ctx context.Context, userID string, projectUUID string) (*domain.User, error)
    Update(ctx context.Context, user *domain.User) error
    SoftDelete(ctx context.Context, userID string, projectUUID string) error
    List(ctx context.Context, projectUUID string, limit, offset int) ([]*domain.User, int, error)
    ListOrdered(ctx context.Context, projectUUID string, limit, offset int, orderBy string) ([]*domain.User, int, error)
}

// Output Port — event publisher
type UserEventPublisher interface {
    PublishUserCreated(ctx context.Context, event domain.UserCreated) error
    PublishUserDeleted(ctx context.Context, event domain.UserDeleted) error
}
```

### 4.2 CreateUser Use Case

```go
func (uc *CreateUserUseCase) Execute(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
    // 1. Validate user_id format
    userID, err := domain.NewUserID(req.UserID)
    if err != nil {
        return nil, err
    }
    
    // 2. Extract tenant from context
    projectUUID := tenant.FromContext(ctx).ProjectUUID
    
    // 3. Build domain entity
    user := &domain.User{
        UUID:        uuid.NewString(),
        UserID:      userID,
        Email:       req.Email,
        FirstName:   req.FirstName,
        LastName:    req.LastName,
        ProjectUUID: projectUUID,
        Metadata:    domain.Metadata(req.Metadata),
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    // 4. Persist
    if err := uc.repo.Create(ctx, user); err != nil {
        return nil, err
    }
    
    // 5. Publish event
    uc.publisher.PublishUserCreated(ctx, domain.UserCreated{
        UserID: string(userID), ProjectUUID: projectUUID, Timestamp: user.CreatedAt,
    })
    
    return dto.FromUser(user), nil
}
```

---

## 5. gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.user.v1;

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);
  rpc GetUser(GetUserRequest) returns (UserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
  rpc ListAllUsers(ListUsersRequest) returns (UserListResponse);
  rpc ListAllOrderedUsers(ListUsersRequest) returns (UserListResponse);
  rpc ListUserSessions(ListUserSessionsRequest) returns (SessionListResponse);
}

message UserResponse {
  string uuid = 1;
  string user_id = 2;
  string email = 3;
  string first_name = 4;
  string last_name = 5;
  string project_uuid = 6;
  google.protobuf.Struct metadata = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}

message CreateUserRequest {
  string user_id = 1;     // required, alphanumeric_with_underscores
  string email = 2;       // optional
  string first_name = 3;  // optional
  string last_name = 4;   // optional
  google.protobuf.Struct metadata = 5;
}

message UpdateUserRequest {
  string user_id = 1;     // required
  optional string email = 2;
  optional string first_name = 3;
  optional string last_name = 4;
  google.protobuf.Struct metadata = 5;  // merge-patch
}

message ListUsersRequest {
  int32 limit = 1;
  int32 offset = 2;
  string order_by = 3;    // "created_at" | "user_id"
}
```

---

## 6. PostgreSQL Schema

```sql
CREATE TABLE users (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL,
    email        TEXT,
    first_name   TEXT,
    last_name    TEXT,
    project_uuid UUID NOT NULL,
    metadata     JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    
    UNIQUE (user_id, project_uuid)
);

CREATE INDEX user_user_id_idx ON users(user_id) WHERE deleted_at IS NULL;
CREATE INDEX user_email_idx ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX user_project_uuid_idx ON users(project_uuid) WHERE deleted_at IS NULL;
```

---

## 7. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.user.created` | `{user_id, project_uuid, timestamp}` | zep-graph (init user node) |
| `zep.user.updated` | `{user_id, project_uuid, fields[], timestamp}` | zep-graph (update user node) |
| `zep.user.deleted` | `{user_id, project_uuid, timestamp}` | zep-thread (cascade soft delete sessions), zep-graph (delete user graph data) |

---

## 8. Configuration

```yaml
user:
  grpc:
    port: 9041
  health:
    port: 9141
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    conn_max_lifetime: 30m
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  telemetry:
    service_name: "zep-user"
    otel_endpoint: "otel-collector:4317"
```
