// Package event defines domain entities for the event sub-domain.
//
// Absorbed from: vnp-event
package event

import (
	"time"

	"github.com/google/uuid"
)

// UserEvent represents a cross-domain event in the timeline.
type UserEvent struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	UserID    uuid.UUID      `json:"user_id"`
	Engine    string         `json:"engine"`   // cognee, graphiti, memobase, openviking, zep, supermemory
	Type      EventType      `json:"type"`     // ingestion, search, memory, profile, admin
	Action    string         `json:"action"`   // created, updated, deleted, searched, ingested
	Payload   map[string]any `json:"payload,omitempty"`
	GistText  string         `json:"gist_text,omitempty"` // LLM-generated summary
	CreatedAt time.Time      `json:"created_at"`
}

// EventType categorizes events across engines.
type EventType string

const (
	EventIngestion EventType = "ingestion"
	EventSearch    EventType = "search"
	EventMemory    EventType = "memory"
	EventProfile   EventType = "profile"
	EventAdmin     EventType = "admin"
)

// Timeline represents a user's event history for context.
type Timeline struct {
	TenantID uuid.UUID    `json:"tenant_id"`
	UserID   uuid.UUID    `json:"user_id"`
	Events   []UserEvent  `json:"events"`
	Total    int          `json:"total"`
}
