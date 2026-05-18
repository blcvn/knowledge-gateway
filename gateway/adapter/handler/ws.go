// Package handler — WebSocket realtime handler for Console UI (FEAT-012/T07).
package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/infra/middleware"
)

// WSHandler handles WS /v1/console/ws — realtime streaming.
type WSHandler struct {
	logger      *slog.Logger
	mu          sync.RWMutex
	connections map[string]*wsConnection
}

// wsConnection represents a single WebSocket client connection.
type wsConnection struct {
	id       string
	channels map[string]bool
	send     chan []byte
	done     chan struct{}
}

// WSMessage is the WebSocket message envelope.
type WSMessage struct {
	Channel   string `json:"channel"`
	Event     string `json:"event"`
	Data      any    `json:"data"`
	Timestamp string `json:"timestamp"`
}

// WSSubscribe is the client subscribe/unsubscribe message.
type WSSubscribe struct {
	Action   string   `json:"action"` // "subscribe" | "unsubscribe"
	Channels []string `json:"channels"`
}

// NewWSHandler creates a new WebSocket handler.
func NewWSHandler(logger *slog.Logger) *WSHandler {
	return &WSHandler{
		logger:      logger,
		connections: make(map[string]*wsConnection),
	}
}

// HandleWS handles the WebSocket upgrade and message loop.
// Note: This is a simplified SSE-based implementation using Server-Sent Events
// for environments where native WebSocket upgrade is not available.
// For production, replace with gorilla/websocket or nhooyr.io/websocket.
func (h *WSHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	// Validate auth
	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		WriteError(w, domain.ErrUnauthenticated)
		return
	}

	// Check admin role
	isAdmin := false
	for _, role := range auth.Roles {
		if role == "admin" || role == "super_admin" {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		WriteError(w, domain.ErrForbidden.WithMessage("admin role required for console WebSocket"))
		return
	}

	// Use SSE as fallback transport for WebSocket-like streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, domain.ErrCircuitOpen.WithMessage("streaming not supported"))
		return
	}

	reqID := middleware.RequestIDFromContext(r.Context())
	conn := &wsConnection{
		id:       reqID,
		channels: make(map[string]bool),
		send:     make(chan []byte, 256),
		done:     make(chan struct{}),
	}

	// Default channels from query param
	channelsParam := r.URL.Query().Get("channels")
	if channelsParam != "" {
		var channels []string
		if err := json.Unmarshal([]byte(channelsParam), &channels); err == nil {
			for _, ch := range channels {
				conn.channels[ch] = true
			}
		}
	}

	// Register connection
	h.mu.Lock()
	h.connections[conn.id] = conn
	h.mu.Unlock()

	h.logger.Info("ws connection established",
		"connection_id", conn.id,
		"tenant_id", auth.TenantID,
		"channels", len(conn.channels),
	)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Send connected event
	connEvent, _ := json.Marshal(WSMessage{
		Channel: "system",
		Event:   "connected",
		Data:    map[string]string{"connection_id": conn.id},
	})
	w.Write([]byte("data: " + string(connEvent) + "\n\n"))
	flusher.Flush()

	// Stream loop
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			h.removeConnection(conn.id)
			return
		case msg, ok := <-conn.send:
			if !ok {
				return
			}
			w.Write([]byte("data: " + string(msg) + "\n\n"))
			flusher.Flush()
		case <-conn.done:
			return
		}
	}
}

// Broadcast sends a message to all connected clients subscribed to the channel.
func (h *WSHandler) Broadcast(channel, event string, data any) {
	msg, err := json.Marshal(WSMessage{
		Channel: channel,
		Event:   event,
		Data:    data,
	})
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, conn := range h.connections {
		if conn.channels[channel] || len(conn.channels) == 0 {
			select {
			case conn.send <- msg:
			default:
				// Client too slow, skip
			}
		}
	}
}

// removeConnection cleans up a disconnected client.
func (h *WSHandler) removeConnection(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, ok := h.connections[id]; ok {
		close(conn.done)
		delete(h.connections, id)
		h.logger.Info("ws connection closed", "connection_id", id)
	}
}

// ConnectionCount returns the number of active connections.
func (h *WSHandler) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}
