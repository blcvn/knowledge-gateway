// Package memory defines domain entities for the Zep memory sub-domain.
// This is the critical hot path — PutMemory must be sub-200ms p95.
package memory

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a conversation message stored in memory.
type Message struct {
	ID        uuid.UUID `json:"id"`
	ThreadID  uuid.UUID `json:"thread_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ContextAssembly represents an assembled context window for retrieval.
// Must be assembled in sub-200ms (critical performance requirement).
type ContextAssembly struct {
	ThreadID     uuid.UUID      `json:"thread_id"`
	Messages     []Message      `json:"messages"`
	Facts        []Fact         `json:"facts"`     // Priority-based fact overlay
	Summary      string         `json:"summary,omitempty"`
	TokenCount   int            `json:"token_count"`
	AssembledAt  time.Time      `json:"assembled_at"`
}

// Fact represents a knowledge fact extracted by zep-graph.
type Fact struct {
	ID         uuid.UUID `json:"id"`
	Content    string    `json:"content"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"` // graph, user, system
	CreatedAt  time.Time `json:"created_at"`
}
