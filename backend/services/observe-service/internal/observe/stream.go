package observe

import (
    "sync"
)

type StreamEvent struct {
    Type      string `json:"type"`
    SessionID string `json:"session_id,omitempty"`
    Data      any    `json:"data"`
}

type StreamBroker struct {
    mu      sync.RWMutex
    clients map[string]map[chan StreamEvent]struct{}  // sessionID → channels ("" = all)
}

func NewStreamBroker() *StreamBroker {
    return &StreamBroker{clients: make(map[string]map[chan StreamEvent]struct{})}
}

func (sb *StreamBroker) Subscribe(sessionFilter string) (chan StreamEvent, func()) {
    ch := make(chan StreamEvent, 100)
    sb.mu.Lock()
    if sb.clients[sessionFilter] == nil {
        sb.clients[sessionFilter] = make(map[chan StreamEvent]struct{})
    }
    sb.clients[sessionFilter][ch] = struct{}{}
    sb.mu.Unlock()
    cancel := func() { sb.unsubscribe(sessionFilter, ch) }
    return ch, cancel
}

func (sb *StreamBroker) unsubscribe(filter string, ch chan StreamEvent) {
    sb.mu.Lock()
    defer sb.mu.Unlock()
    delete(sb.clients[filter], ch)
    close(ch)
}

func (sb *StreamBroker) Broadcast(event StreamEvent) {
    sb.broadcastToGroup("", event)
    if event.SessionID != "" { sb.broadcastToGroup(event.SessionID, event) }
}

func (sb *StreamBroker) broadcastToGroup(filter string, event StreamEvent) {
    sb.mu.RLock()
    defer sb.mu.RUnlock()
    for ch := range sb.clients[filter] {
        select {
        case ch <- event:
        default: // drop if subscriber is slow
        }
    }
}
