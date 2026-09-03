# TASK-PLAT-016 — Webhook HTTP Handlers & CRUD Routes

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-016 |
| **Wave** | 4 (Events) |
| **Solution** | [SOL-PLAT-005](../solutions/SOL-PLAT-005-Webhook-Delivery-System.md) §2.3 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-015 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** console_sdk.go: ListWebhooks/CreateWebhook/DeleteWebhook handlers via vnp-admin forward
---

## Mục tiêu

Implement HTTP handlers cho webhook CRUD: List, Create, Update, Delete, List Deliveries, Test. Register tất cả routes.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/webhooks.go` [NEW]

```go
package handler

type WebhookHandler struct {
    svc port.WebhookService
}

// GET /v1/console/sdk/webhooks
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    webhooks, err := h.svc.List(r.Context(), auth.TenantID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    // Never expose SecretEnc in response
    type webhookResponse struct {
        ID        string   `json:"id"`
        URL       string   `json:"url"`
        Events    []string `json:"events"`
        Status    string   `json:"status"`
        FailCount int      `json:"fail_count"`
        CreatedAt string   `json:"created_at"`
    }
    resp := make([]webhookResponse, 0, len(webhooks))
    for _, wh := range webhooks {
        resp = append(resp, webhookResponse{
            ID:        wh.ID,
            URL:       wh.URL,
            Events:    wh.Events,
            Status:    string(wh.Status),
            FailCount: wh.FailCount,
            CreatedAt: wh.CreatedAt.Format(time.RFC3339),
        })
    }
    writeJSON(w, http.StatusOK, map[string]interface{}{"webhooks": resp})
}

// POST /v1/console/sdk/webhooks
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    var req struct {
        URL    string   `json:"url"`
        Events []string `json:"events"`
        Secret string   `json:"secret"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }

    // Validate URL
    if _, err := url.ParseRequestURI(req.URL); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_url", req.URL)
        return
    }

    // Validate events
    for _, ev := range req.Events {
        if !domain.IsValidEvent(ev) {
            writeError(w, http.StatusBadRequest, "invalid_event",
                fmt.Sprintf("%s is not a supported event type", ev))
            return
        }
    }

    if req.Secret == "" {
        writeError(w, http.StatusBadRequest, "invalid_request", "secret is required")
        return
    }

    webhook, err := h.svc.Create(r.Context(), auth.TenantID, req.URL, req.Events, req.Secret)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, map[string]interface{}{
        "id":         webhook.ID,
        "url":        webhook.URL,
        "events":     webhook.Events,
        "status":     webhook.Status,
        "created_at": webhook.CreatedAt.Format(time.RFC3339),
    })
}

// PUT /v1/console/sdk/webhooks/{id}
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    webhookID := chi.URLParam(r, "id")
    var req struct {
        URL    string   `json:"url"`
        Events []string `json:"events"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    if err := h.svc.Update(r.Context(), auth.TenantID, webhookID, req.URL, req.Events); err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// DELETE /v1/console/sdk/webhooks/{id}
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    webhookID := chi.URLParam(r, "id")
    if err := h.svc.Delete(r.Context(), auth.TenantID, webhookID); err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// GET /v1/console/sdk/webhooks/{id}/deliveries?limit=50
func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
    webhookID := chi.URLParam(r, "id")
    limit := queryIntDefault(r, "limit", 50)
    deliveries, err := h.svc.GetDeliveries(r.Context(), webhookID, limit)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, map[string]interface{}{"deliveries": deliveries})
}

// POST /v1/console/sdk/webhooks/{id}/test
func (h *WebhookHandler) Test(w http.ResponseWriter, r *http.Request) {
    webhookID := chi.URLParam(r, "id")
    delivery, err := h.svc.SendTestEvent(r.Context(), webhookID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "webhook_test_error", err.Error())
        return
    }
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "status_code": delivery.StatusCode,
        "success":     delivery.Success,
        "delivery_id": delivery.ID,
    })
}
```

### 2. Modify `gateway/adapter/handler/router.go` [MODIFY] — register webhook routes

```go
r.Route("/v1/console/sdk/webhooks", func(r chi.Router) {
    r.Use(requireAdmin)
    r.Get("/",               webhookH.List)
    r.Post("/",              webhookH.Create)
    r.Put("/{id}",           webhookH.Update)
    r.Delete("/{id}",        webhookH.Delete)
    r.Get("/{id}/deliveries", webhookH.ListDeliveries)
    r.Post("/{id}/test",     webhookH.Test)
})
```

---

## Acceptance Criteria

- [ ] `GET /v1/console/sdk/webhooks` lists webhooks without exposing SecretEnc
- [ ] `POST /v1/console/sdk/webhooks` validates URL + events + secret before creating
- [ ] Invalid event type → 400 với error message listing valid events
- [ ] `DELETE /v1/console/sdk/webhooks/{id}` → 204 No Content
- [ ] `GET /v1/console/sdk/webhooks/{id}/deliveries` returns last 50 delivery records
- [ ] `POST /v1/console/sdk/webhooks/{id}/test` sends test event và returns HTTP status received
- [ ] All endpoints require `admin` role
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/webhooks.go   [NEW]
gateway/adapter/handler/router.go     [MODIFY — register /v1/console/sdk/webhooks/* routes]
```
