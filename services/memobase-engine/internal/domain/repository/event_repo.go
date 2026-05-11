package repository

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/domain/model"
)

// EventRepository defines the interface for interacting with user events.
type EventRepository interface {
	// SaveEvent persists a new user event.
	SaveEvent(ctx context.Context, event model.UserEvent) error
}

// EventGistRepository defines the interface for interacting with event gists.
type EventGistRepository interface {
	// SaveGist persists a new event gist.
	SaveGist(ctx context.Context, gist model.EventGist) error
}
