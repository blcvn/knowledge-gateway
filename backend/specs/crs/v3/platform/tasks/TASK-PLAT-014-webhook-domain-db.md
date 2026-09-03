# TASK-PLAT-014 — Webhook Domain Model & DB Migration

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-014 |
| **Wave** | 4 (Events) |
| **Solution** | [SOL-PLAT-005](../solutions/SOL-PLAT-005-Webhook-Delivery-System.md) §2.1, §2.4 |
| **Component** | `services/vnp-platform/internal/domain/`, `deployment/dev/migrations/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 2h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** Webhook domain entity + CreateWebhookPayload + WebhookStatus constants in entity.go (TASK-008/v2)
---

## Mục tiêu

Tạo domain types cho Webhook và WebhookDelivery. Tạo SQL migration cho `webhooks` và `webhook_deliveries` tables.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/domain/webhook.go` [NEW]

```go
package domain

type WebhookStatus string

const (
    WebhookStatusActive   WebhookStatus = "active"
    WebhookStatusDegraded WebhookStatus = "degraded" // 3+ consecutive failures
    WebhookStatusDisabled WebhookStatus = "disabled"
)

// SupportedEvents — all valid webhook event types (per CR-PLAT-005 §2)
var SupportedEvents = []string{
    "memory.stored",
    "memory.forgotten",
    "session.completed",
    "rate_limit.exceeded",
    "health.degraded",
    "pipeline.completed",
}

type Webhook struct {
    ID        string
    TenantID  string
    URL       string
    SecretEnc string        // AES-encrypted HMAC secret
    Events    []string      // subset of SupportedEvents
    Status    WebhookStatus
    FailCount int           // consecutive failures, reset on success
    CreatedAt time.Time
    UpdatedAt time.Time
}

type WebhookDelivery struct {
    ID         string
    WebhookID  string
    TenantID   string
    Event      string
    Payload    string  // JSON payload sent to URL
    StatusCode int     // HTTP response code (0 = network error)
    Success    bool
    Attempt    int     // 1, 2, or 3
    ErrorMsg   string
    CreatedAt  time.Time
}

// IsValidEvent checks if an event type is supported
func IsValidEvent(event string) bool {
    for _, e := range SupportedEvents {
        if e == event {
            return true
        }
    }
    return false
}

// WebhookPayload is the JSON body sent to webhook URLs
type WebhookPayload struct {
    Event     string          `json:"event"`
    Data      json.RawMessage `json:"data"`
    Timestamp string          `json:"timestamp"` // RFC3339
}
```

### 2. Tạo `services/vnp-platform/internal/port/webhook_repository.go` [NEW]

```go
package port

type WebhookRepository interface {
    Insert(ctx context.Context, wh *domain.Webhook) error
    Get(ctx context.Context, id string) (*domain.Webhook, error)
    ListByTenant(ctx context.Context, tenantID string) ([]*domain.Webhook, error)
    FindByTenantAndEvent(ctx context.Context, tenantID, event string) ([]*domain.Webhook, error)
    Update(ctx context.Context, wh *domain.Webhook) error
    Delete(ctx context.Context, id string) error
    IncrementFailCount(ctx context.Context, id string) error
    ResetFailCount(ctx context.Context, id string) error
    UpdateStatus(ctx context.Context, id string, status domain.WebhookStatus) error
    RecordDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error
    GetDeliveries(ctx context.Context, webhookID string, limit int) ([]*domain.WebhookDelivery, error)
}
```

### 3. Tạo migration `deployment/dev/migrations/XXX_webhooks.up.sql` [NEW]

```sql
-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    url         TEXT NOT NULL,
    secret_enc  TEXT NOT NULL,      -- AES-GCM encrypted HMAC secret
    events      TEXT[] NOT NULL,    -- PostgreSQL text array
    status      TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'degraded', 'disabled')),
    fail_count  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_tenant  ON webhooks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_events  ON webhooks USING gin(events);
CREATE INDEX IF NOT EXISTS idx_webhooks_status  ON webhooks(tenant_id, status);

-- Webhook deliveries (keep last 50 per webhook via app-level trim)
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id  UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    tenant_id   TEXT NOT NULL,
    event       TEXT NOT NULL,
    payload     JSONB,
    status_code INT,
    success     BOOLEAN NOT NULL DEFAULT false,
    attempt     INT NOT NULL DEFAULT 1,
    error_msg   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_deliveries_webhook ON webhook_deliveries(webhook_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deliveries_tenant  ON webhook_deliveries(tenant_id, created_at DESC);
```

### 4. Tạo down migration `deployment/dev/migrations/XXX_webhooks.down.sql` [NEW]

```sql
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;
```

---

## Acceptance Criteria

- [ ] `Webhook` domain type với đầy đủ fields: URL, SecretEnc, Events array, Status, FailCount
- [ ] `SupportedEvents` contains all 6 event types per CR spec
- [ ] `IsValidEvent()` returns true only for supported events
- [ ] Migration creates `webhooks` table với correct constraints
- [ ] Migration creates `webhook_deliveries` table với cascade delete
- [ ] GIN index on `events` array for fast lookup by event type
- [ ] Down migration drops both tables cleanly
- [ ] `go build ./services/vnp-platform/...` passes

## Files

```
services/vnp-platform/internal/domain/webhook.go              [NEW]
services/vnp-platform/internal/port/webhook_repository.go     [NEW]
deployment/dev/migrations/XXX_webhooks.up.sql                  [NEW]
deployment/dev/migrations/XXX_webhooks.down.sql                [NEW]
```
