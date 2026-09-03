# SOL-000: Index — API Update Solutions (v2)

**Directory**: `specs/crs/v2/api-update/solutions/`  
**Created**: 2026-06-18  
**Architecture Reference**: [`specs/architecture.md`](../../../architecture.md)

---

## Solution Overview

| Solution | CR | Priority | Estimate | Status |
|----------|----|----------|----------|--------|
| [SOL-001](./SOL-001-auth-api.md) | CR-001 | 🔴 Critical | 1–2 days | Ready |
| [SOL-002](./SOL-002-org-sdk-api.md) | CR-002 | 🟡 High | 2–3 days | Ready |
| [SOL-003](./SOL-003-session-query-params.md) | CR-003 | 🟡 High | 1 day | Ready |
| [SOL-004](./SOL-004-response-schema-contracts.md) | CR-004 | 🟠 Medium | 2–3 days | Ready |

**Total estimate**: ~7–9 working days

---

## Architecture Context

```
Frontend (React/TS)
  ↓ HTTP (VITE_API_BASE_URL → :8080)
VNP Gateway (gateway/ + apps/memory)
  ├── Middleware: Recovery → RequestID → Logger → CORS → Auth → [ErrorNormalizer]
  ├── Router: router.go (150+ routes + new auth/org/sdk routes)
  └── Handlers:
       ├── auth.go          [SOL-001 NEW]
       ├── console_org.go   [SOL-002 NEW]
       ├── console_sdk.go   [SOL-002 NEW]
       ├── console.go       [SOL-003 MODIFY SessionHandler]
       └── handler.go       [SOL-003/SOL-004 MODIFY — add normalisers]
  ↓ gRPC (bufconn in monolith / TCP in distributed)
Backend Services
  ├── sm-auth              → [SOL-001] Login, Logout, Me, Refresh
  ├── vnp-admin            → [SOL-002] Org settings, Members, Roles, SDK keys, Webhooks
  ├── zep-core             → [SOL-003] Sessions (normalised pagination)
  ├── vnp-search-hub       → [SOL-004] Memory search with facets
  ├── vnp-observability    → [SOL-004] Metrics, errors, traces
  └── vnp-dashboard        → [SOL-004] Health with correct HealthStatus enum
```

---

## Implementation Sequence

```
Phase 1 — Day 1–2: SOL-001 (Auth API)
  Reason: All frontend API calls fail without working authentication.
  
  1. Create gateway/internal/adapter/handler/auth.go
  2. Add auth routes to router.go (exempt from JWT middleware)
  3. Update auth middleware skip-list
  4. Verify sm-auth gRPC response includes refresh_token + user object
  5. Wire AuthHandler in bootstrap/gateway.go
  6. Test: POST /v1/auth/login, /logout, /me, /refresh

Phase 2 — Day 2–4: SOL-002 (Org & SDK API)
  Reason: Org Settings and SDK pages are non-functional without these endpoints.
  
  1. Create gateway/internal/adapter/handler/console_org.go
  2. Create gateway/internal/adapter/handler/console_sdk.go
  3. Add org/sdk routes to router.go
  4. Add Webhook entity to vnp-platform domain
  5. Add OrgSettings view struct
  6. Implement org + sdk use cases in vnp-platform
  7. Add webhooks migration
  8. Wire OrgHandler + SDKHandler in bootstrap

Phase 3 — Day 4–5: SOL-003 (Session Pagination)
  Reason: Session list page cannot filter or paginate without this.
  
  1. Add NormalizePaginatedResponse utility to handler.go
  2. Add ForwardToServiceWithNorm utility
  3. Update SessionHandler.ListSessions to use normaliser
  4. Verify sub-route response shapes (diff, working-memory)
  5. Add integration tests

Phase 4 — Day 5–8: SOL-004 (Schema Contracts)
  Reason: Correctness issues causing silent data mismatches in UI.
  
  1. Implement ErrorNormalizer middleware
  2. Fix HealthStatus enum values in vnp-platform
  3. Add facets to MemorySearchResult in vnp-search-hub
  4. Standardise memory ID format (engine:local_id)
  5. Add service filter to observability errors endpoint
  6. Verify MetricsResponse shape
```

---

## Key Design Principles Applied

### 1. Gateway as Transformation Layer (not Logic Layer)

All solutions follow the **forward + transform** pattern:
- The gateway forwards to backend services
- The gateway normalises responses to match frontend contracts
- Business logic stays in services, not in the gateway

### 2. No Changes to Per-Engine Services

SOL-001 through SOL-003 deliberately avoid modifying engine services (`cognee-*`, `graphiti-*`, etc.). Changes are isolated to:
- `gateway/internal/adapter/handler/` — new/modified handlers
- `gateway/internal/infra/middleware/` — error normaliser
- `services/vnp-platform/` — admin/org/sdk domain
- `services/sm-auth/` — verify/extend proto if needed

### 3. Backward Compatible

All changes are additive:
- New routes don't conflict with existing routes
- Error normaliser only rewrites the `{ error: {} }` wrapper pattern
- Pagination normaliser detects and transforms without breaking already-correct responses

### 4. `sm-auth` is the Auth Backend

The architecture already has `sm-auth` as the Supermemory auth service (registered in monolith bootstrap via `supermemory.go`). There is no need to create a new auth service.

---

## File Change Summary

### New Files

| File | Purpose |
|------|---------|
| `gateway/internal/adapter/handler/auth.go` | Auth API handler (SOL-001) |
| `gateway/internal/adapter/handler/console_org.go` | Org settings handler (SOL-002) |
| `gateway/internal/adapter/handler/console_sdk.go` | SDK key/webhook handler (SOL-002) |
| `gateway/internal/infra/middleware/error_normalizer.go` | Error response normaliser (SOL-004) |
| `deployment/dev/migrations/XXXX_add_webhooks.sql` | Webhooks table (SOL-002) |

### Modified Files

| File | Change |
|------|--------|
| `gateway/internal/adapter/handler/router.go` | +15 routes (auth, org, sdk) |
| `gateway/internal/adapter/handler/handler.go` | +normaliser utilities (SOL-003/004) |
| `gateway/internal/adapter/handler/console.go` | `ListSessions` uses normaliser |
| `gateway/internal/infra/middleware/auth.go` | Skip auth/login, auth/refresh |
| `apps/memory/internal/bootstrap/gateway.go` | Wire new handlers |
| `services/vnp-platform/internal/domain/admin/entity.go` | Add `Webhook`, `OrgSettings`, fix `HealthStatus` |
| `services/sm-auth/api/proto/v1/auth.proto` | Add `refresh_token` + `user` to `AuthResponse` if missing |
