# SOL-PLAT-005 — Solution: Webhook Delivery System

| Field | Value |
|---|---|
| **Solution ID** | SOL-PLAT-005 |
| **CR** | [CR-PLAT-005](../../../../docs/crs/v3/platform/CR-PLAT-005-Webhook-Delivery-System.md) |
| **TDD ref** | [08-platform-services.md §VNP-Admin](../../../tdd/architecture/08-platform-services.md) · [backend-api-specs.md §Known-Gaps](../../../tdd/backend-api-specs.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** Webhook entity + handler; delivery service (retry/signature) not implemented
---

## 1. Phân tích kiến trúc

Theo TDD `08-platform-services.md §VNP-Admin`, `vnp-admin` quản lý tenant config và API keys. Webhook management cần:
- Webhook subscription storage (trong `vnp-platform` service)
- NATS subscriber để catch 6 event types
- HTTP delivery với HMAC-SHA256 signature
- Exponential backoff retry (3 retries)
- Delivery history (last 50 per webhook)

Theo `backend-api-specs.md §Known-Gaps`, các `GET/POST /v1/console/sdk/webhooks` endpoints **chưa implement**.

---

## 2. Giải pháp

### 2.1 `services/vnp-platform/internal/domain/webhook.go` [NEW]

```go
package domain

type WebhookStatus string

const (
    WebhookStatusActive   WebhookStatus = "active"
    WebhookStatusDegraded WebhookStatus = "degraded"   // 3+ consecutive failures
    WebhookStatusDisabled WebhookStatus = "disabled"
)

type Webhook struct {
    ID         string
    TenantID   string
    URL        string
    Secret     string        // HMAC secret (stored encrypted)
    Events     []string      // ["memory.stored", "session.completed", ...]
    Status     WebhookStatus
    FailCount  int           // consecutive failure count
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type WebhookDelivery struct {
    ID         string
    WebhookID  string
    TenantID   string
    Event      string        // event type
    Payload    string        // JSON payload sent
    StatusCode int           // HTTP response code (0 if unreachable)
    Success    bool
    Attempt    int
    Error      string
    CreatedAt  time.Time
}

// SupportedEvents returns all valid event types
var SupportedEvents = []string{
    "memory.stored",
    "memory.forgotten",
    "session.completed",
    "rate_limit.exceeded",
    "health.degraded",
    "pipeline.completed",
}
```

### 2.2 `services/vnp-platform/internal/usecase/webhook_delivery.go` [NEW]

```go
package usecase

type WebhookDeliveryService struct {
    repo        port.WebhookRepository
    httpClient  *http.Client
    nats        port.NATSSubscriber
    logger      *slog.Logger
}

// Start subscribes to all NATS subjects and begins webhook fan-out
func (s *WebhookDeliveryService) Start(ctx context.Context) error {
    subjects := []string{
        "memory.blob.inserted",    // → memory.stored event
        "memory.forgotten",        // → memory.forgotten event
        "observe.session.ended",   // → session.completed event
        "rate_limit.exceeded",     // → rate_limit.exceeded event
        "health.status.changed",   // → health.degraded event
        "pipeline.completed",      // → pipeline.completed event
    }

    for _, subject := range subjects {
        subject := subject
        s.nats.Subscribe(subject, func(msg *nats.Msg) {
            s.dispatch(ctx, subject, msg.Data)
        })
    }
    return nil
}

func (s *WebhookDeliveryService) dispatch(ctx context.Context, natsSubject string, payload []byte) {
    tenantID := extractTenantID(payload)
    eventType := natsSubjectToWebhookEvent(natsSubject)

    // Find all active webhooks for this tenant subscribed to this event
    webhooks, err := s.repo.FindByTenantAndEvent(ctx, tenantID, eventType)
    if err != nil || len(webhooks) == 0 {
        return
    }

    webhookPayload := WebhookPayload{
        Event:     eventType,
        Data:      json.RawMessage(payload),
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
    payloadJSON, _ := json.Marshal(webhookPayload)

    for _, wh := range webhooks {
        go s.deliver(ctx, wh, payloadJSON)
    }
}

// deliver sends to a single webhook URL with exponential backoff retry
func (s *WebhookDeliveryService) deliver(ctx context.Context, wh *domain.Webhook, payload []byte) {
    deliveryID := uuid.NewString()

    var lastErr error
    var statusCode int

    // Exponential backoff: attempt 1 → 5s → 25s (3 retries max per CR spec)
    delays := []time.Duration{0, 5 * time.Second, 25 * time.Second}

    for attempt, delay := range delays {
        if delay > 0 {
            time.Sleep(delay)
        }

        statusCode, lastErr = s.sendHTTP(ctx, wh, payload, deliveryID)
        success := lastErr == nil && statusCode >= 200 && statusCode < 300

        // Record delivery attempt
        s.repo.RecordDelivery(ctx, &domain.WebhookDelivery{
            ID:         uuid.NewString(),
            WebhookID:  wh.ID,
            TenantID:   wh.TenantID,
            Event:      extractEventType(payload),
            Payload:    string(payload),
            StatusCode: statusCode,
            Success:    success,
            Attempt:    attempt + 1,
            Error:      errorStr(lastErr),
            CreatedAt:  time.Now().UTC(),
        })

        if success {
            // Reset failure count on success
            s.repo.ResetFailCount(ctx, wh.ID)
            return
        }
    }

    // All 3 attempts failed
    s.repo.IncrementFailCount(ctx, wh.ID)
    wh.FailCount++
    if wh.FailCount >= 3 {
        // Mark webhook as degraded
        s.repo.UpdateStatus(ctx, wh.ID, domain.WebhookStatusDegraded)
        s.logger.Warn("webhook disabled after 3 consecutive failures",
            "webhook_id", wh.ID, "tenant_id", wh.TenantID)
    }
}

func (s *WebhookDeliveryService) sendHTTP(ctx context.Context, wh *domain.Webhook, payload []byte, deliveryID string) (int, error) {
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(payload))

    // HMAC-SHA256 signature: HMAC(secret, body)
    mac := hmac.New(sha256.New, []byte(wh.Secret))
    mac.Write(payload)
    sig := hex.EncodeToString(mac.Sum(nil))

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-VNP-Signature", "sha256="+sig)
    req.Header.Set("X-VNP-Event", extractEventType(payload))
    req.Header.Set("X-VNP-Delivery", deliveryID)
    req.Header.Set("User-Agent", "VNP-Memory-Webhook/1.0")

    resp, err := s.httpClient.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    return resp.StatusCode, nil
}
```

### 2.3 Webhook Management Handler — `gateway/adapter/handler/webhooks.go` [NEW]

```go
package handler

// GET /v1/console/sdk/webhooks
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    webhooks, err := h.svc.List(r.Context(), auth.TenantID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, webhooks)
}

// POST /v1/console/sdk/webhooks
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    var req struct {
        URL    string   `json:"url"`
        Events []string `json:"events"`
        Secret string   `json:"secret"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    // Validate events
    for _, ev := range req.Events {
        if !isValidEvent(ev) {
            writeError(w, http.StatusBadRequest, "invalid_event", ev)
            return
        }
    }

    webhook, err := h.svc.Create(r.Context(), auth.TenantID, req.URL, req.Events, req.Secret)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, webhook)
}

// PUT /v1/console/sdk/webhooks/{id}
// DELETE /v1/console/sdk/webhooks/{id}

// GET /v1/console/sdk/webhooks/{id}/deliveries
func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
    webhookID := chi.URLParam(r, "id")
    // Return last 50 delivery records
    deliveries, err := h.svc.GetDeliveries(r.Context(), webhookID, 50)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, deliveries)
}

// POST /v1/console/sdk/webhooks/{id}/test
func (h *WebhookHandler) Test(w http.ResponseWriter, r *http.Request) {
    webhookID := chi.URLParam(r, "id")
    // Send a sample "memory.stored" event immediately (no NATS)
    delivery, err := h.svc.SendTestEvent(r.Context(), webhookID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, delivery)
}
```

### 2.4 DB Migration — `deployment/dev/migrations/xxx_webhooks.up.sql` [NEW]

```sql
CREATE TABLE webhooks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    url         TEXT NOT NULL,
    secret_enc  TEXT NOT NULL,      -- AES-encrypted HMAC secret
    events      TEXT[] NOT NULL,    -- array of event types
    status      TEXT NOT NULL DEFAULT 'active',
    fail_count  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_tenant ON webhooks(tenant_id);
CREATE INDEX idx_webhooks_events ON webhooks USING gin(events);

CREATE TABLE webhook_deliveries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id  UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    tenant_id   TEXT NOT NULL,
    event       TEXT NOT NULL,
    payload     JSONB,
    status_code INT,
    success     BOOLEAN NOT NULL,
    attempt     INT NOT NULL DEFAULT 1,
    error_msg   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deliveries_webhook ON webhook_deliveries(webhook_id, created_at DESC);
```

### 2.5 NATS Subject → Webhook Event Mapping

```go
// services/vnp-platform/internal/usecase/event_mapping.go [NEW]

var natsToWebhookEvent = map[string]string{
    "memory.blob.inserted":   "memory.stored",
    "memory.forgotten":       "memory.forgotten",
    "observe.session.ended":  "session.completed",
    "rate_limit.exceeded":    "rate_limit.exceeded",
    "health.status.changed":  "health.degraded",
    "pipeline.completed":     "pipeline.completed",
}

func natsSubjectToWebhookEvent(subject string) string {
    if ev, ok := natsToWebhookEvent[subject]; ok {
        return ev
    }
    return subject
}
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `services/vnp-platform/internal/domain/webhook.go` | NEW | Webhook + WebhookDelivery domain types |
| `services/vnp-platform/internal/usecase/webhook_delivery.go` | NEW | Delivery service: NATS subscribe + HTTP dispatch + retry |
| `services/vnp-platform/internal/usecase/event_mapping.go` | NEW | NATS subject → webhook event mapping |
| `services/vnp-platform/internal/port/webhook.go` | NEW | WebhookRepository interface |
| `gateway/adapter/handler/webhooks.go` | NEW | CRUD + deliveries + test handlers |
| `gateway/adapter/handler/router.go` | MODIFY | Register `/v1/console/sdk/webhooks/*` routes |
| `deployment/dev/migrations/xxx_webhooks.up.sql` | NEW | webhooks + webhook_deliveries tables |

---

## 4. Acceptance Criteria

- [ ] All 6 event types delivered correctly (memory.stored, memory.forgotten, session.completed, rate_limit.exceeded, health.degraded, pipeline.completed)
- [ ] HMAC-SHA256 signature: `X-VNP-Signature: sha256={hex}` on every delivery
- [ ] Exponential backoff: 3 retries max (immediate → 5s → 25s)
- [ ] Delivery history: last 50 deliveries per webhook (accessible via `/deliveries` endpoint)
- [ ] Test endpoint sends sample event immediately without waiting for NATS trigger
- [ ] Webhook marked as `degraded` after 3 consecutive complete failures
- [ ] `X-VNP-Signature` verifiable by receiver using known secret
- [ ] Webhook secret stored encrypted at rest (AES)
- [ ] HTTP client timeout: 10 seconds per delivery attempt

---

## 5. Dependencies

- NATS JetStream for event subscription
- `services/vnp-platform` PostgreSQL for webhook storage
- HTTP client with 10s timeout for delivery
- AES key for secret encryption (`WEBHOOK_SECRET_KEY` env var)
- SOL-PLAT-002 rate limit events (`rate_limit.exceeded`) published via NATS
