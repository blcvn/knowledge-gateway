// Package grpc implements the gRPC handler for vnp-search-hub.
package grpc

import (
	"context"

	"github.com/google/uuid"
	"vnp-memory/services/vnp-search-hub/domain/model"
	"vnp-memory/services/vnp-search-hub/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SearchHubHandler implements VnpSearchHubService gRPC.
type SearchHubHandler struct {
	recall *usecase.RecallService
}

func NewSearchHubHandler(recall *usecase.RecallService) *SearchHubHandler {
	return &SearchHubHandler{recall: recall}
}

// Recall implements VnpSearchHubService.Recall.
func (h *SearchHubHandler) Recall(ctx context.Context, query string, tenantIDStr string, scope string, maxResults int, rerankStrategy string, tokenBudget int) (*model.RecallResponse, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id")
	}

	req := &model.RecallRequest{
		Query:          query,
		TenantID:       tenantID,
		Scope:          model.RecallScope(scope),
		MaxResults:     maxResults,
		RerankStrategy: model.RerankStrategy(rerankStrategy),
		TokenBudget:    tokenBudget,
	}

	if req.Scope == "" {
		req.Scope = model.ScopeAll
	}
	if req.RerankStrategy == "" {
		req.RerankStrategy = model.RerankRRF
	}
	if req.TokenBudget <= 0 {
		req.TokenBudget = 4096
	}

	resp, err := h.recall.Recall(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "recall failed: %v", err)
	}
	return resp, nil
}
