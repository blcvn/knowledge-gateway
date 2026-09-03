package cogneetools

import (
	"context"
	"fmt"

	ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
)

// DeleteDatasetHandler implements the "delete_dataset" MCP tool.
type DeleteDatasetHandler struct {
	ingestionClient ingestionpb.IngestionServiceClient
}

// NewDeleteDatasetHandler constructs a DeleteDatasetHandler.
func NewDeleteDatasetHandler(client ingestionpb.IngestionServiceClient) *DeleteDatasetHandler {
	return &DeleteDatasetHandler{ingestionClient: client}
}

// Handle processes the "delete_dataset" MCP tool call.
func (h *DeleteDatasetHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
	tenantID  := extractTenantFromContext(ctx)
	datasetID := getString(input, "dataset_id")
	dsName    := getString(input, "dataset_name")

	if datasetID == "" && dsName == "" {
		return nil, fmt.Errorf("dataset_id or dataset_name is required")
	}

	resp, err := h.ingestionClient.DeleteDataset(ctx, &ingestionpb.DeleteDatasetRequest{
		TenantId:    tenantID,
		DatasetId:   datasetID,
		DatasetName: dsName,
	})
	if err != nil { return nil, fmt.Errorf("delete dataset: %w", err) }
	return map[string]any{
		"deleted":    resp.Deleted,
		"dataset_id": resp.DatasetId,
		"message":    "Dataset and all associated data deleted",
	}, nil
}
