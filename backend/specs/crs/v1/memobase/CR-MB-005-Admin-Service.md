# Change Request: CR-MB-005 — Admin Service (User, Project, Profile Config, Billing)

**CR ID:** CR-MB-005  
**Component:** `services/memobase-admin` [NEW SERVICE]  
**Priority:** High  
**Status:** In Progress
**Reference:** memobase PRD §5.6 (F-6), SRS §3.2, §3.9, specs/services/06-memobase-admin.md  
**Maps to Python:** `controllers/user.py`, `controllers/project.py`, `controllers/billing.py`

---

## 1. Mô tả

Xây dựng **memobase-admin** service — quản lý admin operations:
1. **User Management** — CRUD users trong project scope với custom metadata.
2. **Project Management** — project configuration (profile schema, language, strict_mode).
3. **Billing & Usage** — token consumption tracking, billing info.
4. **Multi-Tenant** — project_id partitioning, project token authentication.
5. **User Status Tracking** — track user state/metadata per project.
6. **Project User Listing** — list all users per project with pagination.

---

## 2. Vấn đề hiện tại

VNP Memory hiện tại:
- ✅ Có basic user management.
- ❌ Không có **multi-project isolation** (composite PK `(id, project_id)`).
- ❌ Không có **profile config per project** (YAML overrides via API).
- ❌ Không có **billing tracking** (token_left, next_refill_at).
- ❌ Không có **usage statistics** (daily aggregation: insert_count, input_tokens, output_tokens).
- ❌ Không có **project token authentication** (`sk-proj-*` format).
- ❌ Không có **project status** (suspended → 403 Forbidden).
- ❌ Không có **user cascade delete** (blobs, buffers, profiles, events).
- ❌ Không có **NATS broadcast** khi user deleted / project updated.
- ❌ Không có **user listing** với pagination (`GET /project/users`).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/memobase-admin/`

**Port:** `9045` (gRPC internal), **Health:** `9095`

```
services/memobase-admin/
├── internal/
│   ├── domain/
│   │   ├── user.go         # User entity, UserMetadata
│   │   ├── project.go      # Project entity, ProjectStatus
│   │   ├── billing.go      # Billing, UsageRecord
│   │   └── errors.go       # ErrUserNotFound, ErrProjectSuspended
│   ├── usecase/
│   │   ├── create_user.go
│   │   ├── get_user.go
│   │   ├── update_user.go
│   │   ├── delete_user.go          # CASCADE notify via NATS
│   │   ├── list_project_users.go   # Paginated listing
│   │   ├── get_project_profile_config.go
│   │   ├── update_project_profile_config.go
│   │   ├── get_billing.go
│   │   ├── get_usage.go
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go   # UserRepository, ProjectRepository, BillingRepository,
│   │                       #   EventPublisher, StatusRepository
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   ├── repository/postgres/
│   │   │   ├── user_repo.go
│   │   │   ├── project_repo.go
│   │   │   └── billing_repo.go
│   │   └── event/publisher.go  # NATS: admin.user.deleted, admin.project.updated
```

### 3.2. Domain Models

```go
// internal/domain/user.go
type User struct {
    ID               uuid.UUID
    ProjectID        string
    AdditionalFields map[string]any  // JSONB custom metadata
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

// internal/domain/project.go
type ProjectStatus string
const (
    ProjectStatusActive    ProjectStatus = "active"
    ProjectStatusSuspended ProjectStatus = "suspended"
)

type Project struct {
    ProjectID     string
    ProjectSecret string        // Hashed (for token validation)
    ProfileConfig string        // YAML config string
    Status        ProjectStatus
    CreatedAt     time.Time
}

// internal/domain/billing.go
type Billing struct {
    ID          uuid.UUID
    UsageLeft   int64   // remaining token quota
    NextRefillAt time.Time
}

type UsageRecord struct {
    Date            time.Time
    TotalInsert     int64   // blob insert count
    TotalInputToken int64
    TotalOutputToken int64
}
```

### 3.3. User CRUD

```go
// internal/usecase/create_user.go
func (uc *CreateUserUseCase) Execute(ctx, req CreateUserRequest) (*User, error) {
    // Validate project_id from request context
    // Generate UUID v4
    // Insert user with (id, project_id) composite PK
    // Return user with created_at
}

// internal/usecase/delete_user.go
func (uc *DeleteUserUseCase) Execute(ctx, userID, projectID string) error {
    // DELETE FROM users WHERE id=$1 AND project_id=$2
    // → ON DELETE CASCADE propagates to:
    //   general_blobs, buffer_zones, user_profiles,
    //   user_events, user_event_gists, user_statuses

    // Publish NATS: memobase.admin.user.deleted
    // → ingestion: clear any in-flight buffers
    // → context: invalidate Redis cache
    // → event: clear any indexed embeddings
    err = uc.eventPublisher.PublishUserDeleted(ctx, userID, projectID)
    return err
}
```

### 3.4. Project Profile Config API

```go
// internal/usecase/update_project_profile_config.go
// POST /api/v1/project/profile_config
// Input: YAML string with profile configuration

type ProjectProfileConfig struct {
    Language          string               `yaml:"language"`           // en | zh
    StrictMode        bool                 `yaml:"profile_strict_mode"`
    ValidateMode      bool                 `yaml:"profile_validate_mode"`
    MaxSubtopics      int                  `yaml:"max_profile_subtopics"`
    MaxSlotTokenSize  int                  `yaml:"max_pre_profile_token_size"`
    EventTags         []EventTagDef        `yaml:"event_tags"`
    Additional        []ProfileTopicDef    `yaml:"additional_user_profiles"`
    Overwrite         []ProfileTopicDef    `yaml:"overwrite_user_profiles"`
}

// Stored as YAML string in projects.profile_config column
// Published: NATS memobase.admin.project.updated → engine reloads config
```

### 3.5. Billing & Usage

```go
// GET /api/v1/project/billing
// Response:
// {
//   "data": {
//     "token_left": 1000000,
//     "next_refill_at": "2026-07-01T00:00:00Z"
//   }
// }

// GET /api/v1/project/usage
// Query: ?days=30 (default: 7)
// Response:
// {
//   "data": [
//     {
//       "date": "2026-06-16",
//       "total_insert": 150,
//       "total_input_token": 45000,
//       "total_output_token": 12000
//     }
//   ]
// }
```

### 3.6. Project Token Authentication

```go
// pkg/middleware/auth/

// Token formats:
// 1. Root token: ACCESS_TOKEN env var (direct string comparison)
// 2. Project token: "sk-proj-{project_id}-{hash}" format
//    → parse project_id from token
//    → verify hash against projects.project_secret in DB
//    → check project.status != "suspended" → 403 if suspended
//    → propagate project_id to gRPC metadata

// Token validation flow:
// Authorization: Bearer sk-proj-XXXX-YYYY
// → extract project_id = "XXXX"
// → DB: SELECT project_secret, status FROM projects WHERE project_id = "XXXX"
// → bcrypt.Compare(extracted_hash, project_secret)
// → if status == "suspended": return 403

// Health check endpoint: /api/v1/healthcheck → NO auth required
```

### 3.7. User Status Tracking

```go
// user_statuses table: track per-user state per project
// Used for: conversation state, active flags, custom metadata

CREATE TABLE user_statuses (
    id          UUID NOT NULL,
    project_id  VARCHAR NOT NULL,
    user_id     UUID NOT NULL,
    status_data JSONB NOT NULL,  // flexible key-value state
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
```

### 3.8. Database Schema

```sql
-- projects (multi-tenant registry)
CREATE TABLE projects (
    project_id     VARCHAR NOT NULL PRIMARY KEY,
    project_secret VARCHAR NOT NULL,
    profile_config TEXT,         -- YAML string
    status         VARCHAR NOT NULL DEFAULT 'active'  -- active | suspended
);

-- billings
CREATE TABLE billings (
    id             UUID NOT NULL PRIMARY KEY,
    usage_left     BIGINT NOT NULL DEFAULT 0,
    next_refill_at TIMESTAMPTZ
);

-- project_billings (N:1 association)
CREATE TABLE project_billings (
    project_id  VARCHAR NOT NULL,
    billing_id  UUID NOT NULL,
    PRIMARY KEY (project_id, billing_id),
    FOREIGN KEY (project_id) REFERENCES projects(project_id),
    FOREIGN KEY (billing_id) REFERENCES billings(id)
);

-- users
CREATE TABLE users (
    id                UUID NOT NULL,
    project_id        VARCHAR NOT NULL,
    additional_fields JSONB,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (project_id) REFERENCES projects(project_id)
);
CREATE INDEX idx_users_id_project_id ON users(id, project_id);
```

### 3.9. gRPC API

```protobuf
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

    // Auth (internal)
    rpc ValidateProjectToken(ValidateTokenRequest) returns (ValidateTokenResponse);
    rpc GetProject(GetProjectRequest) returns (GetProjectResponse);
}

message ListProjectUsersRequest {
    string project_id = 1;
    int32 limit = 2;    // default: 100
    string cursor = 3;  // for pagination
}

message UpdateProfileConfigRequest {
    string project_id = 1;
    string yaml_config = 2;  // raw YAML string
}
```

### 3.10. REST Endpoints

```
POST   /api/v1/users                           # Create user
GET    /api/v1/users/{user_id}                 # Get user
PUT    /api/v1/users/{user_id}                 # Update user metadata
DELETE /api/v1/users/{user_id}                 # Delete user (cascade)

POST   /api/v1/project/profile_config          # Update profile config
GET    /api/v1/project/profile_config          # Get current config
GET    /api/v1/project/billing                 # Get billing info
GET    /api/v1/project/users                   # List project users
GET    /api/v1/project/usage                   # Get daily usage stats

GET    /api/v1/healthcheck                     # Health (no auth)
GET    /api/v1/admin/status_check              # System status (root auth)
```

### 3.11. NATS Events Published

| Subject | Trigger | Payload |
|---|---|---|
| `memobase.admin.user.deleted` | DeleteUser | `{user_id, project_id}` |
| `memobase.admin.project.updated` | UpdateProfileConfig | `{project_id, config_yaml}` |

---

## 4. Configuration

```yaml
admin:
  grpc:
    port: 9045
  health:
    port: 9095
  auth:
    root_token: "${ACCESS_TOKEN}"
    project_token_prefix: "sk-proj-"
  database:
    url: "${DATABASE_URL}"
    pool_size: 15
  nats:
    url: "nats://nats:4222"
    stream: "memobase"
```

---

## 5. Acceptance Criteria

- [ ] `POST /api/v1/users` → trả về `{data: {id: <uuid>}, errno: 0}` với UUID v4.
- [ ] `POST /api/v1/users` với `{data: {name: "Alice", tier: "premium"}}` → `GET /api/v1/users/{id}` trả về `additional_fields: {name: "Alice", tier: "premium"}`.
- [ ] `DELETE /api/v1/users/{user_id}` → user + tất cả related data bị xóa (blobs, buffers, profiles, events).
- [ ] Request không có Bearer token → 401 Unauthorized.
- [ ] Project token `sk-proj-*` → parsed correctly, project_id extracted, project_secret verified.
- [ ] Project với `status = "suspended"` → 403 Forbidden trên mọi API call.
- [ ] `POST /api/v1/project/profile_config` với YAML `{language: zh, strict_mode: true}` → next engine flush sử dụng ZH prompts và strict mode.
- [ ] `GET /api/v1/project/users` → trả về list users với pagination (limit/cursor).
- [ ] `GET /api/v1/project/usage` → trả về daily aggregation cho 7 ngày gần nhất.
- [ ] `GET /api/v1/project/billing` → trả về `{token_left, next_refill_at}`.
- [ ] `GET /api/v1/healthcheck` → 200 OK (không cần auth).
- [ ] NATS `memobase.admin.user.deleted` published sau `DELETE /api/v1/users/{id}`.
- [ ] Project config update → NATS `memobase.admin.project.updated` → engine reloads config within 5s.
