# TASK-008: Add `Webhook` Entity + `OrgSettings` View to `vnp-platform`

**Solution**: [SOL-002](../solutions/SOL-002-org-sdk-api.md)  
**CR**: CR-002  
**Priority**: 🟡 High  
**Estimate**: 2 hours  
**Status**: ✅ Implemented

---

## Context

The `vnp-admin` service needs two new domain entities:
1. `OrgSettings` — a view of the `Tenant` struct formatted for the console Org Settings page
2. `Webhook` — a new entity for SDK webhook management (no table currently exists)

File to modify: `services/vnp-platform/internal/domain/admin/entity.go`

---

## Exact Task

### Step 1: Add `OrgSettings` view struct

In `services/vnp-platform/internal/domain/admin/entity.go`, add after the existing `Tenant` struct:

```go
// OrgSettings is the tenant configuration view exposed to admin users.
// It maps from Tenant fields and provides the org settings page data.
type OrgSettings struct {
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Domain             string `json:"domain,omitempty"`
	Timezone           string `json:"timezone"`
	MaxAgents          int    `json:"max_agents"`
	MaxMemoriesPerUser int    `json:"max_memories_per_user"`
	Plan               string `json:"plan"` // "free" | "pro" | "enterprise"
}

// OrgMember represents a user within the organization for the members list.
type OrgMember struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Status   string    `json:"status"` // "active" | "inactive"
	JoinedAt time.Time `json:"joined_at"`
}

// OrgRole defines a role and its permissions.
type OrgRole struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}
```

### Step 2: Add `Webhook` entity

In the same file, add after the existing `APIKey` struct:

```go
// Webhook represents a registered webhook endpoint for SDK events.
type Webhook struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	URL         string     `json:"url"`
	Events      []string   `json:"events"`
	Status      string     `json:"status"`       // "active" | "paused" | "failed"
	SecretHash  string     `json:"-"`             // SHA-256 of signing secret — never exposed in JSON
	SuccessRate float64    `json:"success_rate"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreateWebhookPayload is the request body for creating a webhook.
type CreateWebhookPayload struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret,omitempty"` // Optional signing secret — stored as hash
}

// WebhookStatus values
const (
	WebhookActive string = "active"
	WebhookPaused string = "paused"
	WebhookFailed string = "failed"
)
```

### Step 3: Update `APIKey` JSON struct if needed

The frontend expects `APIKey` to have a `prefix` field (visible masked key prefix) and a `status` field. Check the current `APIKey` struct and add if missing:

```go
// APIKey — verify these fields exist in the current struct:
// prefix    string     `json:"prefix"`    // e.g. "vnp_prod_sk_3f9a..."
// status    string     `json:"status"`    // "active" | "revoked" | "expired"
// last_used *time.Time `json:"last_used,omitempty"`
// scopes    []string   `json:"scopes"`
```

If missing, add them to the `APIKey` struct.

### Step 4: Add `CreateAPIKeyResponse` struct

```go
// CreateAPIKeyResponse is returned from POST /v1/console/sdk/keys.
// raw_key is only included in the creation response — never returned again.
type CreateAPIKeyResponse struct {
	Key    APIKey `json:"key"`
	RawKey string `json:"raw_key"` // ONLY returned once at creation
}

// CreateAPIKeyPayload is the request body for creating an API key.
type CreateAPIKeyPayload struct {
	Name          string   `json:"name"`
	Permissions   []string `json:"permissions"`
	ExpiresInDays int      `json:"expires_in_days,omitempty"`
}
```

### Step 5: Add required imports

Ensure `time` and `uuid` are imported:
```go
import (
    "time"
    "github.com/google/uuid"
)
```

---

## Files to Modify

| File | Change |
|------|--------|
| `services/vnp-platform/internal/domain/admin/entity.go` | Add `OrgSettings`, `OrgMember`, `OrgRole`, `Webhook`, `CreateWebhookPayload`, `CreateAPIKeyResponse`, `CreateAPIKeyPayload` |

---

## Acceptance Criteria

- [ ] `OrgSettings`, `OrgMember`, `OrgRole` structs added with correct JSON tags
- [ ] `Webhook` struct added with `SecretHash` excluded from JSON (`json:"-"`)
- [ ] `CreateAPIKeyResponse` struct added with `raw_key` field
- [ ] `APIKey` struct has `prefix`, `status`, `last_used`, `scopes` fields
- [ ] `go build ./services/vnp-platform/...` passes

---

**Audit Note:** OrgSettings, OrgMember, OrgRole, Webhook, CreateWebhookPayload entities added to entity.go
