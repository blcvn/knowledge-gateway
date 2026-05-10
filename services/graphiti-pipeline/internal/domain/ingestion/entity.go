// Package ingestion defines domain entities for the Graphiti episode ingestion.
package ingestion

import (
	"time"

	"github.com/google/uuid"
)

// Episode represents a conversational episode for graph construction.
type Episode struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	GroupID   string    `json:"group_id"` // Serialization key
	Content   string    `json:"content"`
	Speaker   string    `json:"speaker,omitempty"`
	Source    string    `json:"source"` // message, text, json
	ValidAt  time.Time `json:"valid_at"`
	CreatedAt time.Time `json:"created_at"`
}

// SagaState tracks the multi-step pipeline execution.
type SagaState struct {
	ID           uuid.UUID   `json:"id"`
	EpisodeID    uuid.UUID   `json:"episode_id"`
	CurrentStep  string      `json:"current_step"`
	CompletedAt  map[string]time.Time `json:"completed_steps"`
	Status       string      `json:"status"` // running, completed, compensating, failed
	Error        string      `json:"error,omitempty"`
}
