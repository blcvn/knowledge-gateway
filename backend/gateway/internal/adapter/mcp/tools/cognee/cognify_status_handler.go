package cogneetools

import (
	"context"
	"fmt"

	cognifypb "github.com/vnp-memory/api/proto/cognee/cognify/v1"
)

// CognifyStatusHandler implements the "cognify_status" MCP tool.
type CognifyStatusHandler struct {
	cognifyClient cognifypb.CognifyServiceClient
}

// NewCognifyStatusHandler constructs a CognifyStatusHandler.
func NewCognifyStatusHandler(client cognifypb.CognifyServiceClient) *CognifyStatusHandler {
	return &CognifyStatusHandler{cognifyClient: client}
}

// Handle processes the "cognify_status" MCP tool call.
func (h *CognifyStatusHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
	runID     := getString(input, "pipeline_run_id")
	datasetID := getString(input, "dataset_id")

	if runID == "" && datasetID == "" {
		return nil, fmt.Errorf("pipeline_run_id or dataset_id is required")
	}

	resp, err := h.cognifyClient.GetPipelineStatus(ctx, &cognifypb.GetPipelineStatusRequest{
		PipelineRunId: runID,
		DatasetId:     datasetID,
	})
	if err != nil { return nil, fmt.Errorf("get pipeline status: %w", err) }

	result := map[string]any{
		"pipeline_run_id": resp.PipelineRunId,
		"status":          resp.Status,
	}
	if resp.NewNodes > 0  { result["new_nodes"] = resp.NewNodes }
	if resp.NewEdges > 0  { result["new_edges"] = resp.NewEdges }
	if resp.Error != ""   { result["error"] = resp.Error }
	return result, nil
}
