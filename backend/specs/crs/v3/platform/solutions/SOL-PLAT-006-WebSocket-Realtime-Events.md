# SOL-PLAT-006 — Solution: WebSocket Real-time Event Streaming

| Field | Value |
|---|---|
| **Solution ID** | SOL-PLAT-006 |
| **CR** | [CR-PLAT-006](../../../../docs/crs/v3/platform/CR-PLAT-006-WebSocket-Realtime-Events.md) |
| **TDD ref** | [01-gateway.md §4-Routes](../../../tdd/architecture/01-gateway.md) · [backend-api-specs.md §12.12-WebSocket](../../../tdd/backend-api-specs.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

---

## 1. Phân tích kiến trúc

Theo TDD `01-gateway.md §4`, gateway đã có SSE endpoint `GET /v1/observe/stream` trong AgentMemory handler. Theo `backend-api-specs.md §12.12`, `GET /v1/console/ws?token=<jwt>` chưa implement.

Cần:
- WebSocket upgrade với JWT auth từ query param
- NATS subscription scoped theo `tenant_id`: `events.{tenant_id}.>`
- Tenant isolation: zero cross-tenant event leakage
- Event buffer: last 100 events per tenant (Redis)
- Reconnect: client gửi `last_event_id` → server replays missed events
- SSE fallback: `GET /v1/console/events`

---

## 2. Giải pháp

### 2.1 `gateway/adapter/handler/websocket_handler.go` [NEW]

```go
package handler

import (
    "github.com/gorilla/websocket"
    "go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        // Allow configured origins (CORS_ORIGINS env)
        return true // TODO: validate against allowed origins list
    },
    ReadBufferSize:  1024,
    WriteBufferSize: 4096,
}

type WebSocketHandler struct {
    nats      port.NATSSubscriber
    redis     port.RedisClient
    auth      port.AuthUseCase
    logger    *zap.Logger
    hub       *WSHub
}

// GET /v1/console/ws?token=<jwt>
func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // Auth: validate JWT from query param
    token := r.URL.Query().Get("token")
    if token == "" {
        http.Error(w, "missing token", http.StatusUnauthorized)
        return
    }

    claims, err := h.auth.ValidateJWT(r.Context(), token)
    if err != nil {
        http.Error(w, "invalid token", http.StatusUnauthorized)
        return
    }

    // Upgrade HTTP → WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error("websocket upgrade failed", zap.Error(err))
        return
    }

    tenantID := claims.TenantID
    lastEventID := r.URL.Query().Get("last_event_id")

    client := &WSClient{
        conn:     conn,
        tenantID: tenantID,
        send:     make(chan []byte, 256),
    }

    h.hub.Register(client)
    defer h.hub.Unregister(client)

    // Send connected confirmation
    client.SendJSON(map[string]interface{}{
        "event": "connected",
        "data": map[string]string{
            "tenant_id": tenantID,
        },
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    })

    // Replay missed events if last_event_id provided
    if lastEventID != "" {
        h.replayMissedEvents(r.Context(), client, tenantID, lastEventID)
    }

    // Subscribe to NATS: events.{tenant_id}.> (tenant-scoped)
    natsSubject := fmt.Sprintf("events.%s.>", tenantID)
    sub, err := h.nats.Subscribe(natsSubject, func(msg *nats.Msg) {
        // Forward NATS event to WebSocket client
        client.Send(msg.Data)
        // Buffer in Redis (last 100)
        h.bufferEvent(r.Context(), tenantID, msg.Data)
    })
    if err != nil {
        h.logger.Error("nats subscribe failed", zap.Error(err))
        return
    }
    defer sub.Unsubscribe()

    // Read pump (handle client disconnect + ping/pong)
    h.readPump(client)
}

func (h *WebSocketHandler) readPump(client *WSClient) {
    defer client.conn.Close()
    client.conn.SetReadLimit(512)
    client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    client.conn.SetPongHandler(func(string) error {
        client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, msg, err := client.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                h.logger.Info("client disconnected", zap.String("tenant_id", client.tenantID))
            }
            break
        }

        // Handle client messages (e.g., reconnect with last_event_id)
        var clientMsg struct {
            LastEventID string `json:"last_event_id"`
        }
        if json.Unmarshal(msg, &clientMsg) == nil && clientMsg.LastEventID != "" {
            go h.replayMissedEvents(context.Background(), client, client.tenantID, clientMsg.LastEventID)
        }
    }
}
```

### 2.2 `gateway/adapter/handler/ws_hub.go` [NEW]

```go
package handler

// WSHub manages all active WebSocket connections
type WSHub struct {
    clients    map[string][]*WSClient  // tenantID → []clients
    mu         sync.RWMutex
}

func NewWSHub() *WSHub {
    return &WSHub{clients: make(map[string][]*WSClient)}
}

func (h *WSHub) Register(c *WSClient) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.clients[c.tenantID] = append(h.clients[c.tenantID], c)
}

func (h *WSHub) Unregister(c *WSClient) {
    h.mu.Lock()
    defer h.mu.Unlock()
    clients := h.clients[c.tenantID]
    for i, cl := range clients {
        if cl == c {
            h.clients[c.tenantID] = append(clients[:i], clients[i+1:]...)
            break
        }
    }
    close(c.send)
}

// WSClient represents a single WebSocket connection
type WSClient struct {
    conn     *websocket.Conn
    tenantID string
    send     chan []byte
}

func (c *WSClient) Send(data []byte) {
    select {
    case c.send <- data:
    default:
        // Client slow: drop message
    }
}

func (c *WSClient) SendJSON(v interface{}) {
    data, _ := json.Marshal(v)
    c.Send(data)
}

// WritePump drains the send channel to WebSocket
func (c *WSClient) WritePump() {
    ticker := time.NewTicker(54 * time.Second) // ping interval
    defer ticker.Stop()
    defer c.conn.Close()

    for {
        select {
        case msg, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

### 2.3 Event Buffer (Redis) — replay missed events

```go
// gateway/adapter/handler/ws_buffer.go [NEW]

const eventBufferKey = "ws:events:%s"  // ws:events:{tenant_id}
const maxBufferSize = 100

func (h *WebSocketHandler) bufferEvent(ctx context.Context, tenantID string, payload []byte) {
    key := fmt.Sprintf(eventBufferKey, tenantID)
    // Add event with sequence ID
    seqID := fmt.Sprintf("%d", time.Now().UnixNano())

    h.redis.ZAdd(ctx, key, redis.Z{
        Score:  float64(time.Now().UnixNano()),
        Member: seqID + ":" + string(payload),
    })

    // Trim to last 100
    h.redis.ZRemRangeByRank(ctx, key, 0, -int64(maxBufferSize+1))
    h.redis.Expire(ctx, key, 24*time.Hour)
}

func (h *WebSocketHandler) replayMissedEvents(ctx context.Context, client *WSClient, tenantID, lastEventID string) {
    key := fmt.Sprintf(eventBufferKey, tenantID)

    // Get events newer than lastEventID (stored as score = unix nano)
    lastScore, err := strconv.ParseFloat(lastEventID, 64)
    if err != nil {
        return
    }

    events, err := h.redis.ZRangeByScore(ctx, key, &redis.ZRangeBy{
        Min: fmt.Sprintf("(%f", lastScore), // exclusive: events AFTER lastEventID
        Max: "+inf",
    }).Result()
    if err != nil {
        return
    }

    for _, ev := range events {
        // Strip seqID prefix and send payload
        parts := strings.SplitN(ev, ":", 2)
        if len(parts) == 2 {
            client.Send([]byte(parts[1]))
        }
    }
}
```

### 2.4 SSE Fallback — `gateway/adapter/handler/sse_handler.go` [NEW/MODIFY]

```go
// GET /v1/console/events — SSE fallback for environments without WebSocket

func (h *WebSocketHandler) SSEFallback(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context()) // from standard auth middleware (not query param)
    if auth == nil {
        writeError(w, http.StatusUnauthorized, "unauthorized", "")
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "data: %s\n\n", `{"event":"connected","data":{"tenant_id":"`+auth.TenantID+`"}}`)
    flusher.Flush()

    natsSubject := fmt.Sprintf("events.%s.>", auth.TenantID)
    sub, _ := h.nats.Subscribe(natsSubject, func(msg *nats.Msg) {
        fmt.Fprintf(w, "data: %s\n\n", string(msg.Data))
        flusher.Flush()
    })
    defer sub.Unsubscribe()

    // Hold connection until client disconnects
    <-r.Context().Done()
}
```

### 2.5 NATS Publishing Convention

All services must publish tenant-scoped events under `events.{tenant_id}.{event_type}`:

```go
// Example: publish from memory store success
h.nats.Publish(fmt.Sprintf("events.%s.memory_stored", tenantID), &MemoryStoredEvent{
    Event:    "memory_stored",
    Data: map[string]interface{}{
        "engine":    "graphiti",
        "type":      "episodic",
        "memory_id": memID,
    },
    Timestamp: time.Now().UTC().Format(time.RFC3339),
})
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `gateway/adapter/handler/websocket_handler.go` | NEW | WebSocket upgrade, auth, NATS subscription, disconnect |
| `gateway/adapter/handler/ws_hub.go` | NEW | WSHub: connection registry, per-tenant fan-out |
| `gateway/adapter/handler/ws_buffer.go` | NEW | Redis event buffer + replay missed events |
| `gateway/adapter/handler/sse_handler.go` | NEW/MODIFY | SSE fallback endpoint |
| `gateway/adapter/handler/router.go` | MODIFY | Register `/v1/console/ws` and `/v1/console/events` |
| `gateway/internal/infra/config/config.go` | VERIFY | CORS_ORIGINS, WS config |
| `services/*/internal/usecase/*.go` | MODIFY | Publish to `events.{tenant_id}.{type}` NATS subject |

---

## 4. Acceptance Criteria

- [ ] `GET /v1/console/ws?token=<jwt>` → WebSocket upgrade succeeds với valid JWT
- [ ] Invalid JWT → close connection with WebSocket 1008 (Policy Violation) code
- [ ] Memory stored event delivered within 500ms of NATS event
- [ ] Health change event delivered within 200ms
- [ ] Tenant isolation: NATS subject `events.{tenant_id}.>` — zero cross-tenant event delivery at protocol level
- [ ] Event buffer: Redis stores last 100 events per tenant (24h TTL)
- [ ] Client reconnect: sends `{"last_event_id": "..."}` → server replays missed events
- [ ] SSE fallback `GET /v1/console/events` functional for non-WebSocket environments
- [ ] Server sends ping every 54s to detect dead connections
- [ ] Slow consumers: message dropped (not blocked), no goroutine leak

---

## 5. Dependencies

- `github.com/gorilla/websocket` library
- NATS JetStream (publish from all engine services)
- Redis (event buffer ring, 24h TTL)
- Auth middleware (SOL-PLAT-001 JWT RS256 validation)
- All engine services must adopt `events.{tenant_id}.{type}` NATS publish pattern
