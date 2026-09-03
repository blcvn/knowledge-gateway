// Package grpc implements the console search handler for vnp-search-hub.
package grpc

import (
	"context"
	"log/slog"

	"vnp-memory/services/vnp-search-hub/usecase"
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

// ──── Dummy HTTP Router Methods ────────────────────────────────

// HandleSearch provides dummy response for POST /v1/console/memory/search
func (h *ConsoleSearchHandler) HandleSearch(_ context.Context) ([]byte, error) {
	return []byte(`{
		"results": [{
			"id": "mem-123", 
			"memoryType": "document",
			"title": "Dummy memory title",
			"engine": "zep",
			"score": 0.95,
			"summary": "This is a dummy search result summary.",
			"content": "Dummy search result full content",
			"entities": ["entity1", "entity2"]
		}], 
		"total": 1,
		"facets": {
			"byType": {
				"document": 1
			}
		}
	}`), nil
}

// HandleGetMemory provides dummy response for GET /v1/console/memory/{id}
func (h *ConsoleSearchHandler) HandleGetMemory(_ context.Context) ([]byte, error) {
	return []byte(`{"id": "mem-123", "content": "Dummy search result", "source": "zep", "timestamp": "2026-05-25T10:00:00Z"}`), nil
}

// HandleGetNeighbors provides dummy response for GET /v1/console/memory/{id}/neighbors
func (h *ConsoleSearchHandler) HandleGetNeighbors(_ context.Context) ([]byte, error) {
	return []byte(`{"nodes": [], "edges": []}`), nil
}

// HandleCreateTrace provides dummy response for POST /v1/console/debugger/trace
func (h *ConsoleSearchHandler) HandleCreateTrace(_ context.Context) ([]byte, error) {
	return []byte(`{"trace_id": "trace-999", "status": "simulated"}`), nil
}
