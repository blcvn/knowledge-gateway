# TASK-PLAT-017 — WebSocket Hub (Connection Registry)

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-017 |
| **Wave** | 4 (Events) |
| **Solution** | [SOL-PLAT-006](../solutions/SOL-PLAT-006-WebSocket-Realtime-Events.md) §2.2 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** ws.go: WSHandler with connections map + mu sync.RWMutex hub (191 lines)
---

## Mục tiêu

Implement `WSHub` (per-tenant connection registry) và `WSClient` (single WebSocket connection). Write pump với ping/pong keepalive.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/ws_hub.go` [NEW]

```go
package handler

import (
    "encoding/json"
    "sync"
    "time"
    "github.com/gorilla/websocket"
)

// WSHub manages all active WebSocket connections grouped by tenant
type WSHub struct {
    clients map[string][]*WSClient // tenantID → []clients
    mu      sync.RWMutex
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
    if len(h.clients[c.tenantID]) == 0 {
        delete(h.clients, c.tenantID)
    }
}

// ClientCount returns the number of connected clients for a tenant
func (h *WSHub) ClientCount(tenantID string) int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients[tenantID])
}

// WSClient represents a single WebSocket connection
type WSClient struct {
    conn     *websocket.Conn
    tenantID string
    send     chan []byte // buffered channel for outbound messages
}

const (
    writeWait      = 10 * time.Second // max time to write a message
    pongWait       = 60 * time.Second // max time between pings
    pingPeriod     = 54 * time.Second // ping interval (< pongWait)
    maxMessageSize = 512              // max read size from client
    sendBufSize    = 256             // send channel buffer
)

func NewWSClient(conn *websocket.Conn, tenantID string) *WSClient {
    return &WSClient{
        conn:     conn,
        tenantID: tenantID,
        send:     make(chan []byte, sendBufSize),
    }
}

// Send queues a message for delivery (non-blocking, drops if buffer full)
func (c *WSClient) Send(data []byte) bool {
    select {
    case c.send <- data:
        return true
    default:
        // Client too slow: drop message (log warning)
        return false
    }
}

// SendJSON marshals v and queues for delivery
func (c *WSClient) SendJSON(v interface{}) bool {
    data, err := json.Marshal(v)
    if err != nil {
        return false
    }
    return c.Send(data)
}

// WritePump drains c.send to the WebSocket connection
// Must run in its own goroutine
func (c *WSClient) WritePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case msg, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                // Hub closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
                return
            }

        case <-ticker.C:
            // Send ping to detect dead connections
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

### 2. Unit test `gateway/adapter/handler/ws_hub_test.go` [NEW]

```go
package handler_test

func TestWSHub_Register_Unregister(t *testing.T) {
    hub := NewWSHub()

    // Create mock client (no real WS conn needed for hub test)
    c := &WSClient{tenantID: "tenant-1", send: make(chan []byte, 10)}
    hub.Register(c)
    assert.Equal(t, 1, hub.ClientCount("tenant-1"))

    hub.Unregister(c)
    assert.Equal(t, 0, hub.ClientCount("tenant-1"))
}

func TestWSHub_Send_DropsWhenFull(t *testing.T) {
    c := &WSClient{tenantID: "t1", send: make(chan []byte, 1)}
    c.send <- []byte("fill buffer")
    // Second send should return false (dropped)
    ok := c.Send([]byte("overflow"))
    assert.False(t, ok, "should drop when buffer full")
}

func TestWSHub_MultiTenantIsolation(t *testing.T) {
    hub := NewWSHub()
    c1 := &WSClient{tenantID: "tenant-A", send: make(chan []byte, 10)}
    c2 := &WSClient{tenantID: "tenant-B", send: make(chan []byte, 10)}
    hub.Register(c1)
    hub.Register(c2)
    assert.Equal(t, 1, hub.ClientCount("tenant-A"))
    assert.Equal(t, 1, hub.ClientCount("tenant-B"))
}
```

### 3. Add gorilla/websocket dependency to `gateway/go.mod` [MODIFY]

```
require (
    github.com/gorilla/websocket v1.5.3
)
```

---

## Acceptance Criteria

- [ ] `WSHub.Register()` / `Unregister()` thread-safe (RWMutex)
- [ ] `WSClient.Send()` is non-blocking: returns false if buffer full (no goroutine block)
- [ ] `WritePump()` sends ping every 54s to detect dead connections
- [ ] `WritePump()` closes connection on write error
- [ ] Multiple tenants isolated: `hub.clients[tenantA]` and `hub.clients[tenantB]` separate
- [ ] `go test ./gateway/adapter/handler/...` passes

## Files

```
gateway/adapter/handler/ws_hub.go       [NEW]
gateway/adapter/handler/ws_hub_test.go  [NEW]
gateway/go.mod                          [MODIFY — add gorilla/websocket]
```
