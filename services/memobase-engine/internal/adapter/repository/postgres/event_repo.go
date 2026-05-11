package postgres

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/domain/model"
	"vnp-memory/services/memobase-engine/internal/domain/repository"
)

type eventRepository struct {
	// db *sql.DB or *gorm.DB
}

// NewEventRepository creates a new PostgreSQL backed event repository.
func NewEventRepository() repository.EventRepository {
	return &eventRepository{}
}

func (r *eventRepository) SaveEvent(ctx context.Context, event model.UserEvent) error {
	return nil
}

type eventGistRepository struct {
	// db *sql.DB or *gorm.DB
}

// NewEventGistRepository creates a new PostgreSQL backed event gist repository.
func NewEventGistRepository() repository.EventGistRepository {
	return &eventGistRepository{}
}

func (r *eventGistRepository) SaveGist(ctx context.Context, gist model.EventGist) error {
	return nil
}
