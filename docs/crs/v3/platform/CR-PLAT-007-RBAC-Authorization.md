# Change Request: CR-PLAT-007 — RBAC Authorization Middleware

**CR ID:** CR-PLAT-007
**Component:** `backend/gateway`, `backend/shared/pkg/auth`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F14](../../../features/14-authentication-multi-tenancy/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P4-03 | Enterprise Architect | Không có fine-grained access control |
| PP-P2-04 | Platform Engineer | Admin/viewer roles không được enforce |

**Before:** Mọi authenticated user đều có full access.
**After:** Role-based enforcement: admin/editor/viewer per tenant.

---

## 2. Role Matrix

| Permission | admin | editor | viewer |
|---|---|---|---|
| memory.store | ✅ | ✅ | ❌ |
| memory.recall | ✅ | ✅ | ✅ |
| memory.forget | ✅ | ❌ | ❌ |
| admin.forget_user | ✅ | ❌ | ❌ |
| admin.view_members | ✅ | ❌ | ❌ |
| org.settings.write | ✅ | ❌ | ❌ |
| api_keys.create | ✅ | ✅ | ❌ |
| api_keys.revoke | ✅ | ❌ | ❌ |
| governance.view_audit | ✅ | ❌ | ❌ |

---

## 3. Implementation

```go
// shared/pkg/auth/rbac.go
type Permission string

const (
    PermMemoryStore   Permission = "memory:store"
    PermMemoryRecall  Permission = "memory:recall"
    PermMemoryForget  Permission = "memory:forget"
    PermAdminForget   Permission = "admin:forget"
    PermOrgWrite      Permission = "org:write"
    PermAPIKeyCreate  Permission = "api_key:create"
    PermAPIKeyRevoke  Permission = "api_key:revoke"
    PermAuditView     Permission = "audit:view"
)

// RBACMiddleware — requires specific permission
func RequirePermission(perm Permission) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := auth.RoleFromContext(r.Context())
            if !hasPermission(role, perm) {
                writeError(w, 403, "forbidden",
                    fmt.Sprintf("role '%s' cannot perform '%s'", role, perm))
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 4. API Endpoints

| Method | Path | Required Permission |
|---|---|---|
| `POST` | `/v1/memory/store` | `memory:store` |
| `POST` | `/v1/memory/recall` | `memory:recall` |
| `POST` | `/v1/admin/forget` | `admin:forget` |
| `GET` | `/v1/admin/audit` | `audit:view` |
| `PUT` | `/v1/console/org/settings` | `org:write` |

---

## 5. Acceptance Criteria

- [ ] viewer: cannot store or forget
- [ ] editor: can store/recall, cannot forget user
- [ ] admin: full access
- [ ] 403 with role and permission in error body
- [ ] Role from JWT claims (`roles` field)
- [ ] Unit tests for each role/permission combination
