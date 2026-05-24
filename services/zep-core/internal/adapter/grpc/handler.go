// Package grpc provides stub handlers for zep-core console endpoints.
// Returns mock data matching UI's session.ts types.
package grpc

import (
	"context"
	"encoding/json"
	"time"
)

// ZepCoreHandler provides stub console endpoints for session management.
type ZepCoreHandler struct{}

func NewZepCoreHandler() *ZepCoreHandler {
	return &ZepCoreHandler{}
}

type Session struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	Title        string `json:"title"`
	AgentID      string `json:"agent_id,omitempty"`
	Status       string `json:"status"`
	MessageCount int    `json:"message_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Message struct {
	ID            string   `json:"id"`
	Role          string   `json:"role"`
	Content       string   `json:"content"`
	Timestamp     string   `json:"timestamp"`
	MemorySources []string `json:"memory_sources,omitempty"`
}

type Conversation struct {
	SessionID string    `json:"session_id"`
	Messages  []Message `json:"messages"`
}

// ListSessions returns stub sessions.
func (h *ZepCoreHandler) ListSessions(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := []Session{
		{ID: "sess-001", UserID: "user-001", Title: "Architecture Discussion", AgentID: "agent-01", Status: "completed", MessageCount: 24, CreatedAt: now, UpdatedAt: now},
		{ID: "sess-002", UserID: "user-002", Title: "Code Review Session", AgentID: "agent-02", Status: "active", MessageCount: 12, CreatedAt: now, UpdatedAt: now},
		{ID: "sess-003", UserID: "user-001", Title: "Bug Investigation", Status: "active", MessageCount: 8, CreatedAt: now, UpdatedAt: now},
		{ID: "sess-004", UserID: "user-003", Title: "Feature Planning", AgentID: "agent-01", Status: "completed", MessageCount: 36, CreatedAt: now, UpdatedAt: now},
	}
	return json.Marshal(data)
}

// ListLiveSessions returns stub active sessions.
func (h *ZepCoreHandler) ListLiveSessions(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := []Session{
		{ID: "sess-002", UserID: "user-002", Title: "Code Review Session", AgentID: "agent-02", Status: "active", MessageCount: 12, CreatedAt: now, UpdatedAt: now},
		{ID: "sess-003", UserID: "user-001", Title: "Bug Investigation", Status: "active", MessageCount: 8, CreatedAt: now, UpdatedAt: now},
	}
	return json.Marshal(data)
}

// GetSession returns a stub conversation.
func (h *ZepCoreHandler) GetSession(_ context.Context, sessionID string) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := Conversation{
		SessionID: sessionID,
		Messages: []Message{
			{ID: "msg-001", Role: "user", Content: "Can you help me review this code?", Timestamp: now},
			{ID: "msg-002", Role: "assistant", Content: "Of course! Please share the code you'd like me to review.", Timestamp: now, MemorySources: []string{"mem-002", "mem-004"}},
			{ID: "msg-003", Role: "user", Content: "Here's the gRPC handler implementation...", Timestamp: now},
			{ID: "msg-004", Role: "assistant", Content: "I see a few areas for improvement...", Timestamp: now, MemorySources: []string{"mem-004"}},
		},
	}
	return json.Marshal(data)
}

// GetTimeline returns a stub session timeline.
func (h *ZepCoreHandler) GetTimeline(_ context.Context, sessionID string) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := map[string]interface{}{
		"entries": []map[string]interface{}{
			{"type": "message", "content": "Session started", "timestamp": now, "metadata": map[string]string{"session_id": sessionID}},
			{"type": "memory_recall", "content": "Recalled 3 relevant memories", "timestamp": now},
			{"type": "message", "content": "User asked for code review", "timestamp": now},
			{"type": "memory_store", "content": "Stored new memory: code review context", "timestamp": now},
		},
	}
	return json.Marshal(data)
}
