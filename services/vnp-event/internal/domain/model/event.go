// Package model defines domain entities for vnp-event.
// Reference: specs/tdd.md §2
package model

import (
	"time"
	"github.com/google/uuid"
)

// EventSource identifies which engine produced an event.
type EventSource string

const (
	SourceCognee      EventSource = "COGNEE"
	SourceGraphiti    EventSource = "GRAPHITI"
	SourceMemobase    EventSource = "MEMOBASE"
	SourceOpenViking  EventSource = "OPENVIKING"
	SourceZep         EventSource = "ZEP"
	SourceSupermemory EventSource = "SUPERMEMORY"
)

// UserEvent represents a temporal event from any engine.
type UserEvent struct {
	ID        uuid.UUID   `json:"id"`
	UserID    uuid.UUID   `json:"user_id"`
	TenantID  uuid.UUID   `json:"tenant_id"`
	Source    EventSource `json:"source"`
	Content   string      `json:"content"`
	Tags      []string    `json:"tags"`
	Embedding []float32   `json:"-"`        // vector(1536), not exposed in JSON
	CreatedAt time.Time   `json:"created_at"`
	ValidAt   time.Time   `json:"valid_at"` // bi-temporal: when this event was true
	InvalidAt *time.Time  `json:"invalid_at,omitempty"` // bi-temporal: when this event ceased being true
}

// EventGist summarizes a batch of events.
type EventGist struct {
	ID        uuid.UUID   `json:"id"`
	EventIDs  []uuid.UUID `json:"event_ids"`
	Summary   string      `json:"summary"`
	Embedding []float32   `json:"-"`
	CreatedAt time.Time   `json:"created_at"`
}

// TimelineEntry is a single item in a user's timeline view.
type TimelineEntry struct {
	Event  *UserEvent `json:"event"`
	Score  float64    `json:"score,omitempty"` // Relevance score for search results
}
