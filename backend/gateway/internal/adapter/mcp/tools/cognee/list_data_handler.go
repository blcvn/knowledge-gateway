package cogneetools

import (
	"context"
	"fmt"

	ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
)

// ListDataHandler implements the "list_data" MCP tool.
type ListDataHandler struct {
	ingestionClient ingestionpb.IngestionServiceClient
}

// NewListDataHandler constructs a ListDataHandler.
func NewListDataHandler(client ingestionpb.IngestionServiceClient) *ListDataHandler {
	return &ListDataHandler{ingestionClient: client}
}

// Handle processes the "list_data" MCP tool call.
// If dataset_id or dataset_name is provided, lists entries within that dataset.
// Otherwise, lists all datasets for the tenant.
func (h *ListDataHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
	tenantID  := extractTenantFromContext(ctx)
	datasetID := getString(input, "dataset_id")
	dsName    := getString(input, "dataset_name")
	limit     := getIntOrDefault(input, "limit", 20)
	offset    := getIntOrDefault(input, "offset", 0)

	if datasetID != "" || dsName != "" {
		resp, err := h.ingestionClient.ListDataEntries(ctx, &ingestionpb.ListDataEntriesRequest{
			TenantId:    tenantID,
			DatasetId:   datasetID,
			DatasetName: dsName,
			Limit:       int32(limit),
			Offset:      int32(offset),
		})
		if err != nil { return nil, fmt.Errorf("list data entries: %w", err) }
		return resp, nil
	}

	resp, err := h.ingestionClient.ListDatasets(ctx, &ingestionpb.ListDatasetsRequest{
		TenantId: tenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil { return nil, fmt.Errorf("list datasets: %w", err) }
	return resp, nil
}
