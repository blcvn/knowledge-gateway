# CR-000: Index — API Update Change Requests (v2)

**Directory**: `specs/crs/v2/api-update/`  
**Updated**: 2026-06-18  
**Cross-reference**: [`specs/backend-api-specs.md`](../../backend-api-specs.md) vs [`ui/specs/frontend-backend-api-specs.md`](../../../ui/specs/frontend-backend-api-specs.md)

---

## Summary of Gaps

After comparing the backend's implemented API (derived from `gateway/internal/adapter/handler/router.go`) against the frontend's actual HTTP calls (`ui/src/services/*.ts`), the following gaps were identified:

| CR | Priority | Status | Description |
|----|----------|--------|-------------|
| [CR-001](./CR-001-auth-api.md) | 🔴 Critical | Open | Auth API (`/v1/auth/*`) — completely missing, frontend cannot authenticate |
| [CR-002](./CR-002-org-sdk-api.md) | 🟡 High | Open | Org & SDK API (`/v1/console/org/*`, `/v1/console/sdk/*`) — missing 11 endpoints |
| [CR-003](./CR-003-session-query-params.md) | 🟡 High | Open | Session API — `GET /v1/console/sessions` must support 7 query params + `PaginatedResponse` shape |
| [CR-004](./CR-004-response-schema-contracts.md) | 🟠 Medium | Open | Response schema contracts — TypeScript types must match backend JSON exactly |

---

## Complete Gap Table

### ❌ Missing Endpoints (Not Implemented in Gateway)

| Method | Path | CR | Backend Service Needed |
|--------|------|----|----------------------|
| `POST` | `/v1/auth/login` | CR-001 | `sm-auth` |
| `POST` | `/v1/auth/logout` | CR-001 | `sm-auth` |
| `GET` | `/v1/auth/me` | CR-001 | `sm-auth` |
| `POST` | `/v1/auth/refresh` | CR-001 | `sm-auth` |
| `GET` | `/v1/console/org/settings` | CR-002 | `vnp-admin` |
| `PUT` | `/v1/console/org/settings` | CR-002 | `vnp-admin` |
| `GET` | `/v1/console/org/members` | CR-002 | `vnp-admin` |
| `GET` | `/v1/console/org/roles` | CR-002 | `vnp-admin` |
| `GET` | `/v1/console/sdk/keys` | CR-002 | `vnp-admin` |
| `POST` | `/v1/console/sdk/keys` | CR-002 | `vnp-admin` |
| `DELETE` | `/v1/console/sdk/keys/{id}` | CR-002 | `vnp-admin` |
| `GET` | `/v1/console/sdk/rate-limits` | CR-002 | `vnp-admin` |
| `GET` | `/v1/console/sdk/webhooks` | CR-002 | `vnp-admin` |
| `POST` | `/v1/console/sdk/webhooks` | CR-002 | `vnp-admin` |
| `DELETE` | `/v1/console/sdk/webhooks/{id}` | CR-002 | `vnp-admin` |

### ⚠️ Implemented but Schema/Behaviour Needs Verification

| Endpoint | Issue | CR |
|----------|-------|----|
| `GET /v1/console/sessions` | Must support 7 query params + return `PaginatedResponse<Session>` with camelCase aliases | CR-003 |
| `GET /v1/console/sessions/{id}` | Must return `Conversation { session_id, messages[] }` shape | CR-003 |
| `GET /v1/console/sessions/{id}/diff` | Must return `SessionDiff { added[], updated[], deleted[] }` | CR-003 |
| `GET /v1/console/dashboard/health` | `status` must be `"Healthy"`/`"Warning"`/`"Critical"` (capital) | CR-004 |
| `POST /v1/console/memory/search` | Must return `facets.byEngine` + `facets.byType` | CR-004 |
| `GET /v1/console/memory/{id}/neighbors` | Must accept `strategy` + `limit` query params | CR-004 |
| `GET /v1/console/observability/errors` | Must accept `?service=xxx` filter | CR-004 |
| All error responses | Backend wraps in `{ error: {...} }`, frontend expects flat `{ message, code, status }` | CR-004 |

### ✅ Verified — Endpoint Exists and Should Work

All other endpoints in `specs/backend-api-specs.md` sections 2–11 are implemented in the gateway router and map correctly to the frontend's calls.

---

## Recommended Implementation Order

```
Phase 1 (Blocking): CR-001 — Auth API
  → Without this, the frontend cannot authenticate at all

Phase 2 (Functional gaps): CR-002 — Org & SDK API
  → Org Settings and SDK pages are completely non-functional

Phase 3 (Data quality): CR-003 — Session query params
  → Session list filtering/pagination is broken

Phase 4 (Correctness): CR-004 — Response schema contracts
  → Silent data mismatches cause UI to show wrong values
```
