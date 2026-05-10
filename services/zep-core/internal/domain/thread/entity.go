// Package thread defines domain entities for the Zep thread/session sub-domain.
package thread

import (
	"time"

	"github.com/google/uuid"
)

// Thread represents a conversation thread (session container).
type Thread struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	UserID    string         `json:"user_id"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	EndedAt   *time.Time     `json:"ended_at,omitempty"` // nil = active
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// IsActive returns true if the thread has not been ended.
func (t *Thread) IsActive() bool {
	return t.EndedAt == nil
}

// Session represents a session within a thread.
type Session struct {
	ID        uuid.UUID  `json:"id"`
	ThreadID  uuid.UUID  `json:"thread_id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}
