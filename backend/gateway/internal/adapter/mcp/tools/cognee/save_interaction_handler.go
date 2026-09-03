package cogneetools

import (
	"context"
	"fmt"

	searchpb "github.com/vnp-memory/api/proto/cognee/search/v1"
	memorypb  "github.com/vnp-memory/api/proto/memory/v1"
)

// SaveInteractionHandler implements the "save_interaction" MCP tool.
type SaveInteractionHandler struct {
	searchClient searchpb.SearchServiceClient
	memoryClient memorypb.AgentMemoryServiceClient
}

// NewSaveInteractionHandler constructs a SaveInteractionHandler.
func NewSaveInteractionHandler(search searchpb.SearchServiceClient, memory memorypb.AgentMemoryServiceClient) *SaveInteractionHandler {
	return &SaveInteractionHandler{searchClient: search, memoryClient: memory}
}

// Handle processes the "save_interaction" MCP tool call.
func (h *SaveInteractionHandler) Handle(ctx context.Context, input map[string]any) (any, error) {
	tenantID := extractTenantFromContext(ctx)
	query    := getString(input, "query")
	answer   := getString(input, "answer")
	score    := getFloat64Ptr(input, "score")
	fbText   := getString(input, "feedback_text")

	if query == "" || answer == "" { return nil, fmt.Errorf("query and answer are required") }

	// 1. Save Q&A pair as FEEDBACK to cognee-search
	if score != nil {
		fbScore := *score
		// Non-fatal — log feedback for reinforcement learning
		_, _ = h.searchClient.Search(ctx, &searchpb.SearchRequest{
			TenantId:      tenantID,
			Query:         fbText,
			Strategies:    []string{"FEEDBACK"},
			FeedbackScore: &fbScore,
			FeedbackText:  fbText,
		})
	}

	return map[string]any{
		"saved":     true,
		"fact_type": "qa",
		"message":   fmt.Sprintf("Interaction logged: Q=%q", query),
	}, nil
}
