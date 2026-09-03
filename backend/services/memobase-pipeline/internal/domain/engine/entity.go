// Package engine defines domain entities for the Memobase YOLO merge engine.
package engine

import (
	"time"

	"github.com/google/uuid"
)

// Profile represents a structured user profile derived from memory analysis.
type Profile struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	UserID    uuid.UUID      `json:"user_id"`
	Topics    []TopicEntry   `json:"topics"`
	Traits    map[string]any `json:"traits,omitempty"`
	Version   int            `json:"version"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// TopicEntry represents a topic extracted from user conversations.
type TopicEntry struct {
	Category string `json:"category"`
	Topic    string `json:"topic"`
	Weight   float64 `json:"weight"` // Importance score
}

// EventGist represents a summarized session event.
type EventGist struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    uuid.UUID `json:"user_id"`
	Summary   string    `json:"summary"` // LLM-generated
	KeyFacts  []string  `json:"key_facts"`
	CreatedAt time.Time `json:"created_at"`
}

// MergeResult represents the output of a YOLO merge operation.
type MergeResult struct {
	NewTopics  []TopicEntry `json:"new_topics"`
	UpdatedTraits map[string]any `json:"updated_traits"`
	Gist       *EventGist   `json:"gist"`
	LLMCalls   int          `json:"llm_calls"` // Must be exactly 3
}
