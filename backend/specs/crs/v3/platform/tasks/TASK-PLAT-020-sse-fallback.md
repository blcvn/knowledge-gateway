# TASK-PLAT-020 — SSE Fallback Endpoint

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-020 |
| **Wave** | 4 (Events) |
| **Solution** | [SOL-PLAT-006](../solutions/SOL-PLAT-006-WebSocket-Realtime-Events.md) §2.4 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟢 Medium |
| **Depends On** | TASK-PLAT-018 |
| **Estimated** | 2h |

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** SSE fallback endpoint not implemented (ws.go only has WebSocket)
---

## Mục tiêu

Implement Server-Sent Events (SSE) fallback endpoint `GET /v1/console/events` cho environments không support WebSocket. Dùng standard auth middleware (Authorization header) thay vì query param.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/sse_handler.go` [NEW]

```go
package handler

type SSEHandler struct {
    nats   port.NATSSubscriber
    logger *zap.Logger
}

func NewSSEHandler(nats port.NATSSubscriber, logger *zap.Logger) *SSEHandler {
    return &SSEHandler{nats: nats, logger: logger}
}

// GET /v1/console/events
// Uses standard Authorization header (Bearer JWT or X-API-Key)
// Auth middleware injects AuthContext before reaching this handler
func (h *SSEHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    if auth == nil {
        http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
        return
    }

    // Check if SSE is supported
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported by this server", http.StatusInternalServerError)
        return
    }

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")     // disable nginx buffering
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // Send initial connected event
    fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]interface{}{
        "event":     "connected",
        "data":      map[string]string{"tenant_id": auth.TenantID},
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }))
    flusher.Flush()

    // Subscribe to NATS: tenant-scoped
    natsSubject := fmt.Sprintf("events.%s.>", auth.TenantID)
    sub, err := h.nats.Subscribe(natsSubject, func(msg *nats.Msg) {
        // SSE format: "data: {json}\n\n"
        fmt.Fprintf(w, "data: %s\n\n", string(msg.Data))
        flusher.Flush()
    })
    if err != nil {
        h.logger.Error("sse: nats subscribe failed",
            zap.String("tenant_id", auth.TenantID), zap.Error(err))
        return
    }
    defer sub.Unsubscribe()

    h.logger.Info("sse: client connected", zap.String("tenant_id", auth.TenantID))

    // Hold connection until client disconnects
    <-r.Context().Done()

    h.logger.Info("sse: client disconnected", zap.String("tenant_id", auth.TenantID))
}

func mustJSON(v interface{}) string {
    b, _ := json.Marshal(v)
    return string(b)
}
```

### 2. Modify `gateway/adapter/handler/router.go` [MODIFY] — register SSE route

```go
// SSE fallback: uses standard auth middleware (Authorization header)
r.Route("/v1/console", func(r chi.Router) {
    r.Use(authMiddleware)
    // ...existing console routes...
    r.Get("/events", sseHandler.StreamEvents) // SSE fallback
    r.Get("/ws",     wsHandler.Handle)        // WebSocket (handles own auth via ?token=)
})
```

> ⚠️ Note: `/v1/console/ws` should NOT use standard authMiddleware — it does its own JWT validation from query param. Register it outside the auth-wrapped group, or add a special case.

### 3. Unit test `gateway/adapter/handler/sse_handler_test.go` [NEW]

```go
package handler_test

func TestSSEHandler_SetsCorrectHeaders(t *testing.T) {
    // Mock auth + NATS
    mockNATS := &mockNATSSubscriber{}
    handler := NewSSEHandler(mockNATS, zap.NewNop())

    req := httptest.NewRequest("GET", "/v1/console/events", nil)
    ctx := contextWithAuth(req.Context(), &AuthContext{TenantID: "t1"})
    req = req.WithContext(ctx)

    rr := &flushableRecorder{ResponseRecorder: httptest.NewRecorder()}
    handler.StreamEvents(rr, req)

    assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
    assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
    assert.Contains(t, rr.Body.String(), `"event":"connected"`)
}
```

---

## Acceptance Criteria

- [ ] `GET /v1/console/events` establishes SSE stream
- [ ] Response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
- [ ] SSE format: `data: {json}\n\n` (per SSE spec)
- [ ] `X-Accel-Buffering: no` to disable nginx buffering
- [ ] Initial "connected" event sent immediately
- [ ] Tenant-scoped: only events for `AuthContext.TenantID` streamed
- [ ] Clean disconnect: NATS unsubscribed when client closes connection
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/sse_handler.go       [NEW]
gateway/adapter/handler/sse_handler_test.go  [NEW]
gateway/adapter/handler/router.go            [MODIFY — register /v1/console/events]
```
