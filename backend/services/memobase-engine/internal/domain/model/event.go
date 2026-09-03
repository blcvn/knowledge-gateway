package model

import (
	"time"

	"github.com/google/uuid"
)

// UserEvent represents a temporal event associated with a user.
type UserEvent struct {
	ID        uuid.UUID
	UserID    string
	ProjectID string
	EventData map[string]interface{}
	Embedding []float32
	CreatedAt time.time
}

// EventGist represents a fine-grained chunk of an event for specific retrieval.
type EventGist struct {
	ID        uuid.UUID
	UserID    string
	ProjectID string
	EventID   uuid.UUID
	GistData  map[string]interface{}
	Embedding []float32
	CreatedAt time.time
}
