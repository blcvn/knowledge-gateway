package grpc

import (
\t"context"
\t"errors"
\t"log/slog"

\tdomain "vnp-memory/services/zep-search/internal/domain/search"
\tusecase "vnp-memory/services/zep-search/internal/usecase/search"
)

// In a real scenario, this would be imported from the generated pb file:
// pb "vnp-memory/services/zep-search/api/proto/v1"

// Handler acts as the gRPC adapter for the Search Usecase.
type Handler struct {
\t// pb.UnimplementedZepSearchServiceServer
\tusecase *usecase.SearchUseCase
}

// NewHandler creates a new gRPC handler for zep-search.
func NewHandler(uc *usecase.SearchUseCase) *Handler {
\treturn &Handler{
\t\tusecase: uc,
\t}
}

// HybridSearch handles the incoming gRPC request for hybrid searching.
// The real signature would match the pb generated code:
// func (h *Handler) HybridSearch(ctx context.Context, req *pb.HybridSearchRequest) (*pb.HybridSearchResponse, error)
func (h *Handler) HybridSearch(ctx context.Context, queryText string, limit int) ([]domain.SearchResult, error) {
\tslog.Info("Received HybridSearch request", slog.String("query", queryText), slog.Int("limit", limit))

\tif queryText == "" {
\t\treturn nil, errors.New("query text cannot be empty")
\t}

\tquery := &domain.SearchQuery{
\t\tText:  queryText,
\t\tLimit: limit,
\t}

\tresults, err := h.usecase.HybridSearch(ctx, query)
\tif err != nil {
\t\tslog.Error("HybridSearch usecase failed", slog.String("error", err.Error()))
\t\treturn nil, err
\t}

\tslog.Info("HybridSearch completed successfully", slog.Int("result_count", len(results)))

\t// Map domain.SearchResult to pb.SearchResult here...
\t
\treturn results, nil
}
