# CR-002: Org Settings & SDK Management API

**CR ID**: CR-002-org-sdk-api  
**Status**: Open  
**Priority**: 🟡 High  
**Target Components**: `vnp-gateway` (router), `vnp-admin` (backend service)  
**Frontend Source**: `ui/src/services/org.service.ts`, `ui/src/types/org.ts`  
**Created**: 2026-06-18

---

## Problem

The frontend has dedicated pages for:
- **Organization Settings** (`/console/settings`) — lets admins configure org name, limits, timezone, plan
- **SDK / API Key Management** (`/console/sdk`) — lets admins create/revoke API keys and configure webhooks

None of these endpoints exist in the gateway router. The frontend calls them and receives `404 Not Found`.

---

## Required Endpoints

### Org Settings — `/v1/console/org/*`

#### GET `/v1/console/org/settings`

**Auth**: `admin` role required

**Response (200):**
```json
{
  "name":                  "string",
  "slug":                  "string",
  "domain":                "string | null",
  "timezone":              "string (IANA tz, e.g. Asia/Ho_Chi_Minh)",
  "max_agents":            100,
  "max_memories_per_user": 10000,
  "plan":                  "free | pro | enterprise"
}
```

#### PUT `/v1/console/org/settings`

**Auth**: `admin` role required

**Request Body**: `Partial<OrgSettings>` — partial update supported

**Response (200):** Updated `OrgSettings` object

#### GET `/v1/console/org/members`

**Auth**: `admin` role required

**Response (200):**
```json
[
  {
    "id":        "string",
    "name":      "string",
    "email":     "string",
    "role":      "string",
    "status":    "active | inactive",
    "joined_at": "ISO 8601"
  }
]
```

#### GET `/v1/console/org/roles`

**Auth**: `admin` role required

**Response (200):**
```json
[
  {
    "id":          "string",
    "name":        "string",
    "permissions": ["string"]
  }
]
```

---

### SDK / API Keys — `/v1/console/sdk/*`

#### GET `/v1/console/sdk/keys`

**Auth**: `admin` role required

**Response (200):**
```json
[
  {
    "id":          "string",
    "name":        "string",
    "prefix":      "vnp_prod_sk_3f9a...",
    "scopes":      ["string"],
    "created_at":  "ISO 8601",
    "last_used":   "ISO 8601 | null",
    "expires_at":  "ISO 8601 | null",
    "status":      "active | revoked | expired"
  }
]
```

> **Important**: `raw_key` MUST NOT be included in list response — only masked `prefix`.

#### POST `/v1/console/sdk/keys`

**Auth**: `admin` role required

**Request Body:**
```json
{
  "name":             "string",
  "permissions":      ["string"],
  "expires_in_days":  30
}
```

**Response (201):**
```json
{
  "key":     { /* APIKey object */ },
  "raw_key": "vnp_prod_sk_xxxxxxxx..."
}
```

> **Critical**: `raw_key` is returned **only once**. The UI must display it immediately with copy-to-clipboard. The backend must never store/return it again.

#### DELETE `/v1/console/sdk/keys/{id}`

**Auth**: `admin` role required

**Response**: `204 No Content`

#### GET `/v1/console/sdk/rate-limits`

**Auth**: `admin` role required

**Response (200):**
```json
[
  {
    "scope":  "string",
    "rps":    10,
    "rpm":    600,
    "burst":  50,
    "tier":   "enterprise | standard | restricted"
  }
]
```

#### GET `/v1/console/sdk/webhooks`

**Auth**: `admin` role required

**Response (200):**
```json
[
  {
    "id":           "string",
    "url":          "https://...",
    "events":       ["memory.stored", "session.ended"],
    "status":       "active | paused | failed",
    "success_rate": 0.98,
    "created_at":   "ISO 8601"
  }
]
```

#### POST `/v1/console/sdk/webhooks`

**Auth**: `admin` role required

**Request Body:**
```json
{
  "url":    "https://...",
  "events": ["memory.stored"],
  "secret": "optional signing secret"
}
```

**Response (201):** Created `Webhook` object

#### DELETE `/v1/console/sdk/webhooks/{id}`

**Auth**: `admin` role required

**Response**: `204 No Content`

---

## Implementation Notes

1. **Gateway Router** — Add to `router.go`:
   ```go
   // Org settings
   mux.HandleFunc("GET /v1/console/org/settings",    org.GetSettings)
   mux.HandleFunc("PUT /v1/console/org/settings",    org.UpdateSettings)
   mux.HandleFunc("GET /v1/console/org/members",     org.ListMembers)
   mux.HandleFunc("GET /v1/console/org/roles",       org.ListRoles)

   // SDK management
   mux.HandleFunc("GET /v1/console/sdk/keys",              sdk.ListKeys)
   mux.HandleFunc("POST /v1/console/sdk/keys",             sdk.CreateKey)
   mux.HandleFunc("DELETE /v1/console/sdk/keys/{id}",      sdk.DeleteKey)
   mux.HandleFunc("GET /v1/console/sdk/rate-limits",       sdk.GetRateLimits)
   mux.HandleFunc("GET /v1/console/sdk/webhooks",          sdk.ListWebhooks)
   mux.HandleFunc("POST /v1/console/sdk/webhooks",         sdk.CreateWebhook)
   mux.HandleFunc("DELETE /v1/console/sdk/webhooks/{id}",  sdk.DeleteWebhook)
   ```

2. **New Handlers**: Create `console_org.go` and `console_sdk.go` in `gateway/internal/adapter/handler/` — forward to `vnp-admin`.

3. **Auth**: All these routes use `requireAdmin` (not `requireSuperAdmin`).

4. **Security**: The `raw_key` must be generated and passed through only at creation time. The backend storage layer must store only the hashed key.

---

## Acceptance Criteria

- [ ] `GET /v1/console/org/settings` returns org settings for current tenant
- [ ] `PUT /v1/console/org/settings` updates settings (partial update supported)
- [ ] `GET /v1/console/org/members` returns paginated member list
- [ ] `GET /v1/console/org/roles` returns available role definitions
- [ ] `GET /v1/console/sdk/keys` lists keys without `raw_key`
- [ ] `POST /v1/console/sdk/keys` returns `raw_key` exactly once
- [ ] `DELETE /v1/console/sdk/keys/{id}` revokes key immediately
- [ ] `GET /v1/console/sdk/rate-limits` returns tiered rate limit configs
- [ ] `GET /v1/console/sdk/webhooks` lists webhooks
- [ ] `POST /v1/console/sdk/webhooks` creates webhook with optional signing secret
- [ ] `DELETE /v1/console/sdk/webhooks/{id}` deletes webhook
