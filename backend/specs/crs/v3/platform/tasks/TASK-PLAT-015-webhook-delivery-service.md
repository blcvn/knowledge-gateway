# TASK-PLAT-015 — Webhook Delivery Service (NATS Subscribe + HTTP Dispatch)

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-015 |
| **Wave** | 4 (Events) |
| **Solution** | [SOL-PLAT-005](../solutions/SOL-PLAT-005-Webhook-Delivery-System.md) §2.2, §2.5 |
| **Component** | `services/vnp-platform/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-014 |
| **Estimated** | 5h |

---

## Mục tiêu

Implement background service: subscribe NATS events → fan-out → HTTP POST với HMAC-SHA256 signature → exponential backoff retry (3 attempts) → record delivery history → degrade webhook after 3 consecutive failures.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/usecase/event_mapping.go` [NEW]

```go
package usecase

// natsToWebhookEvent maps NATS subject → webhook event type
var natsToWebhookEvent = map[string]string{
    "memory.blob.inserted":  "memory.stored",
    "memory.forgotten":      "memory.forgotten",
    "observe.session.ended": "session.completed",
    "rate_limit.exceeded":   "rate_limit.exceeded",
    "health.status.changed": "health.degraded",
    "pipeline.completed":    "pipeline.completed",
}

func natsSubjectToWebhookEvent(subject string) string {
    if ev, ok := natsToWebhookEvent[subject]; ok {
        return ev
    }
    return subject
}

// extractTenantID extracts tenant_id from NATS message JSON payload
func extractTenantID(payload []byte) string {
    var msg struct {
        TenantID string `json:"tenant_id"`
    }
    json.Unmarshal(payload, &msg)
    return msg.TenantID
}
```

### 2. Tạo `services/vnp-platform/internal/usecase/webhook_delivery.go` [NEW]

```go
package usecase

type WebhookDeliveryService struct {
    repo       port.WebhookRepository
    nats       port.NATSSubscriber
    httpClient *http.Client
    logger     *slog.Logger
}

func NewWebhookDeliveryService(
    repo port.WebhookRepository,
    nats port.NATSSubscriber,
    logger *slog.Logger,
) *WebhookDeliveryService {
    return &WebhookDeliveryService{
        repo:   repo,
        nats:   nats,
        httpClient: &http.Client{Timeout: 10 * time.Second},
        logger: logger,
    }
}

// Start subscribes to all NATS subjects and begins fan-out delivery
func (s *WebhookDeliveryService) Start(ctx context.Context) error {
    subjects := []string{
        "memory.blob.inserted",
        "memory.forgotten",
        "observe.session.ended",
        "rate_limit.exceeded",
        "health.status.changed",
        "pipeline.completed",
    }
    for _, subject := range subjects {
        sub := subject
        if _, err := s.nats.Subscribe(sub, func(msg *nats.Msg) {
            s.dispatch(context.Background(), sub, msg.Data)
        }); err != nil {
            return fmt.Errorf("subscribe %s: %w", sub, err)
        }
    }
    s.logger.Info("webhook delivery service started", "subjects", subjects)
    return nil
}

// dispatch fans out a NATS event to all matching webhooks for the tenant
func (s *WebhookDeliveryService) dispatch(ctx context.Context, natsSubject string, rawPayload []byte) {
    tenantID := extractTenantID(rawPayload)
    eventType := natsSubjectToWebhookEvent(natsSubject)
    if tenantID == "" || eventType == "" {
        return
    }

    webhooks, err := s.repo.FindByTenantAndEvent(ctx, tenantID, eventType)
    if err != nil || len(webhooks) == 0 {
        return
    }

    payload := domain.WebhookPayload{
        Event:     eventType,
        Data:      json.RawMessage(rawPayload),
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
    payloadJSON, _ := json.Marshal(payload)

    for _, wh := range webhooks {
        go s.deliver(ctx, wh, payloadJSON)
    }
}

// deliver sends with exponential backoff: immediate → 5s → 25s (max 3 attempts per CR spec)
func (s *WebhookDeliveryService) deliver(ctx context.Context, wh *domain.Webhook, payload []byte) {
    deliveryID := uuid.NewString()
    delays := []time.Duration{0, 5 * time.Second, 25 * time.Second}

    for attempt, delay := range delays {
        if delay > 0 {
            time.Sleep(delay)
        }

        statusCode, err := s.sendHTTP(ctx, wh, payload, deliveryID)
        success := err == nil && statusCode >= 200 && statusCode < 300

        s.repo.RecordDelivery(ctx, &domain.WebhookDelivery{
            ID:         uuid.NewString(),
            WebhookID:  wh.ID,
            TenantID:   wh.TenantID,
            Event:      extractEventFromPayload(payload),
            Payload:    string(payload),
            StatusCode: statusCode,
            Success:    success,
            Attempt:    attempt + 1,
            ErrorMsg:   errStr(err),
            CreatedAt:  time.Now().UTC(),
        })

        if success {
            s.repo.ResetFailCount(ctx, wh.ID)
            return
        }
        s.logger.Warn("webhook delivery failed", "attempt", attempt+1,
            "webhook_id", wh.ID, "status", statusCode)
    }

    // All 3 attempts failed
    if err := s.repo.IncrementFailCount(ctx, wh.ID); err == nil {
        // Check if we should degrade the webhook
        updated, _ := s.repo.Get(ctx, wh.ID)
        if updated != nil && updated.FailCount >= 3 {
            s.repo.UpdateStatus(ctx, wh.ID, domain.WebhookStatusDegraded)
            s.logger.Warn("webhook degraded after 3 consecutive failures",
                "webhook_id", wh.ID, "url", wh.URL)
        }
    }
}

// sendHTTP POSTs payload with HMAC-SHA256 signature
func (s *WebhookDeliveryService) sendHTTP(ctx context.Context, wh *domain.Webhook, payload []byte, deliveryID string) (int, error) {
    // Decrypt HMAC secret
    secret := decryptSecret(wh.SecretEnc) // AES-GCM decrypt

    // Compute HMAC-SHA256
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(payload))
    if err != nil { return 0, err }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-VNP-Signature", sig)
    req.Header.Set("X-VNP-Event", extractEventFromPayload(payload))
    req.Header.Set("X-VNP-Delivery", deliveryID)
    req.Header.Set("User-Agent", "VNP-Memory-Webhook/3.0")

    resp, err := s.httpClient.Do(req)
    if err != nil { return 0, err }
    defer resp.Body.Close()
    return resp.StatusCode, nil
}

// SendTestEvent delivers a sample payload immediately (for /webhooks/{id}/test)
func (s *WebhookDeliveryService) SendTestEvent(ctx context.Context, webhookID string) (*domain.WebhookDelivery, error) {
    wh, err := s.repo.Get(ctx, webhookID)
    if err != nil { return nil, err }

    testPayload := domain.WebhookPayload{
        Event:     "memory.stored",
        Data:      json.RawMessage(`{"test": true, "memory_id": "test-123"}`),
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
    payloadJSON, _ := json.Marshal(testPayload)

    statusCode, deliveryErr := s.sendHTTP(ctx, wh, payloadJSON, uuid.NewString())
    delivery := &domain.WebhookDelivery{
        ID:         uuid.NewString(),
        WebhookID:  webhookID,
        Event:      "memory.stored",
        StatusCode: statusCode,
        Success:    deliveryErr == nil && statusCode < 300,
        Attempt:    1,
        CreatedAt:  time.Now().UTC(),
    }
    s.repo.RecordDelivery(ctx, delivery)
    return delivery, nil
}
```

---

## Acceptance Criteria

- [ ] Service subscribes to all 6 NATS subjects on startup
- [ ] NATS event → fan-out to all matching webhooks for that tenant
- [ ] HTTP POST with `X-VNP-Signature: sha256={hex}` header
- [ ] Exponential backoff: 3 attempts (immediate → 5s → 25s)
- [ ] Delivery recorded after each attempt (success or failure)
- [ ] After 3 consecutive failures → webhook status=degraded
- [ ] `FailCount` reset to 0 on any successful delivery
- [ ] `SendTestEvent()` sends immediately without NATS trigger
- [ ] HTTP client timeout: 10 seconds
- [ ] `go build ./services/vnp-platform/...` passes

## Files

```
services/vnp-platform/internal/usecase/event_mapping.go        [NEW]
services/vnp-platform/internal/usecase/webhook_delivery.go     [NEW]
services/vnp-platform/internal/adapter/postgres/webhook_repo.go [NEW]
```
