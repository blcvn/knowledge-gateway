package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/event"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EventHandler implements VnpEventService gRPC.
type EventHandler struct {
	events port.EventUseCase
}

func NewEventHandler(e port.EventUseCase) *EventHandler {
	return &EventHandler{events: e}
}

func (h *EventHandler) CreateEvent(ctx context.Context, evt *event.UserEvent) error {
	return h.events.CreateEvent(ctx, evt)
}

func (h *EventHandler) GetTimeline(ctx context.Context, tenantIDStr, userIDStr string, limit int) (*event.Timeline, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id")
	}
	return h.events.GetTimeline(ctx, tenantID, userID, limit)
}

func (h *EventHandler) SearchEvents(ctx context.Context, tenantIDStr, query string, limit int) ([]event.UserEvent, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	return h.events.SearchEvents(ctx, tenantID, query, limit)
}
