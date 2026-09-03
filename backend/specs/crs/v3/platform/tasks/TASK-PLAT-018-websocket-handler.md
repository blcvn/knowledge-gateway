# TASK-PLAT-018 — WebSocket HTTP Handler (Upgrade + NATS Subscribe)

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-018 |
| **Wave** | 4 (Events) |
| **Solution** | [SOL-PLAT-006](../solutions/SOL-PLAT-006-WebSocket-Realtime-Events.md) §2.1 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-017 |
| **Estimated** | 4h |

---

## Mục tiêu

Implement `WebSocketHandler.Handle()`: upgrade HTTP → WebSocket, validate JWT từ query param, subscribe NATS `events.{tenant_id}.>`, forward events to client, handle client disconnect.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/websocket_handler.go` [NEW]

```go
package handler

import (
    "github.com/gorilla/websocket"
    "go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 4096,
    CheckOrigin: func(r *http.Request) bool {
        // TODO: validate against CORS_ORIGINS env var
        // For dev: allow all
        return true
    },
}

type WebSocketHandler struct {
    hub    *WSHub
    nats   port.NATSSubscriber
    buffer *WSEventBuffer
    auth   port.JWTValidator
    logger *zap.Logger
}

func NewWebSocketHandler(
    hub *WSHub,
    nats port.NATSSubscriber,
    buffer *WSEventBuffer,
    auth port.JWTValidator,
    logger *zap.Logger,
) *WebSocketHandler {
    return &WebSocketHandler{hub: hub, nats: nats, buffer: buffer, auth: auth, logger: logger}
}

// GET /v1/console/ws?token=<jwt>&last_event_id=<id>
func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // 1. Validate JWT from query param (can't use Authorization header for WS)
    token := r.URL.Query().Get("token")
    if token == "" {
        http.Error(w, "missing token", http.StatusUnauthorized)
        return
    }

    claims, err := h.auth.Validate(token)
    if err != nil {
        h.logger.Warn("ws: invalid jwt", zap.Error(err))
        http.Error(w, "invalid token", http.StatusUnauthorized)
        return
    }

    // 2. Upgrade to WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error("ws: upgrade failed", zap.Error(err))
        return
    }

    tenantID := claims.TenantID
    lastEventID := r.URL.Query().Get("last_event_id")

    client := NewWSClient(conn, tenantID)
    h.hub.Register(client)

    h.logger.Info("ws: client connected", zap.String("tenant_id", tenantID))

    // 3. Start write pump in background
    go client.WritePump()

    // 4. Send connected confirmation
    client.SendJSON(map[string]interface{}{
        "event":     "connected",
        "data":      map[string]string{"tenant_id": tenantID},
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    })

    // 5. Replay missed events if last_event_id provided
    if lastEventID != "" {
        go h.buffer.Replay(r.Context(), client, tenantID, lastEventID)
    }

    // 6. Subscribe to NATS: events.{tenant_id}.> (tenant-scoped, zero cross-tenant leakage)
    natsSubject := fmt.Sprintf("events.%s.>", tenantID)
    sub, err := h.nats.Subscribe(natsSubject, func(msg *nats.Msg) {
        if !client.Send(msg.Data) {
            h.logger.Warn("ws: slow consumer, message dropped",
                zap.String("tenant_id", tenantID))
        }
        // Buffer event for replay
        h.buffer.Store(context.Background(), tenantID, msg.Data)
    })
    if err != nil {
        h.logger.Error("ws: nats subscribe failed", zap.Error(err))
        h.hub.Unregister(client)
        conn.Close()
        return
    }
    defer func() {
        sub.Unsubscribe()
        h.hub.Unregister(client)
        h.logger.Info("ws: client disconnected", zap.String("tenant_id", tenantID))
    }()

    // 7. Read pump — blocks until client disconnects
    h.readPump(client)
}

// readPump handles incoming client messages and detects disconnect
func (h *WebSocketHandler) readPump(client *WSClient) {
    defer client.conn.Close()

    client.conn.SetReadLimit(maxMessageSize)
    client.conn.SetReadDeadline(time.Now().Add(pongWait))
    client.conn.SetPongHandler(func(string) error {
        client.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, msg, err := client.conn.ReadMessage()
        if err != nil {
            // Normal disconnect or error
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                h.logger.Info("ws: client closed unexpectedly",
                    zap.String("tenant_id", client.tenantID), zap.Error(err))
            }
            return
        }

        // Handle client reconnect message: {"last_event_id": "xxx"}
        var clientMsg struct {
            LastEventID string `json:"last_event_id"`
        }
        if json.Unmarshal(msg, &clientMsg) == nil && clientMsg.LastEventID != "" {
            go h.buffer.Replay(context.Background(), client, client.tenantID, clientMsg.LastEventID)
        }
    }
}
```

### 2. Modify `gateway/adapter/handler/router.go` [MODIFY] — register WebSocket route

```go
// Public-ish: auth done via query param ?token=<jwt>
// Do NOT wrap with standard authMiddleware (JWT comes from query param, not header)
mux.Get("/v1/console/ws", wsHandler.Handle)
```

---

## Acceptance Criteria

- [ ] `GET /v1/console/ws?token=<valid_jwt>` → WebSocket connection established
- [ ] Invalid/missing JWT → 401 before upgrade (no WebSocket connection)
- [ ] Connected message sent immediately: `{"event":"connected","data":{"tenant_id":"..."}}`
- [ ] NATS subscription scoped to `events.{tenant_id}.>` — tenant isolation at protocol level
- [ ] Event forwarding within 500ms of NATS publish
- [ ] Slow consumer: message dropped (non-blocking), warning logged
- [ ] Client disconnect triggers NATS unsubscribe + hub unregister (no goroutine leak)
- [ ] Client sends `{"last_event_id":"xxx"}` → server replays missed events
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/websocket_handler.go   [NEW]
gateway/adapter/handler/router.go              [MODIFY — register /v1/console/ws]
```
