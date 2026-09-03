# TASK-PLAT-004 — Dev Mode Localhost Guard

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-004 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-PLAT-001](../solutions/SOL-PLAT-001-Auth-API-Key-JWT.md) §2.4 |
| **Component** | `gateway/internal/infra/middleware/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-003 |
| **Estimated** | 1h |

---

## Mục tiêu

Implement dev mode guard: khi `AUTH_DEV_MODE=true`, bypass auth nhưng chỉ cho phép requests từ localhost (127.0.0.1, ::1). Inject mock AuthContext với `tenant_id=dev-tenant`.

---

## Công việc cụ thể

### 1. Tạo `gateway/internal/infra/middleware/devmode.go` [NEW]

```go
package middleware

import (
    "context"
    "net"
    "net/http"
)

// DevModeGuard wraps the auth middleware when AUTH_DEV_MODE=true
// Only allows traffic from loopback addresses (127.0.0.1, ::1)
// Injects a mock AuthContext to skip all real auth checks
func DevModeGuard(devMode bool, next http.Handler) http.Handler {
    if !devMode {
        return next
    }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        host, _, err := net.SplitHostPort(r.RemoteAddr)
        if err != nil {
            // Fallback: try without port
            host = r.RemoteAddr
        }

        if host != "127.0.0.1" && host != "::1" && host != "localhost" {
            http.Error(w, `{"error":"dev_mode_localhost_only","message":"AUTH_DEV_MODE only accepts localhost traffic"}`,
                http.StatusForbidden)
            return
        }

        // Inject mock auth context (skip real JWT/API key validation)
        mockAuth := &AuthContext{
            TenantID: "dev-tenant",
            UserID:   "dev-user",
            Roles:    []string{"admin", "super_admin"},
            Scopes:   []string{"read", "write", "admin"},
            RateTier: "enterprise",
        }
        ctx := context.WithValue(r.Context(), authCtxKey, mockAuth)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2. Modify `gateway/internal/infra/middleware/auth.go` [MODIFY] — integrate dev mode

```go
// In AuthMiddleware constructor or Apply():
func NewAuthMiddleware(cfg AuthConfig, validator *JWTValidator, apiKeyUC port.APIKeyUseCase) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        // If dev mode, use DevModeGuard instead of real auth
        if cfg.DevMode {
            return DevModeGuard(true, next)
        }
        // Normal auth flow...
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // ... existing auth logic ...
        })
    }
}
```

### 3. Unit test `gateway/internal/infra/middleware/devmode_test.go` [NEW]

```go
package middleware_test

func TestDevModeGuard_LocalhostAllowed(t *testing.T) {
    handler := DevModeGuard(true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        auth := AuthFromContext(r.Context())
        assert.Equal(t, "dev-tenant", auth.TenantID)
        w.WriteHeader(http.StatusOK)
    }))
    req := httptest.NewRequest("GET", "/", nil)
    req.RemoteAddr = "127.0.0.1:12345"
    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDevModeGuard_NonLocalhostBlocked(t *testing.T) {
    handler := DevModeGuard(true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    req := httptest.NewRequest("GET", "/", nil)
    req.RemoteAddr = "192.168.1.100:12345"
    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestDevModeGuard_DevModeOff_PassThrough(t *testing.T) {
    called := false
    handler := DevModeGuard(false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.WriteHeader(http.StatusOK)
    }))
    req := httptest.NewRequest("GET", "/", nil)
    req.RemoteAddr = "192.168.1.100:12345"
    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    assert.True(t, called, "should pass through when dev mode is off")
}
```

---

## Acceptance Criteria

- [ ] `AUTH_DEV_MODE=true` + localhost → inject `tenant_id=dev-tenant`, role=admin
- [ ] `AUTH_DEV_MODE=true` + non-localhost → 403 Forbidden
- [ ] `AUTH_DEV_MODE=false` → normal auth middleware, DevModeGuard is a no-op
- [ ] Injected AuthContext has `RateTier=enterprise` (no rate limiting in dev mode)
- [ ] `go test ./gateway/internal/infra/middleware/...` passes

## Files

```
gateway/internal/infra/middleware/devmode.go       [NEW]
gateway/internal/infra/middleware/devmode_test.go  [NEW]
gateway/internal/infra/middleware/auth.go          [MODIFY — integrate dev mode]
```
