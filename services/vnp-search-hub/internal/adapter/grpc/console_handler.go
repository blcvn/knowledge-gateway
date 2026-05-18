// Package grpc implements the console search handler for vnp-search-hub.
package grpc

import (
	"context"
	"log/slog"

	"github.com/vnp-community/vnp-memory/services/vnp-search-hub/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConsoleSearchHandler provides the unified cross-engine search endpoint.
type ConsoleSearchHandler struct {
	orchestrator *usecase.SearchOrchestrator
	logger       *slog.Logger
}

// NewConsoleSearchHandler creates a console search handler.
func NewConsoleSearchHandler(orch *usecase.SearchOrchestrator, logger *slog.Logger) *ConsoleSearchHandler {
	return &ConsoleSearchHandler{orchestrator: orch, logger: logger}
}

// UnifiedSearch handles POST /api/v1/search — fan-out to selected engines.
func (h *ConsoleSearchHandler) UnifiedSearch(ctx context.Context, req usecase.SearchRequest) (*usecase.SearchResponse, error) {
	if req.Query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "query must not be empty")
	}

	resp, err := h.orchestrator.Search(ctx, req)
	if err != nil {
		h.logger.Error("unified search failed", "error", err, "query", req.Query)
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	h.logger.Info("unified search completed",
		"query", req.Query,
		"engines", len(req.Engines),
		"results", resp.Total,
		"latency_ms", resp.LatencyMs,
	)

	return resp, nil
}
