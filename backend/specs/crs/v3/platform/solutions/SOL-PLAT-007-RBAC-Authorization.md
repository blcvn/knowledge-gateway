# Solution: SOL-PLAT-007 — RBAC Authorization Middleware

**CR:** CR-PLAT-007
**TDD refs:** `architecture/01-gateway.md §2`, `architecture/09-shared-packages.md §2`
**Version:** v3/platform

---

## 1. Architecture Analysis

Gateway TDD shows `AuthContext` already injects `Roles []string` into request context:
```go
// gateway/internal/infra/middleware/auth.go (existing)
type AuthContext struct {
    TenantID string
    UserID   string
    Roles    []string   // populated from JWT claims["roles"]
    RateTier string
}
```

Solution: add `shared/pkg/auth/rbac.go` with `RequirePermission()` middleware that reads from this AuthContext.

---

## 2. Permission Model

```go
// shared/pkg/auth/rbac.go [NEW]
package auth

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

// rolePermissions — immutable permission matrix
var rolePermissions = map[string]map[Permission]bool{
    "admin": {
        PermMemoryStore: true, PermMemoryRecall: true, PermMemoryForget: true,
        PermAdminForget: true, PermAuditView: true, PermOrgWrite: true,
        PermAPIKeyCreate: true, PermAPIKeyRevoke: true, PermMembersAdmin: true,
    },
    "editor": {
        PermMemoryStore: true, PermMemoryRecall: true,
        PermAPIKeyCreate: true,
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

// RequirePermission — HTTP middleware
func RequirePermission(perm Permission) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ac := AuthContextFromCtx(r.Context())
            if ac == nil {
                writeError(w, 401, "unauthorized", "auth context missing")
                return
            }
            for _, role := range ac.Roles {
                if HasPermission(role, perm) {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            writeError(w, 403, "forbidden",
                fmt.Sprintf("role '%s' cannot perform '%s'", ac.Roles, perm))
        })
    }
}
```

---

## 3. Router Integration

```go
// gateway/adapter/handler/router.go [MODIFY]
import "vnp-memory/shared/pkg/auth"

// Memory API — role enforcement
r.With(auth.RequirePermission(auth.PermMemoryStore)).
    Post("/v1/memory/store", memoryHandler.Store)

r.With(auth.RequirePermission(auth.PermMemoryRecall)).
    Post("/v1/memory/recall", memoryHandler.Recall)

r.With(auth.RequirePermission(auth.PermMemoryForget)).
    Post("/v1/memory/forget", memoryHandler.Forget)

// Admin API
r.With(auth.RequirePermission(auth.PermAdminForget)).
    Post("/v1/admin/forget", adminHandler.ForgetUser)

r.With(auth.RequirePermission(auth.PermAuditView)).
    Get("/v1/admin/audit", adminHandler.GetAuditLog)

r.With(auth.RequirePermission(auth.PermOrgWrite)).
    Put("/v1/console/org/settings", orgHandler.UpdateSettings)

r.With(auth.RequirePermission(auth.PermMembersAdmin)).
    Post("/v1/console/org/members/invite", orgHandler.InviteMember)

r.With(auth.RequirePermission(auth.PermAPIKeyRevoke)).
    Delete("/v1/console/sdk/keys/{id}", sdkHandler.RevokeKey)
```

---

## 4. Tests

```go
// shared/pkg/auth/rbac_test.go [NEW]
func TestHasPermission_AdminAllAccess(t *testing.T) {
    assert.True(t, HasPermission("admin", PermAdminForget))
    assert.True(t, HasPermission("admin", PermAuditView))
    assert.True(t, HasPermission("admin", PermAPIKeyRevoke))
}

func TestHasPermission_EditorLimited(t *testing.T) {
    assert.True(t, HasPermission("editor", PermMemoryStore))
    assert.False(t, HasPermission("editor", PermAdminForget))
    assert.False(t, HasPermission("editor", PermAPIKeyRevoke))
}

func TestHasPermission_ViewerReadOnly(t *testing.T) {
    assert.True(t, HasPermission("viewer", PermMemoryRecall))
    assert.False(t, HasPermission("viewer", PermMemoryStore))
    assert.False(t, HasPermission("viewer", PermMemoryForget))
}

func TestRequirePermission_Middleware_Forbidden(t *testing.T) {
    ctx := ContextWithAuth(context.Background(), &AuthContext{Roles: []string{"viewer"}})
    req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
    rr := httptest.NewRecorder()
    handler := RequirePermission(PermMemoryStore)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
    }))
    handler.ServeHTTP(rr, req)
    assert.Equal(t, 403, rr.Code)
}
```

---

## 5. File Changes

| File | Action |
|---|---|
| `shared/pkg/auth/rbac.go` | **[NEW]** Permission model + middleware |
| `shared/pkg/auth/rbac_test.go` | **[NEW]** Unit tests |
| `shared/pkg/auth/context.go` | **[MODIFY]** add `AuthContextFromCtx()` helper |
| `gateway/adapter/handler/router.go` | **[MODIFY]** add permission guards |
| `shared/pkg/auth/go.mod` | **[MODIFY or NEW]** ensure module exists |
