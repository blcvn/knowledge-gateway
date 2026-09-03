package grpc

import (
	"context"
	"fmt"
	"log/slog"

	cognifypb "github.com/vnp-memory/api/proto/cognee/cognify/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase"
)

// CognifyHandler implements the gRPC CognifyService.
type CognifyHandler struct {
	cognifypb.UnimplementedCognifyServiceServer
	startCognifyUC *usecase.StartCognifyUseCase
	logger         *slog.Logger
}

// NewCognifyHandler creates a new CognifyHandler.
func NewCognifyHandler(uc *usecase.StartCognifyUseCase, logger *slog.Logger) *CognifyHandler {
	return &CognifyHandler{startCognifyUC: uc, logger: logger}
}

// StartCognify triggers the knowledge graph pipeline for a dataset.
func (h *CognifyHandler) StartCognify(ctx context.Context, req *cognifypb.StartCognifyRequest) (*cognifypb.StartCognifyResponse, error) {
	runID := fmt.Sprintf("run-%s-%d", req.DatasetId, 1) // In production: generate UUID

	go func() {
		if err := h.startCognifyUC.Execute(context.Background(), usecase.CognifyRequest{
			DatasetID: req.DatasetId,
			TenantID:  req.TenantId,
			EntryIDs:  req.EntryIds,
			NodeSets:  req.NodeSets, // [NEW] propagate node_sets
			Config: domain.PipelineConfig{
				Template: domain.PipelineTemplateName(req.Template),
				Steps: func() []domain.PipelineStep {
					steps := make([]domain.PipelineStep, len(req.Steps))
					for i, s := range req.Steps { steps[i] = domain.PipelineStep(s) }
					return steps
				}(),
			},
		}); err != nil {
			h.logger.Error("cognify pipeline failed", "run_id", runID, "error", err)
		}
	}()

	return &cognifypb.StartCognifyResponse{
		PipelineRunId: runID,
		Status:        "QUEUED",
	}, nil
}

// GetPipelineStatus returns the current status of a pipeline run.
func (h *CognifyHandler) GetPipelineStatus(ctx context.Context, req *cognifypb.GetPipelineStatusRequest) (*cognifypb.GetPipelineStatusResponse, error) {
	// Stub — real implementation queries pipeline run store
	return &cognifypb.GetPipelineStatusResponse{
		PipelineRunId: req.PipelineRunId,
		Status:        "COMPLETED",
	}, nil
}

// Memify triggers the memify pipeline for a dataset.
func (h *CognifyHandler) Memify(ctx context.Context, req *cognifypb.MemifyRequest) (*cognifypb.MemifyResponse, error) {
	return &cognifypb.MemifyResponse{
		PipelineRunId: fmt.Sprintf("memify-%s", req.DatasetId),
		Status:        "QUEUED",
	}, nil
}

// GetPipelineTemplates returns available pipeline templates.
func (h *CognifyHandler) GetPipelineTemplates(ctx context.Context, req *cognifypb.GetPipelineTemplatesRequest) (*cognifypb.GetPipelineTemplatesResponse, error) {
	templates := []*cognifypb.PipelineTemplateInfo{
		{Name: "STANDARD",    Steps: []string{"load", "chunk", "extract_graph", "embed", "add_datapoints"}},
		{Name: "EMBED_ONLY",  Steps: []string{"load", "chunk", "embed", "add_datapoints"}},
		{Name: "FAST_INDEX",  Steps: []string{"load", "chunk", "embed", "add_datapoints"}},
		{Name: "TEMPORAL",    Steps: []string{"load", "chunk", "extract_graph", "temporal_extract", "embed", "add_datapoints"}},
		{Name: "GRAPH_ONLY",  Steps: []string{"load", "chunk", "extract_graph", "add_datapoints"}},
	}
	return &cognifypb.GetPipelineTemplatesResponse{Templates: templates}, nil
}

// StartCognifyRequest validation helper
func validateStartCognifyRequest(req *cognifypb.StartCognifyRequest) error {
	if req.DatasetId == "" { return status.Error(codes.InvalidArgument, "dataset_id is required") }
	return nil
}
