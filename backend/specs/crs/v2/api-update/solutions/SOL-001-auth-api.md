# SOL-001: Auth API Implementation

**Solution for**: [CR-001](../CR-001-auth-api.md)  
**Priority**: 🔴 Critical  
**Status**: Ready to Implement  
**Created**: 2026-06-18  
**Estimate**: 1–2 days

---

## Analysis

### What Already Exists

The `sm-auth` service is **fully implemented** at the gRPC level:

| Layer | Status | Location |
|-------|--------|----------|
| gRPC proto | ✅ Exists | `services/sm-auth/api/proto/v1/auth.proto` |
| gRPC generated code | ✅ Exists | `services/sm-auth/api/proto/v1/auth.pb.go`, `auth_grpc.pb.go` |
| `AuthUseCase.Login()` | ✅ Exists | `services/sm-auth/internal/usecase/auth.go:92` |
| `AuthHandler.Login()` (gRPC) | ✅ Exists | `services/sm-auth/internal/adapter/grpc/auth_handler.go:36` |

### What's Missing

Only the **gateway HTTP handler** (`AuthHandler` → `AuthUseCase` → `sm-auth` gRPC) and **router registration** are missing. The gateway currently has:
- **No** `gateway/internal/adapter/handler/auth.go`
- **No** `mux.HandleFunc("POST /v1/auth/...")` routes in `router.go`
- **No** exemption for `/v1/auth/login` and `/v1/auth/refresh` from JWT middleware

---

## Architecture Decision

Since `sm-auth` already supports gRPC (`SmAuthServiceClient`), the solution follows the **same forwarding pattern** used by all other console handlers:

```
Frontend HTTP → Gateway AuthHandler → sm-auth (gRPC via ServiceRegistry)
```

In monolith mode, this uses `bufconn` (in-process, no network).  
In standalone gateway mode, this uses the external gRPC address.

> **Do NOT** implement JWT issuance logic in the gateway itself. The gateway's job is forwarding. `sm-auth` handles credential validation and token minting.

### Response Transformation

The gRPC `AuthResponse` returns a single `token` string. The frontend expects a richer `LoginResponse`:
```json
{
  "access_token":  "...",
  "refresh_token": "...",
  "expires_in":    3600,
  "token_type":    "Bearer",
  "user": { "id": "...", "name": "...", "email": "...", "role": "...", "tenant_id": "..." }
}
```

The `AuthHandler` must **transform** the gRPC response into this JSON shape before writing to the HTTP response. This is the one exception to the pure-forward pattern.

---

## Implementation Plan

### Step 1: Create `gateway/internal/adapter/handler/auth.go`

```go
package handler

import (
    "encoding/json"
    "net/http"
    "log/slog"
    "github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// AuthHandler handles /v1/auth/* routes.
// Routes are forwarded to sm-auth service with response transformation.
type AuthHandler struct {
    registry port.ServiceRegistry
    logger   *slog.Logger
}

func NewAuthHandler(registry port.ServiceRegistry, logger *slog.Logger) *AuthHandler {
    return &AuthHandler{registry: registry, logger: logger}
}

// Login handles POST /v1/auth/login — no JWT required.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    // Forward body to sm-auth, transform response to LoginResponse shape
    ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// Logout handles POST /v1/auth/logout — JWT required.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// Me handles GET /v1/auth/me — JWT required, returns AuthUser from token claims.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}

// Refresh handles POST /v1/auth/refresh — no JWT required.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "sm-auth", h.logger)(w, r)
}
```

### Step 2: Register Routes in `gateway/internal/adapter/handler/router.go`

Add the `AuthHandler` parameter and register routes **before** all other routes (auth routes must be registered first to avoid middleware conflicts):

```go
// Modify Router() signature — add auth *AuthHandler parameter
func Router(
    auth *AuthHandler,    // ← Add this
    memory *MemoryHandler,
    // ... existing params
) http.Handler {
    mux := http.NewServeMux()

    // === /v1/auth/* — Public auth routes (no JWT middleware) ===
    mux.HandleFunc("POST /v1/auth/login",   auth.Login)
    mux.HandleFunc("POST /v1/auth/refresh", auth.Refresh)

    // === /v1/auth/* — Protected auth routes (JWT required) ===
    mux.HandleFunc("POST /v1/auth/logout", auth.Logout)
    mux.HandleFunc("GET /v1/auth/me",      auth.Me)

    // ... existing routes
}
```

### Step 3: Exempt Auth Routes from JWT Middleware

In `gateway/internal/infra/middleware/auth.go`, update the skip-list to bypass JWT validation for public auth endpoints:

```go
// skipAuthPaths lists paths that don't require JWT validation
var skipAuthPaths = map[string]bool{
    "/v1/auth/login":   true,
    "/v1/auth/refresh": true,
    // health check (if not already skipped)
}

func Auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if skipAuthPaths[r.URL.Path] {
            next.ServeHTTP(w, r)
            return
        }
        // ... existing JWT validation logic
    })
}
```

### Step 4: Wire in Bootstrap (`apps/memory/internal/bootstrap/gateway.go`)

Register `sm-auth` in the InProcessRegistry and wire the `AuthHandler` into `Router()`.

```go
// In bootstrap/gateway.go or bootstrap/supermemory.go
// sm-auth is already registered as part of Supermemory bootstrap
// Just ensure it's wired to the Router:
authH := handler.NewAuthHandler(registry, logger)
// Pass authH to Router(...)
```

### Step 5: Ensure `sm-auth` is Registered in Monolith Mode

Check `apps/memory/internal/bootstrap/supermemory.go` ensures `sm-auth` is registered under the name `"sm-auth"` in `InProcessRegistry`. From architecture docs, `sm-auth` is part of the Supermemory bootstrap module.

---

## Response Shape Mapping

The sm-auth gRPC proto returns `AuthResponse`. The gateway handler must translate this when forwarding:

| gRPC Field | HTTP JSON Field | Notes |
|------------|----------------|-------|
| `token` | `access_token` | Primary JWT |
| (from token claims) | `refresh_token` | sm-auth must implement refresh token logic |
| (hardcoded) | `expires_in` | Return from sm-auth or set default (3600) |
| (hardcoded) | `token_type` | Always `"Bearer"` |
| user fields | `user.id`, `user.email`, `user.role`, `user.tenant_id`, `user.name` | Extract from token claims or sm-auth response |

> **Action**: Check `services/sm-auth/api/proto/v1/auth.proto` — if `AuthResponse` doesn't include `refresh_token` and `user` object, update the proto and re-generate to add them.

---

## Files to Create/Modify

| Action | File |
|--------|------|
| **CREATE** | `gateway/internal/adapter/handler/auth.go` |
| **MODIFY** | `gateway/internal/adapter/handler/router.go` — add auth routes |
| **MODIFY** | `gateway/internal/infra/middleware/auth.go` — skip auth routes |
| **MODIFY** | `apps/memory/internal/bootstrap/gateway.go` — wire AuthHandler |
| **VERIFY** | `services/sm-auth/api/proto/v1/auth.proto` — ensure response includes refresh_token + user |

---

## Acceptance Criteria

- [ ] `POST /v1/auth/login` with `{ email, password }` returns `LoginResponse` with JWT
- [ ] `POST /v1/auth/login` does NOT require `Authorization` header
- [ ] `GET /v1/auth/me` returns `AuthUser` for valid JWT
- [ ] `POST /v1/auth/refresh` returns new `access_token` for valid `refresh_token`
- [ ] `POST /v1/auth/refresh` returns `401` for expired refresh token
- [ ] `POST /v1/auth/logout` invalidates session; frontend clears localStorage
- [ ] Auth middleware does NOT block `/v1/auth/login` and `/v1/auth/refresh`
