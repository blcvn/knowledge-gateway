# TASK-SM-001 — Auth: sm_ API Key + RBAC Permission Matrix

**Task ID:** TASK-SM-001  
**Wave:** 1 (Foundation)  
**Solution:** [SOL-SM-007](../solutions/SOL-SM-007-Auth-Organization-RBAC.md)  
**Depends on:** —  
**Ước tính:** 3h  
**Priority:** Critical — auth foundation cho tất cả services

---

## Mục tiêu

Nâng cấp auth domain trong `services/vnp-platform/` với:
1. `sm_` prefixed API key generation (format: `sm_` + base62(32 bytes))
2. RBAC 4-role system (`owner`, `admin`, `editor`, `viewer`) với permission matrix
3. Redis cache cho token validation (TTL 5 phút, hot path ~1ms)

---

## Công việc cụ thể

### 1. Tạo/Nâng cấp `services/vnp-platform/internal/domain/auth/apikey.go`

```go
// GenerateAPIKey: "sm_" + base62(32 bytes) = 46 chars total
// Returns: (plaintext, sha256_hash)
// IMPORTANT: plaintext shown ONCE, store hash only
func GenerateAPIKey() (plaintext, hash string) { ... }

// IsValidAPIKeyFormat: bắt đầu bằng "sm_" + 43 base62 chars
func IsValidAPIKeyFormat(key string) bool { ... }
```

### 2. Tạo `services/vnp-platform/internal/domain/auth/rbac.go`

```go
type Role string
const (
    RoleOwner  Role = "owner"   // Full access
    RoleAdmin  Role = "admin"   // Most operations
    RoleEditor Role = "editor"  // Create/delete content, search
    RoleViewer Role = "viewer"  // Read-only
)

type Permission string
const (
    PermDocumentCreate   Permission = "document:create"
    PermDocumentDelete   Permission = "document:delete"
    PermMemoryForget     Permission = "memory:forget"
    PermSearchExecute    Permission = "search:execute"
    PermConnectionCreate Permission = "connection:create"
    PermSettingsWrite    Permission = "settings:write"
    PermMemberManage     Permission = "member:manage"
    PermAPIKeyManage     Permission = "apikey:manage"
    PermAnalyticsRead    Permission = "analytics:read"
)

// permissionMatrix theo CR-SM-007 spec:
// Owner: tất cả true
// Admin: PermSettingsWrite=false
// Editor: PermSettingsWrite=PermMemberManage=PermAPIKeyManage=PermAnalyticsRead=false
// Viewer: chỉ PermSearchExecute=PermAnalyticsRead=true

var permissionMatrix = map[Role]map[Permission]bool{ ... }

func HasPermission(role Role, perm Permission) bool { ... }
```

### 3. Nâng cấp Token Validation với Redis Cache

**`services/vnp-platform/internal/usecase/auth/validate_token.go`**

```go
// ValidateAPIKey flow:
// 1. IsValidAPIKeyFormat → ErrInvalidKeyFormat
// 2. SHA-256 hash
// 3. Redis GET "auth:apikey:{hash}" → cache HIT: return AuthContext
// 4. Cache MISS: DB lookup → ErrInvalidAPIKey
// 5. Check ExpiresAt
// 6. Load OrgMember.Role
// 7. Redis SET TTL 5 phút
// 8. Return AuthContext{OrgID, UserID, Role, KeyID}
```

### 4. Tạo RBAC Middleware

**`gateway/infra/middleware/rbac.go`**

```go
// RequirePermission wraps http.Handler với permission check
// 403 Forbidden nếu role không có permission
func RequirePermission(perm auth.Permission) func(http.Handler) http.Handler { ... }
```

### 5. Tests

**`services/vnp-platform/internal/domain/auth/apikey_test.go`**:
- `TestGenerateAPIKey_Format`: starts with "sm_", length = 46
- `TestGenerateAPIKey_Unique`: 1000 calls → 0 duplicates
- `TestIsValidAPIKeyFormat_Valid`: "sm_" + 43 base62 → true
- `TestIsValidAPIKeyFormat_Invalid`: missing prefix, wrong length → false

**`services/vnp-platform/internal/domain/auth/rbac_test.go`**:
- `TestHasPermission_Owner_AllTrue`: owner → tất cả permissions = true
- `TestHasPermission_Viewer_DocumentDelete_False`: viewer → document:delete = false
- `TestHasPermission_Editor_SettingsWrite_False`: editor → settings:write = false
- `TestPermissionMatrix_Complete`: tất cả 4 roles có đủ 9 permissions defined

**`gateway/infra/middleware/rbac_test.go`**:
- `TestRequirePermission_Forbidden`: viewer role + delete perm → 403
- `TestRequirePermission_Allowed`: editor role + create perm → next handler called

---

## Acceptance Criteria

- [ ] `go build ./services/vnp-platform/... && go build ./gateway/...` không lỗi
- [ ] GenerateAPIKey() → output starts with "sm_", total length 46
- [ ] 1000 calls GenerateAPIKey() → 0 duplicates
- [ ] HasPermission(RoleViewer, PermDocumentDelete) = false
- [ ] HasPermission(RoleOwner, PermSettingsWrite) = true
- [ ] RequirePermission middleware: viewer DELETE → 403
- [ ] Redis cache: 2nd ValidateAPIKey call = cache HIT (0 DB queries)
- [ ] `go test ./services/vnp-platform/... ./gateway/infra/middleware/...` pass

---

## Files tạo/sửa

```
services/vnp-platform/internal/domain/auth/
├── apikey.go              (MODIFY: thêm sm_ prefix + base62)
├── apikey_test.go         (NEW)
├── rbac.go                (NEW)
└── rbac_test.go           (NEW)

services/vnp-platform/internal/usecase/auth/
└── validate_token.go      (MODIFY: thêm Redis cache)

gateway/infra/middleware/
├── rbac.go                (NEW)
└── rbac_test.go           (NEW)
```

## Sau khi hoàn thành

Chạy: `go build ./... && go test ./services/vnp-platform/... ./gateway/infra/middleware/...`
