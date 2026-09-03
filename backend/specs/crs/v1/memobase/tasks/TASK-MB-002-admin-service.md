# TASK-MB-002 — `services/memobase-admin` Admin & Multi-Tenant Service

**Wave:** 1 (Data Layer — phải xây trước mọi service khác)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-MB-001 (pkg/config)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-MB-005](../solutions/SOL-MB-005-Admin-Service.md)  
**Trạng thái:** ✅ Implemented
**Port gRPC:** 9045

---

## Mục tiêu

Tạo `services/memobase-admin/` — foundation service quản lý multi-tenant: Projects (với bcrypt token auth), Users (composite PK), Billing, Usage tracking, và NATS lifecycle events. **Phải xây trước mọi service khác** vì `users` và `projects` tables là foreign key dependencies.

---

## Cấu trúc thư mục

```
services/memobase-admin/
├── cmd/server/main.go
├── api/proto/memobase/admin/v1/admin.proto
├── internal/
│   ├── domain/
│   │   ├── project.go          # Project, ProjectStatus, GenerateProjectToken, ParseProjectToken
│   │   ├── user.go             # User, UserStatus
│   │   ├── billing.go          # Billing, UsageRecord
│   │   └── errors.go           # ErrUnauthorized, ErrProjectSuspended, ErrUserNotFound
│   ├── usecase/
│   │   ├── create_project.go
│   │   ├── create_user.go
│   │   ├── get_user.go
│   │   ├── update_user.go
│   │   ├── delete_user.go      # DB delete + NATS publish
│   │   ├── list_project_users.go # Cursor pagination
│   │   ├── validate_project_token.go
│   │   ├── get_profile_config.go
│   │   ├── update_profile_config.go
│   │   ├── get_billing.go
│   │   ├── get_usage.go
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go       # ProjectRepo, UserRepo, BillingRepo, EventPublisher
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/postgres/
│   │   │   ├── project_repo.go
│   │   │   ├── user_repo.go
│   │   │   └── billing_repo.go
│   │   └── event/
│   │       └── publisher.go    # NATS events
│   └── infra/
│       ├── migrations/
│       │   ├── 001_foundation.up.sql
│       │   └── 001_foundation.down.sql
│       ├── config/config.go
│       └── server/grpc.go
```

---

## 1. Database Migrations

**File: `internal/infra/migrations/001_foundation.up.sql`**

```sql
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";  -- pgvector for event embeddings

-- Projects (multi-tenant registry)
CREATE TABLE IF NOT EXISTS projects (
    project_id     VARCHAR     NOT NULL PRIMARY KEY,
    project_secret VARCHAR     NOT NULL,
    profile_config TEXT,
    status         VARCHAR     NOT NULL DEFAULT 'active'
                               CHECK (status IN ('active', 'suspended')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Billings
CREATE TABLE IF NOT EXISTS billings (
    id             UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    usage_left     BIGINT      NOT NULL DEFAULT 0,
    next_refill_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS project_billings (
    project_id VARCHAR NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    billing_id UUID    NOT NULL REFERENCES billings(id),
    PRIMARY KEY (project_id, billing_id)
);

-- Users (composite PK)
CREATE TABLE IF NOT EXISTS users (
    id                UUID        NOT NULL,
    project_id        VARCHAR     NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    additional_fields JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id)
);
CREATE INDEX IF NOT EXISTS idx_users_project     ON users(project_id);
CREATE INDEX IF NOT EXISTS idx_users_created     ON users(project_id, created_at DESC);

-- User statuses
CREATE TABLE IF NOT EXISTS user_statuses (
    id          UUID        NOT NULL,
    project_id  VARCHAR     NOT NULL,
    user_id     UUID        NOT NULL,
    status_data JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);

-- Daily usage records
CREATE TABLE IF NOT EXISTS usage_records (
    id                 UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id         VARCHAR     NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    date               DATE        NOT NULL,
    total_insert       BIGINT      NOT NULL DEFAULT 0,
    total_input_token  BIGINT      NOT NULL DEFAULT 0,
    total_output_token BIGINT      NOT NULL DEFAULT 0,
    UNIQUE (project_id, date)
);
CREATE INDEX IF NOT EXISTS idx_usage_project_date ON usage_records(project_id, date DESC);
```

---

## 2. Domain Models

**File: `internal/domain/project.go`**

```go
type ProjectStatus string
const (
    ProjectStatusActive    ProjectStatus = "active"
    ProjectStatusSuspended ProjectStatus = "suspended"
)

type Project struct {
    ProjectID     string
    ProjectSecret string  // bcrypt hash
    ProfileConfig string  // YAML string
    Status        ProjectStatus
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type ProjectProfileConfig struct {
    MaxSubtopics      int      `yaml:"max_subtopics"`
    MaxSlotTokenSize  int      `yaml:"max_slot_token_size"`
    StrictMode        bool     `yaml:"strict_mode"`
    ValidateMode      bool     `yaml:"validate_mode"`
    Language          string   `yaml:"language"`
    AdditionalTopics  []string `yaml:"additional_topics"`
}

func (c *ProjectProfileConfig) Validate() error
// Validate maxes > 0, language in ["en","zh"], etc.

// Token format: "sk-proj-{projectID}-{base64secret}"
func GenerateProjectToken(projectID string) (token, hashedSecret string, err error)
// Generate 32 random bytes → base64URLEncode → bcrypt → return plaintext + hash

func ParseProjectToken(token string) (projectID, secret string, err error)
// Must start with "sk-proj-"; extract projectID and secret
```

---

## 3. Use Cases

**validate_project_token.go:**
```go
func Execute(ctx context.Context, token string) (*ProjectContext, error)
// 1. ParseProjectToken → projectID, secret
// 2. projectRepo.GetByID → project
// 3. bcrypt.CompareHashAndPassword → verify
// 4. Check project.Status == active
// Returns: &ProjectContext{ProjectID} or ErrUnauthorized/ErrProjectSuspended
```

**list_project_users.go** (cursor pagination):
```go
// Cursor = base64(userID+"|"+createdAt.RFC3339Nano)
// SQL: WHERE project_id=$1 AND created_at < $cursor_time ORDER BY created_at DESC LIMIT $N+1
// Returns: users, nextCursor, hasMore
```

**delete_user.go** (cascade + NATS):
```go
// 1. userRepo.Delete → PostgreSQL CASCADE handles child tables
// 2. publisher.PublishUserDeleted → "memobase.admin.user.deleted"
```

---

## 4. gRPC Proto

**File: `api/proto/memobase/admin/v1/admin.proto`**

```protobuf
syntax = "proto3";
package memobase.admin.v1;
option go_package = "vnp-memory/services/memobase-admin/api/gen/admin/v1;adminv1";

service AdminService {
  rpc CreateProject(CreateProjectRequest) returns (CreateProjectResponse);
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
  rpc ListProjectUsers(ListProjectUsersRequest) returns (ListProjectUsersResponse);
  rpc GetProfileConfig(GetProfileConfigRequest) returns (GetProfileConfigResponse);
  rpc UpdateProfileConfig(UpdateProfileConfigRequest) returns (UpdateProfileConfigResponse);
  rpc GetBilling(GetBillingRequest) returns (GetBillingResponse);
  rpc GetUsage(GetUsageRequest) returns (GetUsageResponse);
  rpc ValidateProjectToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc GetProject(GetProjectRequest) returns (GetProjectResponse);
}

message CreateProjectResponse {
  string project_id    = 1;
  string project_token = 2;  // Returned ONCE
}

message ListProjectUsersRequest {
  string project_id = 1;
  int32  limit      = 2;
  string cursor     = 3;
}

message ValidateTokenRequest { string token = 1; }
message ValidateTokenResponse {
  string project_id = 1;
  bool   valid      = 2;
}
```

---

## 5. Monolith Bootstrap Integration

**File: `apps/memory/internal/bootstrap/memobase_admin.go`** (MODIFY/CREATE)

```go
func bootstrapMemobaseAdmin(ctx context.Context, cfg *config.Config, registry *bus.InProcessRegistry) error {
    db := infra.NewPostgresDB(cfg.MemobaseAdmin.Database)
    natsConn := infra.GetNATSConn(ctx)

    projectRepo := postgres.NewProjectRepository(db)
    userRepo := postgres.NewUserRepository(db)
    billingRepo := postgres.NewBillingRepository(db)
    publisher := event.NewNATSPublisher(natsConn)

    // Wire use cases
    handler := grpchandler.New(
        usecase.NewCreateProjectUseCase(projectRepo),
        usecase.NewCreateUserUseCase(userRepo, projectRepo),
        usecase.NewDeleteUserUseCase(userRepo, publisher),
        usecase.NewListProjectUsersUseCase(userRepo),
        usecase.NewValidateProjectTokenUseCase(projectRepo),
        // ...
    )

    // Register với bufconn (in-process)
    server := grpc.NewServer()
    adminv1.RegisterAdminServiceServer(server, handler)
    registry.Register("memobase-admin", server, bufconn.Listen(1024*1024))
    return nil
}
```

---

## Unit Tests

```
TestGenerateProjectToken_Format          → starts with "sk-proj-{projectID}-"
TestGenerateProjectToken_Unique          → 2 calls → different tokens
TestParseProjectToken_Valid              → extract projectID and secret
TestParseProjectToken_BadPrefix          → "Bearer xxx" → ErrInvalidTokenFormat
TestValidateToken_HappyPath              → valid token → ProjectContext
TestValidateToken_WrongSecret            → bcrypt mismatch → ErrUnauthorized
TestValidateToken_SuspendedProject       → status=suspended → ErrProjectSuspended
TestValidateToken_ProjectNotFound        → unknown projectID → ErrUnauthorized (not 404)
TestCreateUser_AssignsUUID               → ID is valid UUID v4
TestCreateUser_ProjectNotFound           → ErrProjectSuspended returned
TestDeleteUser_NATSPublished             → mock publisher → event called once
TestDeleteUser_UserNotFound              → ErrUserNotFound
TestListProjectUsers_NoCursor            → returns first page
TestListProjectUsers_WithCursor          → returns next page, no overlap
TestListProjectUsers_HasMoreTrue         → N+1 fetched → hasMore=true, N returned
TestListProjectUsers_EmptyResult         → no users → empty list, no cursor
TestUpdateProfileConfig_InvalidYAML     → bad YAML → validation error
TestUpdateProfileConfig_PublishesNATS   → valid config → NATS event published
TestProjectProfileConfig_Validate_Valid → valid config → nil error
TestProjectProfileConfig_Validate_Max0  → max_subtopics=0 → error
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

# Generate proto
buf generate services/memobase-admin/

# Run migrations (requires PostgreSQL)
psql $DATABASE_URL < services/memobase-admin/internal/infra/migrations/001_foundation.up.sql

# Build
go build ./services/memobase-admin/...

# Test
go test ./services/memobase-admin/... -v -count=1
```

---

## Ghi chú triển khai

- `golang.org/x/crypto/bcrypt` (cost=12 ≈ 200ms) — không cache trong phase 1 (thêm Redis cache sau)
- Cursor encoding: `base64.URLEncoding.EncodeToString([]byte(userID+"|"+createdAt.RFC3339Nano))`
- Cascade delete xử lý qua PostgreSQL `ON DELETE CASCADE` — không cần explicit delete child records
- `ACCESS_TOKEN` env: master token để create projects (chỉ dùng bởi admin CLI/operator)
- Monolith: bufconn transport, không bind TCP port khi chạy embedded
