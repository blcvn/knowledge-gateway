# SOL-002: Org Settings & SDK Management API Implementation

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** console_org.go + console_sdk.go created; /v1/console/org/* and /v1/console/sdk/* routes registered; OrgSettings/OrgMember/OrgRole/Webhook entities added to vnp-platform; webhooks migration 0052 created  

**Solution for**: [CR-002](../CR-002-org-sdk-api.md)  
**Priority**: 🟡 High  
**Status**: Ready to Implement  
**Created**: 2026-06-18  
**Estimate**: 2–3 days

---

## Analysis

### What Already Exists

The `vnp-platform` service (`services/vnp-platform`) already has domain entities for `Tenant`, `User`, and `APIKey`. The `vnp-admin` service handles tenant/key management and is already forwarded from `/v1/admin/*` routes.

The gap is:
1. **No org-scoped endpoints** — the existing `/v1/admin/*` endpoints are global admin operations; the frontend needs tenant-scoped org management at `/v1/console/org/*`
2. **No SDK key management endpoints** — existing `/v1/admin/tenants/{id}/keys` is a super-admin operation; the frontend needs self-service key management at `/v1/console/sdk/*`
3. **No webhook management** — entirely missing from both backend and frontend

### Architectural Decision

Route all new endpoints through `vnp-admin` (already exists, already handles tenant/key data). The distinction is:
- `/v1/console/org/*` — **tenant-scoped** (admin sees their own org's data)
- `/v1/console/sdk/*` — **tenant-scoped** (admin manages their own keys/webhooks)
- Both require `requireAdmin` (not `requireSuperAdmin`)

The `vnp-admin` service should scope all reads/writes to `X-Tenant-ID` from the JWT context.

---

## Implementation Plan

### Step 1: Create Two New Handler Files

#### `gateway/internal/adapter/handler/console_org.go`

```go
package handler

import (
    "log/slog"
    "net/http"
    "github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// OrgHandler handles /v1/console/org/* routes.
// All routes require admin role and are scoped to the current tenant.
type OrgHandler struct {
    registry port.ServiceRegistry
    logger   *slog.Logger
}

func NewOrgHandler(registry port.ServiceRegistry, logger *slog.Logger) *OrgHandler {
    return &OrgHandler{registry: registry, logger: logger}
}

// GetSettings handles GET /v1/console/org/settings
func (h *OrgHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// UpdateSettings handles PUT /v1/console/org/settings
func (h *OrgHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// ListMembers handles GET /v1/console/org/members
func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// ListRoles handles GET /v1/console/org/roles
func (h *OrgHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}
```

#### `gateway/internal/adapter/handler/console_sdk.go`

```go
package handler

import (
    "log/slog"
    "net/http"
    "github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// SDKHandler handles /v1/console/sdk/* routes.
// All routes require admin role. Keys/webhooks are tenant-scoped.
type SDKHandler struct {
    registry port.ServiceRegistry
    logger   *slog.Logger
}

func NewSDKHandler(registry port.ServiceRegistry, logger *slog.Logger) *SDKHandler {
    return &SDKHandler{registry: registry, logger: logger}
}

// ListKeys handles GET /v1/console/sdk/keys
func (h *SDKHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// CreateKey handles POST /v1/console/sdk/keys
// IMPORTANT: raw_key is returned only once — vnp-admin must generate & return it here
func (h *SDKHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// DeleteKey handles DELETE /v1/console/sdk/keys/{id}
func (h *SDKHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// GetRateLimits handles GET /v1/console/sdk/rate-limits
func (h *SDKHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// ListWebhooks handles GET /v1/console/sdk/webhooks
func (h *SDKHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// CreateWebhook handles POST /v1/console/sdk/webhooks
func (h *SDKHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}

// DeleteWebhook handles DELETE /v1/console/sdk/webhooks/{id}
func (h *SDKHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) { return }
    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
}
```

### Step 2: Register Routes in `router.go`

```go
// Add to Router() function signature:
// org *OrgHandler, sdk *SDKHandler

// === /v1/console/org/* — Org settings (admin role, tenant-scoped) ===
mux.HandleFunc("GET /v1/console/org/settings",  org.GetSettings)
mux.HandleFunc("PUT /v1/console/org/settings",  org.UpdateSettings)
mux.HandleFunc("GET /v1/console/org/members",   org.ListMembers)
mux.HandleFunc("GET /v1/console/org/roles",     org.ListRoles)

// === /v1/console/sdk/* — SDK management (admin role, tenant-scoped) ===
mux.HandleFunc("GET /v1/console/sdk/keys",               sdk.ListKeys)
mux.HandleFunc("POST /v1/console/sdk/keys",              sdk.CreateKey)
mux.HandleFunc("DELETE /v1/console/sdk/keys/{id}",       sdk.DeleteKey)
mux.HandleFunc("GET /v1/console/sdk/rate-limits",        sdk.GetRateLimits)
mux.HandleFunc("GET /v1/console/sdk/webhooks",           sdk.ListWebhooks)
mux.HandleFunc("POST /v1/console/sdk/webhooks",          sdk.CreateWebhook)
mux.HandleFunc("DELETE /v1/console/sdk/webhooks/{id}",   sdk.DeleteWebhook)
```

### Step 3: Extend `vnp-admin` Service

The `vnp-admin` service (in `services/vnp-platform/internal/domain/admin/`) needs new use cases and/or HTTP handlers for:

#### New Endpoints Needed in `vnp-admin`

| Path | Action |
|------|--------|
| `GET /v1/console/org/settings` | Return `OrgSettings` for tenant from JWT |
| `PUT /v1/console/org/settings` | Update org settings (name, slug, limits) |
| `GET /v1/console/org/members` | List `User[]` scoped to tenant |
| `GET /v1/console/org/roles` | Return role definitions (can be static config) |
| `GET /v1/console/sdk/keys` | List `APIKey[]` for tenant (without `key_hash`) |
| `POST /v1/console/sdk/keys` | Create key, return `CreateKeyResponse` with `raw_key` |
| `DELETE /v1/console/sdk/keys/{id}` | Revoke key |
| `GET /v1/console/sdk/rate-limits` | Return rate limit tiers |
| `GET/POST/DELETE /v1/console/sdk/webhooks` | CRUD for webhooks |

#### New Domain Entity: `Webhook`

Add to `services/vnp-platform/internal/domain/admin/entity.go`:

```go
type Webhook struct {
    ID          uuid.UUID `json:"id"`
    TenantID    uuid.UUID `json:"tenant_id"`
    URL         string    `json:"url"`
    Events      []string  `json:"events"`
    Status      string    `json:"status"`       // "active" | "paused" | "failed"
    SecretHash  string    `json:"-"`             // SHA-256, never exposed
    SuccessRate float64   `json:"success_rate"`
    CreatedAt   time.Time `json:"created_at"`
}
```

#### `OrgSettings` — Extend `Tenant` Entity

The `OrgSettings` fields partially overlap with `Tenant`. Extend to add:

```go
// Add to Tenant or create OrgSettings view struct:
type OrgSettings struct {
    Name               string `json:"name"`
    Slug               string `json:"slug"`
    Domain             string `json:"domain,omitempty"`
    Timezone           string `json:"timezone"`
    MaxAgents          int    `json:"max_agents"`
    MaxMemoriesPerUser int    `json:"max_memories_per_user"`
    Plan               string `json:"plan"`   // "free" | "pro" | "enterprise"
}
```

#### Security: `raw_key` One-Time Return

The `POST /v1/console/sdk/keys` handler in `vnp-admin` must:
1. Generate a cryptographically random key (e.g., `vnp_prod_sk_<32_random_hex_bytes>`)
2. Store only `SHA-256(raw_key)` in the database
3. Return `raw_key` in the response (this is the **only time** it is ever returned)
4. All subsequent list operations return only the masked `prefix` (first 20 chars)

```go
type CreateKeyResponse struct {
    Key    APIKey `json:"key"`
    RawKey string `json:"raw_key"`   // Only returned once — display immediately in UI
}
```

### Step 4: Wire Handlers in Bootstrap

In `apps/memory/internal/bootstrap/gateway.go`:

```go
orgH := handler.NewOrgHandler(registry, logger)
sdkH := handler.NewSDKHandler(registry, logger)
// Pass to Router(...)
```

---

## Files to Create/Modify

| Action | File |
|--------|------|
| **CREATE** | `gateway/internal/adapter/handler/console_org.go` |
| **CREATE** | `gateway/internal/adapter/handler/console_sdk.go` |
| **MODIFY** | `gateway/internal/adapter/handler/router.go` — add 11 routes |
| **MODIFY** | `apps/memory/internal/bootstrap/gateway.go` — wire handlers |
| **MODIFY** | `services/vnp-platform/internal/domain/admin/entity.go` — add `Webhook`, `OrgSettings` |
| **CREATE** | `services/vnp-platform/internal/usecase/org_usecase.go` |
| **CREATE** | `services/vnp-platform/internal/usecase/sdk_usecase.go` |
| **CREATE** | Migration: add `webhooks` table |

---

## Database Migration (New Table)

```sql
-- deployment/dev/migrations/XXXX_add_webhooks.sql
CREATE TABLE webhooks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    url         TEXT NOT NULL,
    events      TEXT[] NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active',
    secret_hash TEXT,           -- SHA-256 of signing secret
    success_rate FLOAT DEFAULT 1.0,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_webhooks_tenant_id ON webhooks(tenant_id);
```

---

## Acceptance Criteria

- [ ] `GET /v1/console/org/settings` returns org settings scoped to current tenant
- [ ] `PUT /v1/console/org/settings` updates settings, returns updated object
- [ ] `GET /v1/console/org/members` returns member list for current tenant
- [ ] `GET /v1/console/org/roles` returns role definitions
- [ ] `GET /v1/console/sdk/keys` lists keys WITHOUT `raw_key` or `key_hash`
- [ ] `POST /v1/console/sdk/keys` returns `CreateKeyResponse` with `raw_key` once
- [ ] `DELETE /v1/console/sdk/keys/{id}` immediately revokes key
- [ ] `GET /v1/console/sdk/rate-limits` returns tiered rate limit configs
- [ ] `GET /v1/console/sdk/webhooks` lists webhooks for current tenant
- [ ] `POST /v1/console/sdk/webhooks` creates webhook with optional secret
- [ ] `DELETE /v1/console/sdk/webhooks/{id}` deletes webhook
- [ ] All endpoints require `admin` role via `requireAdmin`
- [ ] All data is scoped to `X-Tenant-ID` from JWT
