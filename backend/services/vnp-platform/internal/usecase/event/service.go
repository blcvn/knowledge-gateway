// Package event implements the event timeline usecase for vnp-platform.
package event

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/event"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
)

// Service implements port.EventUseCase.
type Service struct {
	repo port.EventRepository
}

func NewService(repo port.EventRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateEvent(ctx context.Context, evt *event.UserEvent) error {
	evt.ID = uuid.New()
	evt.CreatedAt = time.Now()
	return s.repo.Create(ctx, evt)
}

func (s *Service) GetTimeline(ctx context.Context, tenantID, userID uuid.UUID, limit int) (*event.Timeline, error) {
	if limit <= 0 {
		limit = 50
	}
	events, total, err := s.repo.FindByUser(ctx, tenantID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("find events: %w", err)
	}
	return &event.Timeline{
		UserID:   userID,
		TenantID: tenantID,
		Events:   events,
		Total:    total,
	}, nil
}

func (s *Service) SearchEvents(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]event.UserEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.SearchByText(ctx, tenantID, query, limit)
}
