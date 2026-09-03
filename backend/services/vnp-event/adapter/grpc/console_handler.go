// Package grpc implements GDPR and timeline console handlers for vnp-event.
package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-event/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-event/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TimelineFilter defines query parameters for event timeline.
type TimelineFilter struct {
	UserID string
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

// TimelineResponse wraps paginated timeline events.
type TimelineResponse struct {
	Events []*model.TimelineEntry `json:"events"`
	Total  int                    `json:"total"`
}

// EventRepo interface for timeline queries.
type EventRepo interface {
	QueryTimeline(ctx context.Context, userID uuid.UUID, from, to *time.Time, offset, limit int) ([]*model.TimelineEntry, int, error)
}

// ConsoleEventHandler implements GDPR forget and timeline endpoints.
type ConsoleEventHandler struct {
	gdpr   *usecase.GDPRService
	events EventRepo
	logger *slog.Logger
}

// NewConsoleEventHandler creates a console event handler.
func NewConsoleEventHandler(gdpr *usecase.GDPRService, events EventRepo, logger *slog.Logger) *ConsoleEventHandler {
	return &ConsoleEventHandler{gdpr: gdpr, events: events, logger: logger}
}

// GDPRForgetPreview handles POST /api/v1/gdpr/forget/preview.
func (h *ConsoleEventHandler) GDPRForgetPreview(ctx context.Context, userID string) (*usecase.ForgetPreview, error) {
	if userID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id must not be empty")
	}

	preview, err := h.gdpr.Preview(ctx, userID)
	if err != nil {
		h.logger.Error("GDPR forget preview failed", "error", err, "user_id", userID)
		return nil, status.Errorf(codes.Internal, "preview failed: %v", err)
	}

	h.logger.Info("GDPR forget preview", "user_id", userID, "total_records", preview.TotalRecords)
	return preview, nil
}

// GDPRForget handles POST /api/v1/gdpr/forget.
func (h *ConsoleEventHandler) GDPRForget(ctx context.Context, req usecase.ForgetRequest) (*usecase.ForgetResult, error) {
	if req.UserID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id must not be empty")
	}

	result, err := h.gdpr.Forget(ctx, req)
	if err != nil {
		h.logger.Error("GDPR forget failed", "error", err, "user_id", req.UserID)
		return nil, status.Errorf(codes.Internal, "forget failed: %v", err)
	}

	h.logger.Info("GDPR forget completed",
		"forget_id", result.ForgetID,
		"status", result.Status,
		"total_deleted", result.TotalDeleted,
	)
	return result, nil
}

// GetTimeline handles GET /api/v1/events/timeline.
func (h *ConsoleEventHandler) GetTimeline(ctx context.Context, filter TimelineFilter) (*TimelineResponse, error) {
	if filter.UserID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user_id must not be empty")
	}

	userUUID, err := uuid.Parse(filter.UserID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %s", filter.UserID)
	}

	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}

	events, total, err := h.events.QueryTimeline(ctx, userUUID, filter.From, filter.To, filter.Offset, filter.Limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "timeline query failed: %v", err)
	}

	return &TimelineResponse{Events: events, Total: total}, nil
}
