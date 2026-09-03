# TASK-PLAT-022 — RBAC RequirePermission HTTP Middleware

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-022 |
| **Wave** | 2 |
| **Solution** | [SOL-PLAT-007](../solutions/SOL-PLAT-007-RBAC-Authorization.md) §2 |
| **Component** | `shared/pkg/auth/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-PLAT-021 |
| **Estimated** | 2h |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** requireAdmin() guard in console.go; full RBAC middleware with permission matrix not created
---

## Mục tiêu

Tạo `RequirePermission()` HTTP middleware đọc `AuthContext` từ request context.

---

## Công việc cụ thể

### 1. Tạo `shared/pkg/auth/middleware.go` [NEW]

```go
package auth

import (
    "encoding/json"
    "fmt"
    "net/http"
)

func RequirePermission(perm Permission) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ac := AuthContextFromCtx(r.Context())
            if ac == nil {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(401)
                json.NewEncoder(w).Encode(map[string]string{
                    "error": "unauthorized", "message": "auth context missing",
                })
                return
            }
            for _, role := range ac.Roles {
                if HasPermission(role, perm) {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(403)
            json.NewEncoder(w).Encode(map[string]string{
                "error":   "forbidden",
                "message": fmt.Sprintf("roles %v cannot perform '%s'", ac.Roles, perm),
            })
        })
    }
}
```

### 2. Tạo `shared/pkg/auth/middleware_test.go` [NEW]

```go
package auth_test

func TestRequirePermission_Allowed(t *testing.T) {
    ctx := ContextWithAuth(context.Background(), &AuthContext{Roles: []string{"admin"}})
    req := httptest.NewRequest("POST", "/", nil).WithContext(ctx)
    rr := httptest.NewRecorder()
    handler := RequirePermission(PermAdminForget)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(200)
    }))
    handler.ServeHTTP(rr, req)
    assert.Equal(t, 200, rr.Code)
}

func TestRequirePermission_Forbidden(t *testing.T) {
    ctx := ContextWithAuth(context.Background(), &AuthContext{Roles: []string{"viewer"}})
    req := httptest.NewRequest("POST", "/", nil).WithContext(ctx)
    rr := httptest.NewRecorder()
    handler := RequirePermission(PermMemoryStore)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(200)
    }))
    handler.ServeHTTP(rr, req)
    assert.Equal(t, 403, rr.Code)
}

func TestRequirePermission_NoAuthContext(t *testing.T) {
    req := httptest.NewRequest("POST", "/", nil)
    rr := httptest.NewRecorder()
    handler := RequirePermission(PermMemoryStore)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
    handler.ServeHTTP(rr, req)
    assert.Equal(t, 401, rr.Code)
}
```

---

## Acceptance Criteria

- [ ] Admin role → 200 for all permissions
- [ ] Viewer role → 403 for PermMemoryStore
- [ ] No AuthContext → 401
- [ ] Error body contains role and permission info

## Files

```
shared/pkg/auth/middleware.go       [NEW]
shared/pkg/auth/middleware_test.go  [NEW]
```
