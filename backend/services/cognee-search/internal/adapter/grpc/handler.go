package grpc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	searchpb "github.com/vnp-memory/api/proto/cognee/search/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"vnp-memory/services/cognee-search/internal/usecase"
)

// SearchHandler implements the gRPC SearchService.
type SearchHandler struct {
	searchpb.UnimplementedSearchServiceServer
	searchUC *usecase.SearchUseCase
	logger   *slog.Logger
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(searchUC *usecase.SearchUseCase, logger *slog.Logger) *SearchHandler {
	return &SearchHandler{searchUC: searchUC, logger: logger}
}

// Search handles a SearchRequest from the gRPC client, propagating NodeSets to all retrievers.
func (h *SearchHandler) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	strategies := mapStrategiesToDomain(req.Strategies)

	var datasetID *uuid.UUID
	if req.DatasetId != "" {
		id, err := uuid.Parse(req.DatasetId)
		if err != nil { return nil, status.Errorf(codes.InvalidArgument, "invalid dataset_id: %v", err) }
		datasetID = &id
	}

	result, err := h.searchUC.Execute(ctx, usecase.SearchRequest{
		Query:           req.Query,
		Strategies:      strategies,
		DatasetID:       datasetID,
		DatasetName:     req.DatasetName,
		TenantID:        req.TenantId,
		NodeSets:        req.NodeSets,         // [NEW] propagate node_sets to all retrievers
		TopK:            int(req.TopK),
		SaveInteraction: req.SaveInteraction,
	})
	if err != nil { return nil, status.Errorf(codes.Internal, "search: %v", err) }

	return toProtoResponse(result), nil
}

// ListInteractions lists stored interactions for feedback tracking.
func (h *SearchHandler) ListInteractions(ctx context.Context, req *searchpb.ListInteractionsRequest) (*searchpb.ListInteractionsResponse, error) {
	// Stub — real implementation queries interaction store
	return &searchpb.ListInteractionsResponse{}, nil
}

// mapStrategiesToDomain converts proto strategy strings to domain SearchStrategy values.
func mapStrategiesToDomain(strategies []string) []usecase.SearchStrategy {
	result := make([]usecase.SearchStrategy, 0, len(strategies))
	for _, s := range strategies {
		result = append(result, usecase.SearchStrategy(s))
	}
	if len(result) == 0 {
		result = append(result, usecase.StrategyGraphCompletion)
	}
	return result
}

// toProtoResponse converts a domain SearchResponse to a proto SearchResponse.
func toProtoResponse(resp *usecase.SearchResponse) *searchpb.SearchResponse {
	if resp == nil { return &searchpb.SearchResponse{} }
	results := make([]*searchpb.SearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		results = append(results, &searchpb.SearchResult{
			Id:      r.ID,
			Content: r.Content,
			Score:   r.Score,
		})
	}
	pbResp := &searchpb.SearchResponse{Results: results}
	if resp.InteractionID != nil {
		pbResp.InteractionId = resp.InteractionID
	}
	return pbResp
}
