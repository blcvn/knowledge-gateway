package grpc

import (
	"context"
	"vnp-memory/services/memory-service/internal/consolidation"
	"vnp-memory/services/memory-service/internal/consolidation/port"

	memorypb "github.com/vnp-memory/api/proto/memory/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConsolidationHandler struct {
	memorypb.UnimplementedConsolidationServiceServer
	pipeline    *consolidation.ConsolidationPipeline
	summaryRepo port.ISessionSummaryRepo
	procRepo    port.IProceduralRepo
	lessonRepo  port.ILessonRepo
	insightRepo port.IInsightRepo
}

func (h *ConsolidationHandler) SummarizeSession(ctx context.Context, req *memorypb.SummarizeSessionRequest) (*memorypb.SummarizeSessionResponse, error) {
	h.pipeline.SummarizeNow(ctx, req.SessionId)
	return &memorypb.SummarizeSessionResponse{Ok: true}, nil
}

func (h *ConsolidationHandler) RunPipeline(ctx context.Context, req *memorypb.RunPipelineRequest) (*memorypb.RunPipelineResponse, error) {
	go h.pipeline.SummarizeNow(context.Background(), req.SessionId)
	return &memorypb.RunPipelineResponse{Ok: true}, nil
}

func (h *ConsolidationHandler) ListProcedural(ctx context.Context, req *memorypb.ListProceduralRequest) (*memorypb.ListProceduralResponse, error) {
	items, err := h.procRepo.ListByTenant(ctx, req.TenantId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list procedural: %v", err)
	}
	return mapProceduralResponse(items), nil
}

func (h *ConsolidationHandler) ListLessons(ctx context.Context, req *memorypb.ListLessonsRequest) (*memorypb.ListLessonsResponse, error) {
	items, err := h.lessonRepo.ListByTenant(ctx, req.TenantId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list lessons: %v", err)
	}
	return mapLessonsResponse(items), nil
}

func (h *ConsolidationHandler) ListInsights(ctx context.Context, req *memorypb.ListInsightsRequest) (*memorypb.ListInsightsResponse, error) {
	items, err := h.insightRepo.ListByTenant(ctx, req.TenantId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list insights: %v", err)
	}
	return mapInsightsResponse(items), nil
}
