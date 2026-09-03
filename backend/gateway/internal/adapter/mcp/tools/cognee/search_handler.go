package cogneetools

import (
	"context"
	"fmt"

	searchpb "github.com/vnp-memory/api/proto/cognee/search/v1"
)

// SearchHandler implements the "search" MCP tool.
type SearchHandler struct {
	searchClient searchpb.SearchServiceClient
}

// NewSearchHandler constructs a SearchHandler.
func NewSearchHandler(client searchpb.SearchServiceClient) *SearchHandler {
	return &SearchHandler{searchClient: client}
}

// Handle processes the "search" MCP tool call.
func (h *SearchHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
	tenantID  := extractTenantFromContext(ctx)
	query     := getString(input, "query")
	queryType := getStringOrDefault(input, "query_type", "GRAPH_COMPLETION")
	dsName    := getString(input, "dataset_name")
	nodeSets  := toStringSlice(input["node_sets"])
	topK      := getIntOrDefault(input, "top_k", 10)
	saveInter := false
	if v, ok := input["save_interaction"].(bool); ok { saveInter = v }

	if query == "" { return nil, fmt.Errorf("query is required") }

	req := &searchpb.SearchRequest{
		Query:           query,
		Strategies:      []string{queryType},
		DatasetName:     dsName,
		TenantId:        tenantID,
		NodeSets:        nodeSets,
		TopK:            int32(topK),
		SaveInteraction: saveInter,
	}

	resp, err := h.searchClient.Search(ctx, req)
	if err != nil { return nil, fmt.Errorf("search failed: %w", err) }
	return resp, nil
}
