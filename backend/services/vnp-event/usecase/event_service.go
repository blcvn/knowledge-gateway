// Package usecase implements event service business logic.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-event/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-event/domain/repository"
)

// EmbeddingService generates vector embeddings.
type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// EventService implements event timeline operations.
type EventService struct {
	events    repository.EventRepository
	gists     repository.GistRepository
	embedder  EmbeddingService
}

func NewEventService(events repository.EventRepository, gists repository.GistRepository, embedder EmbeddingService) *EventService {
	return &EventService{events: events, gists: gists, embedder: embedder}
}

// CreateEvent stores a new event with its embedding.
func (s *EventService) CreateEvent(ctx context.Context, tenantID, userID uuid.UUID, source model.EventSource, content string, tags []string, validAt time.Time) (*model.UserEvent, error) {
	embedding, err := s.embedder.Embed(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("embed event: %w", err)
	}

	event := &model.UserEvent{
		ID:        uuid.New(),
		UserID:    userID,
		TenantID:  tenantID,
		Source:    source,
		Content:   content,
		Tags:      tags,
		Embedding: embedding,
		CreatedAt: time.Now(),
		ValidAt:   validAt,
	}

	if err := s.events.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	return event, nil
}

// SearchEvents performs semantic search over events.
func (s *EventService) SearchEvents(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]model.TimelineEntry, error) {
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	return s.events.SearchSemantic(ctx, tenantID, embedding, limit)
}

// GetTimeline returns a user's events ordered by valid_at.
func (s *EventService) GetTimeline(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]*model.UserEvent, error) {
	return s.events.GetTimeline(ctx, tenantID, userID, limit)
}

// FilterByTags returns events matching any of the given tags.
func (s *EventService) FilterByTags(ctx context.Context, tenantID uuid.UUID, tags []string, limit int) ([]*model.UserEvent, error) {
	return s.events.FilterByTags(ctx, tenantID, tags, limit)
}

// SearchGists searches over event gists by semantic similarity.
func (s *EventService) SearchGists(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*model.EventGist, error) {
	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	return s.gists.SearchSemantic(ctx, tenantID, embedding, limit)
}
