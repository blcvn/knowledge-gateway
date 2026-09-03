# TASK-002: Register Auth Routes in `router.go` + Exempt from JWT Middleware

**Solution**: [SOL-001](../solutions/SOL-001-auth-api.md)  
**CR**: CR-001  
**Priority**: 🔴 Critical  
**Estimate**: 1 hour  
**Status**: ✅ Implemented
**Depends on**: TASK-001

---

## Context

`router.go` (line 17) defines `Router(...)`. The `auth *AuthHandler` parameter is **not yet in the signature** and **no auth routes are registered**. The `gateway.go` bootstrap (line 63) passes `authH` to `Router()` but `Router()` does not yet accept it.

The auth middleware is applied at line 87 of `gateway.go`:
```go
finalRouter = middleware.Auth(authUC, logger)(router)
```

The `middleware.Auth` function must skip `/v1/auth/login` and `/v1/auth/refresh` since they are public endpoints.

---

## Exact Changes

### 1. Modify `gateway/internal/adapter/handler/router.go`

**Change 1** — Add `auth *AuthHandler` parameter to `Router()` signature (after `admin *AdminHandler`, before `// Console handlers`):

```go
// Current (line 25):
admin *AdminHandler,
// Console handlers (SOL-002)

// Change to:
admin *AdminHandler,
auth  *AuthHandler,
// Console handlers (SOL-002)
```

**Change 2** — Add auth routes section immediately after the `/v1/admin/*` section (after line 118):

```go
// === /v1/auth/* — Authentication (login/logout/me/refresh) ===
// NOTE: login and refresh are PUBLIC — they bypass JWT middleware (see middleware/auth.go)
mux.HandleFunc("POST /v1/auth/login",   auth.Login)
mux.HandleFunc("POST /v1/auth/refresh", auth.Refresh)
mux.HandleFunc("POST /v1/auth/logout",  auth.Logout)
mux.HandleFunc("GET /v1/auth/me",       auth.Me)
```

**Change 3** — Add `org *OrgHandler, sdk *SDKHandler` parameters and their routes (if `console_org.go` and `console_sdk.go` are not yet created, add placeholder `// TODO` comments for the routes):

```go
// Add after observabilityH parameter in Router() signature:
org *OrgHandler,
sdk *SDKHandler,
```

> **Note**: `gateway.go` line 67 already passes `orgH, sdkH` to `Router()`. The `Router()` signature must accept them.

**Change 4** — Add org/sdk routes (after governance section, before pipelines):

```go
// === /v1/console/org/* — Org settings (admin role, tenant-scoped) ===
mux.HandleFunc("GET /v1/console/org/settings",  org.GetSettings)
mux.HandleFunc("PUT /v1/console/org/settings",  org.UpdateSettings)
mux.HandleFunc("GET /v1/console/org/members",   org.ListMembers)
mux.HandleFunc("GET /v1/console/org/roles",     org.ListRoles)

// === /v1/console/sdk/* — SDK management (admin role, tenant-scoped) ===
mux.HandleFunc("GET /v1/console/sdk/keys",             sdk.ListKeys)
mux.HandleFunc("POST /v1/console/sdk/keys",            sdk.CreateKey)
mux.HandleFunc("DELETE /v1/console/sdk/keys/{id}",     sdk.DeleteKey)
mux.HandleFunc("GET /v1/console/sdk/rate-limits",      sdk.GetRateLimits)
mux.HandleFunc("GET /v1/console/sdk/webhooks",         sdk.ListWebhooks)
mux.HandleFunc("POST /v1/console/sdk/webhooks",        sdk.CreateWebhook)
mux.HandleFunc("DELETE /v1/console/sdk/webhooks/{id}", sdk.DeleteWebhook)
```

### 2. Modify `gateway/internal/infra/middleware/auth.go`

Find the auth middleware function and add a skip-list for public auth paths. The exact implementation depends on the current middleware code, but the pattern must be:

```go
// Public paths that don't require JWT validation
var publicPaths = map[string]bool{
    "/v1/auth/login":   true,
    "/v1/auth/refresh": true,
}

// In the Auth middleware handler, before JWT validation:
if publicPaths[r.URL.Path] {
    next.ServeHTTP(w, r)
    return
}
```

---

## Verification

After making changes, run:
```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./gateway/...
go build ./apps/memory/...
```

Both must compile without errors.

---

## Acceptance Criteria

- [ ] `Router()` signature includes `auth *AuthHandler, org *OrgHandler, sdk *SDKHandler` parameters
- [ ] 4 auth routes registered: `POST /v1/auth/login`, `POST /v1/auth/refresh`, `POST /v1/auth/logout`, `GET /v1/auth/me`
- [ ] 4 org routes registered: `GET/PUT /v1/console/org/settings`, `GET /v1/console/org/members`, `GET /v1/console/org/roles`
- [ ] 7 sdk routes registered: keys (list/create/delete), rate-limits, webhooks (list/create/delete)
- [ ] `middleware.Auth` skips JWT check for `POST /v1/auth/login` and `POST /v1/auth/refresh`
- [ ] `go build ./gateway/... ./apps/memory/...` succeeds

---

**Audit Note:** Router() updated: auth/*AuthHandler param added; /v1/auth/* routes registered; isPublicPath includes login+refresh
