# Solution: SOL-MB-005 — Admin Service (User, Project, Billing)

**CR:** [CR-MB-005](../CR-MB-005-Admin-Service.md)  
**Wave:** 1 (Data Layer — cần triển khai trước CR-001)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/memobase-admin` — foundation service cung cấp multi-tenant user/project management. **Phải triển khai trước CR-001** vì `users` và `projects` tables là foreign key dependencies của tất cả các tables khác.

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Không có multi-project isolation | Composite PK `(id, project_id)` trên tất cả user-scoped tables |
| Không có project token auth | `sk-proj-{project_id}-{bcrypt_hash}` format, validated via admin service |
| Không có billing | `billings` table + daily usage aggregation |
| Không có profile config per project | YAML string stored trong `projects.profile_config` |
| Không có cascade delete | PostgreSQL ON DELETE CASCADE + NATS broadcast |
| Không có user listing | Cursor-based pagination (không dùng OFFSET để tránh slow queries) |

---

## 2. Triển khai Order (Critical)

```
Wave 1 — Admin trước Ingestion:
  1. memobase-admin:  tạo projects, users tables → foundation
  2. memobase-ingestion: phụ thuộc users table (foreign key)
  3. memobase-engine:   phụ thuộc users table
  4. memobase-context:  đọc từ user_profiles (tạo bởi engine)
  5. memobase-event:    đọc từ user_events (tạo bởi engine)
```

---

## 3. Database Schema (Foundation)

**File:** `services/memobase-admin/internal/infra/migrations/001_foundation.up.sql`

```sql
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";  -- pgvector

-- Projects (multi-tenant registry)
CREATE TABLE IF NOT EXISTS projects (
    project_id     VARCHAR     NOT NULL PRIMARY KEY,
    project_secret VARCHAR     NOT NULL,  -- bcrypt hash of project token secret
    profile_config TEXT,                   -- YAML string
    status         VARCHAR     NOT NULL DEFAULT 'active'
                               CHECK (status IN ('active', 'suspended')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);

-- Billings
CREATE TABLE IF NOT EXISTS billings (
    id             UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    usage_left     BIGINT      NOT NULL DEFAULT 0,
    next_refill_at TIMESTAMPTZ
);

-- Project ↔ Billing (1:1 in practice, N:1 for future shared billing)
CREATE TABLE IF NOT EXISTS project_billings (
    project_id VARCHAR NOT NULL REFERENCES projects(project_id),
    billing_id UUID    NOT NULL REFERENCES billings(id),
    PRIMARY KEY (project_id, billing_id)
);

-- Users (composite PK for multi-tenant isolation)
CREATE TABLE IF NOT EXISTS users (
    id                UUID        NOT NULL,
    project_id        VARCHAR     NOT NULL REFERENCES projects(project_id),
    additional_fields JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id)
);
CREATE INDEX IF NOT EXISTS idx_users_project ON users(project_id);
CREATE INDEX IF NOT EXISTS idx_users_created ON users(project_id, created_at DESC);

-- User statuses (per-user state tracking)
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
    id                UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        VARCHAR     NOT NULL REFERENCES projects(project_id),
    date              DATE        NOT NULL,
    total_insert      BIGINT      NOT NULL DEFAULT 0,
    total_input_token BIGINT      NOT NULL DEFAULT 0,
    total_output_token BIGINT     NOT NULL DEFAULT 0,
    UNIQUE (project_id, date)
);
CREATE INDEX IF NOT EXISTS idx_usage_project_date ON usage_records(project_id, date DESC);
```

---

## 4. Project Token Authentication

### 4.1 Token Format & Generation

```go
// internal/domain/project.go

// Token format: "sk-proj-{project_id}-{secret}"
// Ví dụ: "sk-proj-my-project-abc123xyz789"

func GenerateProjectToken(projectID string) (token, hashedSecret string, err error) {
    // Generate 32-byte random secret
    secret := make([]byte, 32)
    rand.Read(secret)
    secretStr := base64.URLEncoding.EncodeToString(secret)

    // bcrypt hash (cost=12) → stored in projects.project_secret
    hash, err := bcrypt.GenerateFromPassword([]byte(secretStr), 12)
    
    // Token = "sk-proj-{projectID}-{secretStr}" (plaintext, given to user once)
    token = fmt.Sprintf("sk-proj-%s-%s", projectID, secretStr)
    hashedSecret = string(hash)
    return
}

func ParseProjectToken(token string) (projectID, secret string, err error) {
    // Expect: "sk-proj-{projectID}-{secret}"
    if !strings.HasPrefix(token, "sk-proj-") {
        return "", "", ErrInvalidTokenFormat
    }
    // Remove "sk-proj-" prefix
    rest := strings.TrimPrefix(token, "sk-proj-")
    // projectID là segment đầu tiên (có thể chứa "-" → split từ cuối)
    // secret là phần cuối sau delimiter (dài cố định 43 chars base64)
    const secretLen = 44  // base64 URL encoding of 32 bytes
    if len(rest) < secretLen+1 {
        return "", "", ErrInvalidTokenFormat
    }
    secret = rest[len(rest)-secretLen:]
    projectID = rest[:len(rest)-secretLen-1]  // -1 cho separator "-"
    return
}
```

### 4.2 Token Validation Flow

```go
// usecase/validate_project_token.go

func (uc *ValidateProjectTokenUseCase) Execute(ctx context.Context, token string) (*ProjectContext, error) {
    // 1. Parse token
    projectID, secret, err := ParseProjectToken(token)
    if err != nil {
        return nil, domain.ErrUnauthorized
    }

    // 2. Load project từ DB
    project, err := uc.projectRepo.GetByID(ctx, projectID)
    if err != nil {
        return nil, domain.ErrUnauthorized  // project not found → 401, không 404 (security)
    }

    // 3. Verify secret
    if err := bcrypt.CompareHashAndPassword([]byte(project.ProjectSecret), []byte(secret)); err != nil {
        return nil, domain.ErrUnauthorized
    }

    // 4. Check project status
    if project.Status == ProjectStatusSuspended {
        return nil, domain.ErrProjectSuspended  // → 403
    }

    return &ProjectContext{ProjectID: projectID}, nil
}
```

---

## 5. Use Case Layer

### 5.1 Create User

```go
// usecase/create_user.go
func (uc *CreateUserUseCase) Execute(ctx context.Context, req CreateUserRequest) (*User, error) {
    // Validate project exists và active
    project, err := uc.projectRepo.GetByID(ctx, req.ProjectID)
    if err != nil || project.Status == ProjectStatusSuspended {
        return nil, domain.ErrProjectSuspended
    }

    user := &User{
        ID:               uuid.New(),
        ProjectID:        req.ProjectID,
        AdditionalFields: req.AdditionalFields,
    }
    return uc.userRepo.Save(ctx, user)
}
```

### 5.2 Delete User (Cascade)

```go
// usecase/delete_user.go
func (uc *DeleteUserUseCase) Execute(ctx context.Context, userID, projectID string) error {
    // 1. Verify user exists
    _, err := uc.userRepo.GetByID(ctx, userID, projectID)
    if err != nil {
        return domain.ErrUserNotFound
    }

    // 2. Delete user → PostgreSQL CASCADE xóa:
    //    general_blobs, buffer_zones, user_profiles,
    //    user_events, user_event_gists, user_statuses
    if err := uc.userRepo.Delete(ctx, userID, projectID); err != nil {
        return err
    }

    // 3. Publish NATS event → các services dọn dẹp in-memory state
    return uc.eventPublisher.PublishUserDeleted(ctx, userID, projectID)
}
```

### 5.3 Project Profile Config Update

```go
// usecase/update_project_profile_config.go
func (uc *UpdateProjectProfileConfigUseCase) Execute(ctx context.Context, req UpdateProfileConfigRequest) error {
    // 1. Parse và validate YAML
    var config ProjectProfileConfig
    if err := yaml.Unmarshal([]byte(req.YAMLConfig), &config); err != nil {
        return fmt.Errorf("invalid YAML config: %w", err)
    }
    if err := config.Validate(); err != nil {
        return err
    }

    // 2. Store YAML string trong DB
    if err := uc.projectRepo.UpdateProfileConfig(ctx, req.ProjectID, req.YAMLConfig); err != nil {
        return err
    }

    // 3. Broadcast qua NATS → engine reloads config
    return uc.eventPublisher.PublishProjectUpdated(ctx, req.ProjectID, req.YAMLConfig)
}
```

### 5.4 List Project Users (Cursor Pagination)

```go
// usecase/list_project_users.go

// Dùng cursor-based pagination (không dùng OFFSET)
// Cursor = base64(last_user_id + created_at)

func (uc *ListProjectUsersUseCase) Execute(ctx context.Context, req ListProjectUsersRequest) (*ListProjectUsersResult, error) {
    limit := req.Limit
    if limit <= 0 || limit > 1000 {
        limit = 100
    }

    users, err := uc.userRepo.ListByProject(ctx, ListByProjectQuery{
        ProjectID: req.ProjectID,
        Limit:     limit + 1,  // fetch one extra to detect hasMore
        Cursor:    decodeCursor(req.Cursor),
    })

    // SQL:
    // SELECT id, project_id, additional_fields, created_at
    // FROM users
    // WHERE project_id = $1 AND created_at < $cursor_created_at
    // ORDER BY created_at DESC
    // LIMIT $2

    hasMore := len(users) > limit
    if hasMore {
        users = users[:limit]
    }

    var nextCursor string
    if hasMore && len(users) > 0 {
        last := users[len(users)-1]
        nextCursor = encodeCursor(last.ID, last.CreatedAt)
    }

    return &ListProjectUsersResult{
        Users:      users,
        NextCursor: nextCursor,
        HasMore:    hasMore,
    }, nil
}
```

---

## 6. Billing & Usage Tracking

```go
// Billing được cập nhật bởi Usage service (không trong scope CR-005)
// Admin service chỉ READ billing + usage records

// GET /api/v1/project/billing
type BillingResponse struct {
    TokenLeft    int64     `json:"token_left"`
    NextRefillAt time.Time `json:"next_refill_at"`
}

// GET /api/v1/project/usage?days=30
type UsageResponse struct {
    Records []UsageRecord `json:"records"`
}

type UsageRecord struct {
    Date             string `json:"date"`             // "2026-06-16"
    TotalInsert      int64  `json:"total_insert"`
    TotalInputToken  int64  `json:"total_input_token"`
    TotalOutputToken int64  `json:"total_output_token"`
}
```

**Usage recording** (background task trong memobase-engine):
```go
// Sau mỗi successful ProcessBlobs → upsert usage_records:
// INSERT INTO usage_records (project_id, date, total_insert, total_input_token, total_output_token)
// VALUES ($1, CURRENT_DATE, 1, $llm_input_tokens, $llm_output_tokens)
// ON CONFLICT (project_id, date)
// DO UPDATE SET
//   total_insert = usage_records.total_insert + 1,
//   total_input_token = usage_records.total_input_token + $llm_input_tokens,
//   total_output_token = usage_records.total_output_token + $llm_output_tokens
```

---

## 7. gRPC API

```protobuf
syntax = "proto3";
package memobase.admin.v1;

service AdminService {
    // User CRUD
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
    rpc ListProjectUsers(ListProjectUsersRequest) returns (ListProjectUsersResponse);

    // Project Config
    rpc GetProfileConfig(GetProfileConfigRequest) returns (GetProfileConfigResponse);
    rpc UpdateProfileConfig(UpdateProfileConfigRequest) returns (UpdateProfileConfigResponse);

    // Billing & Usage
    rpc GetBilling(GetBillingRequest) returns (GetBillingResponse);
    rpc GetUsage(GetUsageRequest) returns (GetUsageResponse);

    // Internal Auth
    rpc ValidateProjectToken(ValidateTokenRequest) returns (ValidateTokenResponse);
    rpc GetProject(GetProjectRequest) returns (GetProjectResponse);
}

message CreateUserRequest {
    string project_id = 1;
    map<string, google.protobuf.Value> additional_fields = 2;
}

message ListProjectUsersRequest {
    string project_id = 1;
    int32  limit      = 2;   // default: 100, max: 1000
    string cursor     = 3;   // opaque cursor for pagination
}

message ListProjectUsersResponse {
    repeated UserProto users    = 1;
    string             next_cursor = 2;
    bool               has_more    = 3;
}
```

---

## 8. NATS Events Published

| Subject | Trigger | Payload | Subscribers |
|---|---|---|---|
| `memobase.admin.user.deleted` | DeleteUser | `{user_id, project_id}` | ingestion, context, event |
| `memobase.admin.project.updated` | UpdateProfileConfig | `{project_id, config_yaml}` | engine, context |

---

## 9. Configuration

```yaml
admin:
  server:
    grpc_port: 9045
    health_port: 9095

  auth:
    root_token: "${ACCESS_TOKEN}"
    project_token_prefix: "sk-proj-"
    bcrypt_cost: 12

  database:
    url: "${DATABASE_URL}"
    pool_size: 15
    max_overflow: 5

  nats:
    url: "${NATS_URL}"
    stream: "memobase"
```

---

## 10. Testing Strategy

### Unit Tests
- `TestCreateUser_Success` → UUID v4 generated, `(id, project_id)` composite PK
- `TestDeleteUser_CascadeNATSPublished` → NATS event published after DB delete
- `TestValidateProjectToken_ValidFormat` → project_id extracted, bcrypt verified
- `TestValidateProjectToken_SuspendedProject` → `ErrProjectSuspended` (403)
- `TestListProjectUsers_Pagination` → cursor-based, no duplicates
- `TestUpdateProfileConfig_InvalidYAML` → validation error returned

### Integration Tests
- `TestUserCRUDE2E` — create → get → update → delete cycle
- `TestProjectTokenAuthE2E` — generate token → validate → suspended → 403
- `TestCascadeDeleteE2E` — delete user → verify all child records gone

---

## 11. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| bcrypt cost=12 chậm cho high-frequency token validation | Trung bình | Cache validated tokens trong Redis với TTL 60s (chỉ cache success) |
| User delete với large data volume (nhiều blobs/events) | Thấp | PostgreSQL CASCADE delete hiệu quả với index; không có N+1 |
| Profile config YAML injection | Thấp | Parse YAML → validate struct → reject unknown fields |
| NATS publish fail sau DB delete | Trung bình | Outbox pattern: write NATS event to DB first, publish asynchronously |
| Cursor pagination tampering | Thấp | Sign cursor với HMAC nếu cần; hiện tại opaque base64 đủ |
