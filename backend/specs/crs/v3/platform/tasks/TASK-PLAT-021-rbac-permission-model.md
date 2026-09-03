# TASK-PLAT-021 — RBAC Permission Model

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-021 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-PLAT-007](../solutions/SOL-PLAT-007-RBAC-Authorization.md) §2 |
| **Component** | `shared/pkg/auth/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 2h |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** UserRole constants in entity.go (admin/editor/viewer); Permission model struct not created
---

## Mục tiêu

Tạo permission model và `HasPermission()` function trong `shared/pkg/auth/rbac.go`.

---

## Công việc cụ thể

### 1. Tạo `shared/pkg/auth/rbac.go` [NEW]

```go
package auth

import "net/http"

type Permission string

const (
    PermMemoryStore  Permission = "memory:store"
    PermMemoryRecall Permission = "memory:recall"
    PermMemoryForget Permission = "memory:forget"
    PermAdminForget  Permission = "admin:forget"
    PermAuditView    Permission = "audit:view"
    PermOrgWrite     Permission = "org:write"
    PermAPIKeyCreate Permission = "api_key:create"
    PermAPIKeyRevoke Permission = "api_key:revoke"
    PermMembersAdmin Permission = "members:admin"
)

var rolePermissions = map[string]map[Permission]bool{
    "admin": {
        PermMemoryStore: true, PermMemoryRecall: true, PermMemoryForget: true,
        PermAdminForget: true, PermAuditView: true, PermOrgWrite: true,
        PermAPIKeyCreate: true, PermAPIKeyRevoke: true, PermMembersAdmin: true,
    },
    "editor": {
        PermMemoryStore: true, PermMemoryRecall: true, PermAPIKeyCreate: true,
    },
    "viewer": {
        PermMemoryRecall: true,
    },
}

func HasPermission(role string, perm Permission) bool {
    perms, ok := rolePermissions[role]
    if !ok { return false }
    return perms[perm]
}
```

### 2. Tạo `shared/pkg/auth/rbac_test.go` [NEW]

```go
package auth_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestHasPermission_Admin(t *testing.T) {
    assert.True(t, HasPermission("admin", PermAdminForget))
    assert.True(t, HasPermission("admin", PermAPIKeyRevoke))
    assert.True(t, HasPermission("admin", PermAuditView))
}

func TestHasPermission_Editor(t *testing.T) {
    assert.True(t, HasPermission("editor", PermMemoryStore))
    assert.False(t, HasPermission("editor", PermAdminForget))
    assert.False(t, HasPermission("editor", PermAPIKeyRevoke))
}

func TestHasPermission_Viewer(t *testing.T) {
    assert.True(t, HasPermission("viewer", PermMemoryRecall))
    assert.False(t, HasPermission("viewer", PermMemoryStore))
    assert.False(t, HasPermission("viewer", PermMemoryForget))
}

func TestHasPermission_UnknownRole(t *testing.T) {
    assert.False(t, HasPermission("superuser", PermMemoryStore))
}
```

---

## Acceptance Criteria

- [ ] `HasPermission("admin", PermAdminForget)` → true
- [ ] `HasPermission("editor", PermAdminForget)` → false
- [ ] `HasPermission("viewer", PermMemoryStore)` → false
- [ ] `go test ./shared/pkg/auth/...` passes

## Files

```
shared/pkg/auth/rbac.go       [NEW]
shared/pkg/auth/rbac_test.go  [NEW]
```
