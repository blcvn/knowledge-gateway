# TASK-ZEP-007 — services/zep-admin: Project, API Key & Health Aggregation

**Task ID:** TASK-ZEP-007  
**Wave:** 2 (Core CRUD)  
**Solution:** [SOL-ZEP-008](../solutions/SOL-ZEP-008-Admin-Service-Multi-Tenant.md)  
**Depends on:** TASK-ZEP-001 (pkg/resilience)  
**Ước tính:** 4h  
**Priority:** High — API key auth foundation

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-admin: 6 .go - admin service multi-tenant  
---

## Mục tiêu

Tạo `services/zep-admin/` với 4 capabilities:
1. **Project CRUD** (create, get, list, soft-delete)
2. **API Key Management** (create `vnp_`+base62, list, revoke)
3. **Health Aggregation** (fan-out gRPC health checks tới 5 services)
4. **NATS cascade events** (project.created → init, project.deleted → cascade delete)

---

## Công việc cụ thể

### 1. Tạo Domain Model

**`services/zep-admin/internal/domain/project.go`**
```go
type Project struct {
    UUID      string
    Name      string
    Settings  ProjectSettings
    CreatedAt time.Time
    DeletedAt *time.Time
}

type ProjectSettings struct {
    MaxRequestSizeMB int  // default: 5
    TimeoutSeconds   int  // default: 30
    TelemetryEnabled bool // default: true
}
```

**`services/zep-admin/internal/domain/apikey.go`**
```go
type APIKey struct {
    UUID        string
    ProjectUUID string
    Name        string     // human label
    Hash        string     // SHA-256 hex (stored)
    Prefix      string     // first 8 chars (for identification)
    CreatedAt   time.Time
    LastUsedAt  *time.Time
    RevokedAt   *time.Time // nil = active
}
```

### 2. Tạo API Key Generator

**`services/zep-admin/internal/domain/keygen.go`**
```go
// GenerateAPIKey tạo API key format: "vnp_" + base62(32 random bytes)
// Returns: (plaintext, sha256hash, prefix)
// IMPORTANT: plaintext chỉ trả về 1 lần, KHÔNG lưu vào database
func GenerateAPIKey() (plaintext, hash, prefix string) { ... }

// base62Encode mã hóa bytes thành string A-Za-z0-9
// Pad về đúng 43 chars cho 32 bytes
func base62Encode(data []byte) string { ... }
```

### 3. Tạo PostgreSQL Schema

**`services/zep-admin/migrations/001_create_admin_tables.sql`**
```sql
CREATE TABLE zep_projects (
    uuid       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR NOT NULL,
    settings   JSONB NOT NULL DEFAULT '{"max_request_size_mb":5,"timeout_seconds":30,"telemetry_enabled":true}',
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE zep_api_keys (
    uuid          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_uuid  UUID NOT NULL REFERENCES zep_projects(uuid),
    name          VARCHAR NOT NULL,
    hash          VARCHAR(64) NOT NULL UNIQUE,
    prefix        VARCHAR(8) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    last_used_at  TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ
);

CREATE INDEX zep_api_keys_project_idx ON zep_api_keys(project_uuid);
CREATE INDEX zep_api_keys_hash_idx ON zep_api_keys(hash) WHERE revoked_at IS NULL;
```

### 4. Tạo Health Checker

**`services/zep-admin/internal/infra/health/health_checker.go`**
```go
// HealthChecker fan-out concurrent gRPC health checks tới 5 downstream services
// Sử dụng google.golang.org/grpc/health/grpc_health_v1
// Timeout: 5s per service
// Trả về aggregated status: "healthy" | "degraded" | "unhealthy"
```

### 5. Tạo Use Cases

- `create_project.go`: INSERT project → publish NATS `zep.admin.project.created`
- `delete_project.go`: soft-delete → publish NATS `zep.admin.project.deleted`
- `list_projects.go`, `get_project.go`
- `create_api_key.go`: GenerateAPIKey() → store hash → return plaintext (ONCE)
- `list_api_keys.go`: list với prefix shown
- `revoke_api_key.go`: set revoked_at + bust Redis cache

### 6. Tạo gRPC + REST Handler

**REST endpoints:**
```
GET    /healthz                           → aggregate health (public)
GET    /api/v2/admin/projects             → list
POST   /api/v2/admin/projects             → create
GET    /api/v2/admin/projects/{id}        → get
DELETE /api/v2/admin/projects/{id}        → soft-delete + NATS cascade
POST   /api/v2/admin/api-keys             → create (plaintext returned once)
GET    /api/v2/admin/api-keys             → list (prefix only, no hash)
DELETE /api/v2/admin/api-keys/{id}        → revoke
```

### 7. Tests

- `TestGenerateAPIKey_Format`: output starts with "vnp_", length = 47
- `TestGenerateAPIKey_Unique`: 100 calls → 100 unique keys
- `TestRevokeAPIKey_BlocksAuth`: revoked key → 401
- `TestHealthChecker_AllUp`: mock all services healthy → status "healthy"
- `TestHealthChecker_OneDown`: one service down → status "degraded"

---

## Acceptance Criteria

- [ ] `go build ./services/zep-admin/...` không có lỗi
- [ ] `GenerateAPIKey()` output format: `vnp_` + 43 alphanumeric chars (47 total)
- [ ] 1000 calls `GenerateAPIKey()` → 0 duplicates
- [ ] POST /api-keys → response có `plaintext`, subsequent GET → `plaintext` KHÔNG có
- [ ] POST /projects → NATS event `zep.admin.project.created` được publish
- [ ] DELETE /projects/:id → NATS event `zep.admin.project.deleted` được publish
- [ ] GET /healthz → aggregate status từ 5 services trong 5s timeout
- [ ] `go test ./services/zep-admin/...` pass

---

## Files tạo ra

```
services/zep-admin/
├── internal/
│   ├── domain/
│   │   ├── project.go
│   │   ├── apikey.go
│   │   └── keygen.go
│   ├── usecase/
│   │   ├── create_project.go
│   │   ├── delete_project.go
│   │   ├── list_projects.go
│   │   ├── get_project.go
│   │   ├── create_api_key.go
│   │   ├── list_api_keys.go
│   │   └── revoke_api_key.go
│   ├── adapter/
│   │   └── http/
│   │       └── admin_handler.go
│   └── infra/
│       ├── postgres/
│       │   ├── project_repo.go
│       │   └── apikey_repo.go
│       └── health/
│           └── health_checker.go
└── migrations/
    └── 001_create_admin_tables.sql
```

## Sau khi hoàn thành

Chạy: `go build ./services/zep-admin/... && go test ./services/zep-admin/...`
