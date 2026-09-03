package cogneetools

import (
	"context"
	"fmt"
	"strings"

	ingestionpb "github.com/vnp-memory/api/proto/cognee/ingestion/v1"
	cognifypb   "github.com/vnp-memory/api/proto/cognee/cognify/v1"
)

// CognifyHandler implements the "cognify" MCP tool: ingest + trigger pipeline.
type CognifyHandler struct {
	ingestionClient ingestionpb.IngestionServiceClient
	cognifyClient   cognifypb.CognifyServiceClient
}

// NewCognifyHandler constructs a CognifyHandler.
func NewCognifyHandler(ing ingestionpb.IngestionServiceClient, cog cognifypb.CognifyServiceClient) *CognifyHandler {
	return &CognifyHandler{ingestionClient: ing, cognifyClient: cog}
}

// Handle processes the "cognify" MCP tool call.
func (h *CognifyHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
	tenantID := extractTenantFromContext(ctx)
	data     := getString(input, "data")
	dsName   := getStringOrDefault(input, "dataset_name", "default")
	nodeSets := toStringSlice(input["node_sets"])
	template := getStringOrDefault(input, "template", "STANDARD")

	if data == "" { return nil, fmt.Errorf("data is required") }

	// Step 1: AddData → cognee-ingestion
	addResp, err := h.ingestionClient.AddData(ctx, &ingestionpb.AddDataRequest{
		TenantId:    tenantID,
		DatasetName: dsName,
		Items: []*ingestionpb.DataItem{{
			Content:     data,
			ContentType: detectContentType(data),
		}},
		NodeSets: nodeSets,
	})
	if err != nil { return nil, fmt.Errorf("ingestion failed: %w", err) }

	// Step 2: StartCognify → cognee-cognify (returns quickly, runs in background)
	cognifyResp, err := h.cognifyClient.StartCognify(ctx, &cognifypb.StartCognifyRequest{
		DatasetId: addResp.DatasetId,
		TenantId:  tenantID,
		EntryIds:  addResp.EntryIds,
		NodeSets:  nodeSets,
		Template:  template,
	})
	if err != nil {
		// Partial success: ingestion succeeded, cognify queued but may have failed
		return map[string]any{
			"dataset_id": addResp.DatasetId,
			"status":     "INGESTED_NOT_PROCESSED",
			"error":      err.Error(),
		}, nil
	}

	return map[string]any{
		"dataset_id":      addResp.DatasetId,
		"pipeline_run_id": cognifyResp.PipelineRunId,
		"status":          cognifyResp.Status,
		"message":         fmt.Sprintf("Data ingested. Processing with template '%s' in background.", template),
	}, nil
}

// detectContentType infers content type from the data string.
func detectContentType(data string) string {
	if strings.HasPrefix(data, "http://") || strings.HasPrefix(data, "https://") { return "URL" }
	return "TEXT"
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

// extractTenantFromContext reads tenant_id from context.
func extractTenantFromContext(ctx context.Context) string {
	if id, ok := ctx.Value("tenant_id").(string); ok && id != "" { return id }
	return "default"
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok { return v }
	return ""
}

func getStringOrDefault(m map[string]any, key, def string) string {
	if v := getString(m, key); v != "" { return v }
	return def
}

func getIntOrDefault(m map[string]any, key string, def int) int {
	if v, ok := m[key].(float64); ok { return int(v) }
	return def
}

func toStringSlice(v any) []string {
	if v == nil { return nil }
	arr, ok := v.([]any)
	if !ok { return nil }
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok { result = append(result, s) }
	}
	return result
}

func getFloat64Ptr(m map[string]any, key string) *float64 {
	if v, ok := m[key].(float64); ok { return &v }
	return nil
}
