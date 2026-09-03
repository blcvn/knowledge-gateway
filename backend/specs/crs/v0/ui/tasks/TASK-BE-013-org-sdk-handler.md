# TASK-BE-013 — Console Org & SDK Handlers + `api_keys` + `webhooks` migrations

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-013 |
| **Layer** | Backend — Go / PostgreSQL |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-006 CR-011](../solutions/SOL-006-Adaptive-to-Org-Solutions.md) + [SOL-007 §11](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟡 P2 |
| **Depends On** | TASK-BE-001 |
| **Estimated** | 3h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `vnp-platform/migrations/0008_create_api_keys.sql` |
| CREATE | `vnp-platform/migrations/0009_create_webhooks.sql` |
| CREATE | `gateway/internal/adapter/handler/console_org_handler.go` |
| CREATE | `gateway/internal/adapter/handler/console_sdk_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### Migration: `0008_create_api_keys.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS api_keys (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    name        TEXT        NOT NULL,
    key_hash    TEXT        UNIQUE NOT NULL,  -- SHA-256 of raw key
    prefix      TEXT        NOT NULL,         -- First 8 chars for display
    permissions TEXT[]      DEFAULT '{}',
    expires_at  TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked     BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash   ON api_keys(key_hash);

-- +migrate Down
DROP TABLE IF EXISTS api_keys;
```

### Migration: `0009_create_webhooks.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS webhooks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    url          TEXT        NOT NULL,
    events       TEXT[]      NOT NULL DEFAULT '{}',
    secret_hash  TEXT,                         -- HMAC signing key hash (optional)
    status       TEXT        NOT NULL DEFAULT 'active',
    success_rate FLOAT       NOT NULL DEFAULT 100.0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_tenant ON webhooks(tenant_id);

-- +migrate Down
DROP TABLE IF EXISTS webhooks;
```

### Handler: `console_org_handler.go`

```go
package handler

type ConsoleOrgHandler struct {
    adminSvc VNPAdminClient  // gRPC → vnp-admin
    db       *sql.DB         // tenants table
}

// GET /v1/console/org/settings
// → Lấy tenant settings của tenant hiện tại (from JWT)
func (h *ConsoleOrgHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    tenant, _ := h.adminSvc.GetTenant(r.Context(), tenantID)
    httputil.JSON(w, 200, map[string]any{
        "name":                    tenant.Name,
        "slug":                    tenant.Slug,
        "timezone":                tenant.Timezone,
        "max_agents":              tenant.Limits.MaxAgents,
        "max_memories_per_user":   tenant.Limits.MaxMemoriesPerUser,
    })
}

// PUT /v1/console/org/settings
func (h *ConsoleOrgHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
    var req map[string]any; json.NewDecoder(r.Body).Decode(&req)
    updated, _ := h.adminSvc.UpdateTenant(r.Context(), authctx.TenantID(r.Context()), req)
    httputil.JSON(w, 200, updated)
}

// GET /v1/console/org/members
func (h *ConsoleOrgHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, name, email, role, avatar_url, created_at FROM users WHERE tenant_id = $1 AND is_active = true`,
        tenantID)
    // scan and return
}

// GET /v1/console/org/roles
func (h *ConsoleOrgHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
    httputil.JSON(w, 200, []map[string]any{
        {"id": "owner",  "name": "Owner",  "permissions": []string{"*"}},
        {"id": "admin",  "name": "Admin",  "permissions": []string{"console:*", "api:*"}},
        {"id": "editor", "name": "Editor", "permissions": []string{"console:read", "console:write"}},
        {"id": "viewer", "name": "Viewer", "permissions": []string{"console:read"}},
    })
}
```

### Handler: `console_sdk_handler.go`

```go
package handler

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

type ConsoleSDKHandler struct {
    db *sql.DB
}

// GET /v1/console/sdk/keys
func (h *ConsoleSDKHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, name, prefix, permissions, expires_at, last_used_at, created_at
         FROM api_keys WHERE tenant_id = $1 AND revoked = false ORDER BY created_at DESC`,
        tenantID)
    // scan and return — NEVER return key_hash
}

// POST /v1/console/sdk/keys
func (h *ConsoleSDKHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name          string   `json:"name"`
        Permissions   []string `json:"permissions"`
        ExpiresInDays int      `json:"expires_in_days"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    tenantID := authctx.TenantID(r.Context())

    // Generate random key: vnp_prod_<random32chars>
    raw := make([]byte, 24)
    rand.Read(raw)
    rawKey   := fmt.Sprintf("vnp_%s", hex.EncodeToString(raw))
    prefix   := rawKey[:8]
    hash     := sha256.Sum256([]byte(rawKey))
    keyHash  := hex.EncodeToString(hash[:])

    var expiresAt *time.Time
    if req.ExpiresInDays > 0 {
        t := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
        expiresAt = &t
    }

    var id string
    h.db.QueryRowContext(r.Context(),
        `INSERT INTO api_keys (tenant_id, name, key_hash, prefix, permissions, expires_at)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
        tenantID, req.Name, keyHash, prefix, req.Permissions, expiresAt).Scan(&id)

    // raw_key chỉ trả về 1 lần — sau đó không thể recover
    httputil.JSON(w, 201, map[string]any{
        "key": map[string]any{"id": id, "name": req.Name, "prefix": prefix},
        "raw_key": rawKey,  // Show once
    })
}

// DELETE /v1/console/sdk/keys/{id}
func (h *ConsoleSDKHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
    h.db.ExecContext(r.Context(),
        `UPDATE api_keys SET revoked = true WHERE id = $1 AND tenant_id = $2`,
        r.PathValue("id"), authctx.TenantID(r.Context()))
    w.WriteHeader(204)
}

// GET /v1/console/sdk/rate-limits
func (h *ConsoleSDKHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
    // Return tier config from admin service
    httputil.JSON(w, 200, []map[string]any{{
        "scope": "global", "rps": 100, "rpm": 6000, "burst": 200, "tier_name": "pro",
    }})
}

// GET /v1/console/sdk/webhooks
func (h *ConsoleSDKHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, url, events, status, success_rate, created_at
         FROM webhooks WHERE tenant_id = $1`, authctx.TenantID(r.Context()))
    // scan and return
}

// POST /v1/console/sdk/webhooks
func (h *ConsoleSDKHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
    var req struct {
        URL    string   `json:"url"`
        Events []string `json:"events"`
        Secret string   `json:"secret"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    var secretHash *string
    if req.Secret != "" {
        h := sha256.Sum256([]byte(req.Secret))
        s := hex.EncodeToString(h[:]); secretHash = &s
    }
    var id string
    h.db.QueryRowContext(r.Context(),
        `INSERT INTO webhooks (tenant_id, url, events, secret_hash) VALUES ($1, $2, $3, $4) RETURNING id`,
        authctx.TenantID(r.Context()), req.URL, req.Events, secretHash).Scan(&id)
    httputil.JSON(w, 201, map[string]any{"id": id, "url": req.URL, "events": req.Events, "status": "active"})
}

// DELETE /v1/console/sdk/webhooks/{id}
func (h *ConsoleSDKHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
    h.db.ExecContext(r.Context(),
        `DELETE FROM webhooks WHERE id = $1 AND tenant_id = $2`,
        r.PathValue("id"), authctx.TenantID(r.Context()))
    w.WriteHeader(204)
}
```

### Routes

```go
// Org
mux.HandleFunc("GET /v1/console/org/settings",   authMiddleware(org.GetSettings))
mux.HandleFunc("PUT /v1/console/org/settings",   authMiddleware(org.UpdateSettings))
mux.HandleFunc("GET /v1/console/org/members",    authMiddleware(org.GetMembers))
mux.HandleFunc("GET /v1/console/org/roles",      authMiddleware(org.GetRoles))

// SDK
mux.HandleFunc("GET /v1/console/sdk/keys",         authMiddleware(sdk.ListKeys))
mux.HandleFunc("POST /v1/console/sdk/keys",        authMiddleware(sdk.CreateKey))
mux.HandleFunc("DELETE /v1/console/sdk/keys/{id}", authMiddleware(sdk.RevokeKey))
mux.HandleFunc("GET /v1/console/sdk/rate-limits",  authMiddleware(sdk.GetRateLimits))
mux.HandleFunc("GET /v1/console/sdk/webhooks",         authMiddleware(sdk.ListWebhooks))
mux.HandleFunc("POST /v1/console/sdk/webhooks",        authMiddleware(sdk.CreateWebhook))
mux.HandleFunc("DELETE /v1/console/sdk/webhooks/{id}", authMiddleware(sdk.DeleteWebhook))
```
