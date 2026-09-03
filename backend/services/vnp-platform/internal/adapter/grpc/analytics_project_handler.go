package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/analytics"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/project"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AnalyticsHandler implements SmAnalyticsService gRPC.
type AnalyticsHandler struct {
	analytics port.AnalyticsUseCase
}

func NewAnalyticsHandler(a port.AnalyticsUseCase) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: a}
}

func (h *AnalyticsHandler) TrackUsage(ctx context.Context, record *analytics.UsageRecord) error {
	return h.analytics.TrackUsage(ctx, record)
}

func (h *AnalyticsHandler) GetUsageReport(ctx context.Context, tenantIDStr, period string) ([]*analytics.UsageRecord, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	return h.analytics.GetUsageReport(ctx, tenantID, period)
}

// ProjectHandler implements SmProjectService gRPC.
type ProjectHandler struct {
	projects port.ProjectUseCase
}

func NewProjectHandler(p port.ProjectUseCase) *ProjectHandler {
	return &ProjectHandler{projects: p}
}

func (h *ProjectHandler) CreateSpace(ctx context.Context, tenantIDStr, name string) (*project.Space, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	return h.projects.CreateSpace(ctx, tenantID, name)
}

func (h *ProjectHandler) ListSpaces(ctx context.Context, tenantIDStr string) ([]*project.Space, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant id")
	}
	return h.projects.ListSpaces(ctx, tenantID)
}

func (h *ProjectHandler) DeleteSpace(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid id")
	}
	return h.projects.DeleteSpace(ctx, id)
}
