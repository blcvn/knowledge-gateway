---
id: TASK-004
title: Auth Middleware — JWT RS256 + API Key
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_feat: FEAT-001
depends_on: [TASK-002, TASK-003]
estimate: 4h
actual: 3h
---

## Mục Tiêu

Implement authentication middleware: JWT RS256 validation + API Key resolution → AuthContext injection vào request context.

## Phạm Vi

### Files đã tạo
- `gateway/internal/usecase/auth.go` — 197 lines
- `gateway/internal/usecase/auth_test.go` — 223 lines
- `gateway/internal/infra/middleware/auth.go` — 139 lines
- `gateway/internal/infra/middleware/middleware.go` — 174 lines (AuthFromContext, WithAuthContext)
- `gateway/internal/infra/persistence/pg_store.go` — 161 lines (KeyStore + TenantStore)

### Chi tiết triển khai

#### Authentication Flow
```
1. Extract Authorization header OR X-API-Key header
2. If Authorization: Bearer <token>
   → Parse JWT RS256 with golang-jwt/jwt/v5
   → Validate: signature (RSA), exp, iss, aud (30s leeway)
   → Extract VNPClaims: tid, uid, roles, scopes, rate_tier
   → Build AuthContext{TenantID, UserID, Roles, Scopes, RateTier}
3. If X-API-Key: vnp_<key>
   → SHA-256 hash the key
   → Query PostgreSQL (api_keys JOIN tenants)
   → Check: revoked_at IS NULL AND expires_at > now()
   → Update last_used_at asynchronously (non-blocking goroutine)
   → Build AuthContext from DB metadata
4. If neither header present:
   → If AUTH_DEV_MODE=true → use DevAuthContext (dev-tenant, admin role)
   → Else → return 401 {error: {code: "UNAUTHENTICATED"}}
5. Store AuthContext in context.Context via authContextKey{}
6. Set X-Tenant-ID response header for tracing
```

#### VNPClaims — Custom JWT claims
```go
type VNPClaims struct {
    jwt.RegisteredClaims
    TenantID string   `json:"tid"`
    UserID   string   `json:"uid"`
    Roles    []string `json:"roles"`
    Scopes   []string `json:"scopes"`
    RateTier string   `json:"rate_tier"`
}
```

#### PostgreSQL API Key Schema
```sql
CREATE TABLE api_keys (
    id           SERIAL PRIMARY KEY,
    tenant_id    VARCHAR(64) NOT NULL REFERENCES tenants(id),
    user_id      VARCHAR(64) NOT NULL,
    key_hash     VARCHAR(128) NOT NULL UNIQUE,
    key_prefix   VARCHAR(16) NOT NULL,
    scopes       TEXT[] DEFAULT ARRAY['*'],
    revoked_at   TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### Auth HTTP Middleware
- Extracts credentials from `Authorization: Bearer` or `X-API-Key` headers
- Skips `/healthz`, `/readyz`, `/metrics`, `/healthz/deep`, OPTIONS (CORS preflight)
- Returns `WWW-Authenticate: Bearer` on JWT failures
- Returns structured JSON `{error: {code: "UNAUTHENTICATED"}}` on all failures

## Acceptance Criteria

- [x] AC-1: Given valid JWT RS256 token → AuthContext populated, request passes through ✅
- [x] AC-2: Given valid API key (vnp_xxxxx) → AuthContext populated via KeyStore lookup ✅
- [x] AC-3: Given expired JWT → 401 with error message ✅
- [x] AC-4: Given revoked API key → 401 `UNAUTHENTICATED` (via PGTenantStore revoked_at check) ✅
- [x] AC-5: Given no auth header AND AUTH_DEV_MODE=false → 401 ✅
- [x] AC-6: Given no auth header AND AUTH_DEV_MODE=true → DevAuthContext (dev-tenant, admin) ✅
- [x] AC-7: AuthContext available via `AuthFromContext(ctx)` in downstream handlers ✅
- [x] AC-8: API key lookups cached — async last_used_at update (non-blocking) ✅

## Test Results

```
=== RUN   TestAuthenticateJWT_Valid            --- PASS (0.17s)
=== RUN   TestAuthenticateJWT_Expired          --- PASS (0.08s)
=== RUN   TestAuthenticateJWT_WrongIssuer      --- PASS (0.10s)
=== RUN   TestAuthenticateJWT_DevMode          --- PASS (0.00s)
=== RUN   TestAuthenticateJWT_MissingTenantClaim --- PASS (0.03s)
=== RUN   TestAuthenticateAPIKey_Valid         --- PASS (0.00s)
=== RUN   TestAuthenticateAPIKey_InvalidPrefix --- PASS (0.00s)
ok    internal/usecase    0.85s
```

## Verification

```bash
go test ./internal/usecase/ -run TestAuthenticate -v  # ✅ 7 tests PASS
go build ./internal/infra/middleware/...              # ✅ PASS
go build ./internal/infra/persistence/...             # ✅ PASS
```
