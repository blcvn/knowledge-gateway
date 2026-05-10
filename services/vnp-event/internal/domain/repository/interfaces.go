package repository

import (
	"context"
	"time"
	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-event/internal/domain/model"
)

// EventRepository persists user events with vector search.
type EventRepository interface {
	Create(ctx context.Context, e *model.UserEvent) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.UserEvent, error)

	// SearchSemantic performs vector similarity search using pgvector.
	SearchSemantic(ctx context.Context, tenantID uuid.UUID, embedding []float32, limit int) ([]model.TimelineEntry, error)

	// SearchTemporal returns events within a time range.
	SearchTemporal(ctx context.Context, tenantID uuid.UUID, start, end time.Time, limit int) ([]*model.UserEvent, error)

	// GetTimeline returns a user's events ordered by valid_at.
	GetTimeline(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]*model.UserEvent, error)

	// FilterByTags returns events matching any of the given tags.
	FilterByTags(ctx context.Context, tenantID uuid.UUID, tags []string, limit int) ([]*model.UserEvent, error)
}

// GistRepository persists event gists.
type GistRepository interface {
	Create(ctx context.Context, g *model.EventGist) error
	SearchSemantic(ctx context.Context, tenantID uuid.UUID, embedding []float32, limit int) ([]*model.EventGist, error)
}
