// Package session defines domain entities for chat session management.
//
// Part of storage-service (MERGE-P1-T4: absorbs ov-session)
package session

import "time"

// ChatSession represents an active or committed coding/chat session.
type ChatSession struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	BaseDir   string    `json:"base_dir"` // root directory for this session's workspace
	Status    string    `json:"status"`   // "open" | "committed"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message is a single message within a session.
type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user" | "assistant" | "system"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CommitRecord records the result of committing a session's file changes.
type CommitRecord struct {
	SessionID   string     `json:"session_id"`
	FilesDiff   []FileDiff `json:"files_diff"`
	Summary     string     `json:"summary"`
	CommittedAt time.Time  `json:"committed_at"`
}

// FileDiff records a single file change in a commit.
type FileDiff struct {
	Path   string `json:"path"`
	Action string `json:"action"` // "created" | "modified" | "deleted"
}
