package service

import (
	"encoding/json"
	"net/http"

	"kgs-platform/internal/biz"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// GraphHTTPHandler exposes pure-HTTP endpoints for project-scoped graph queries.
// These are NOT generated from proto — they extend the Kratos HTTP server directly.
type GraphHTTPHandler struct {
	uc *biz.GraphUsecase
}

func NewGraphHTTPHandler(uc *biz.GraphUsecase) *GraphHTTPHandler {
	return &GraphHTTPHandler{uc: uc}
}

// RegisterGraphHTTPRoutes registers the extra HTTP routes on the Kratos HTTP server.
// Call this after RegisterGraphHTTPServer in server/http.go.
func RegisterGraphHTTPRoutes(srv *kratoshttp.Server, h *GraphHTTPHandler) {
	r := srv.Route("/")
	// GET /api/v1/graph?project_id=<id>  — full project graph (nodes + edges)
	r.GET("/api/v1/graph", h.GetProjectGraph)
	// GET /api/v1/graph/nodes?project_id=<id>  — nodes only
	r.GET("/api/v1/graph/nodes", h.GetProjectNodes)
	// GET /api/v1/graph/edges?project_id=<id>  — edges only
	r.GET("/api/v1/graph/edges", h.GetProjectEdges)
	// DELETE /api/v1/graph?project_id=<id>  — cascade-delete entire project KG
	r.DELETE("/api/v1/graph", h.DeleteProjectGraph)
}

// GetProjectGraph — GET /api/v1/graph?project_id=<id>
// Returns { data: { nodes: [...], edges: [...] } }
// Used by preview-backend KGSPlatformClient.GetGraph().
func (h *GraphHTTPHandler) GetProjectGraph(ctx kratoshttp.Context) error {
	projectID := ctx.Request().URL.Query().Get("project_id")
	if projectID == "" {
		return writeJSON(ctx.Response(), http.StatusBadRequest,
			map[string]any{"error": "project_id is required"})
	}

	nodes, edges, err := h.uc.GetProjectGraph(ctx.Request().Context(), projectID)
	if err != nil {
		return writeJSON(ctx.Response(), http.StatusInternalServerError,
			map[string]any{"error": err.Error()})
	}
	if nodes == nil {
		nodes = []map[string]any{}
	}
	if edges == nil {
		edges = []map[string]any{}
	}

	return writeJSON(ctx.Response(), http.StatusOK, map[string]any{
		"data": map[string]any{
			"nodes": nodes,
			"edges": edges,
		},
		"project_id": projectID,
	})
}

// GetProjectNodes — GET /api/v1/graph/nodes?project_id=<id>
func (h *GraphHTTPHandler) GetProjectNodes(ctx kratoshttp.Context) error {
	projectID := ctx.Request().URL.Query().Get("project_id")
	if projectID == "" {
		return writeJSON(ctx.Response(), http.StatusBadRequest,
			map[string]any{"error": "project_id is required"})
	}

	nodes, _, err := h.uc.GetProjectGraph(ctx.Request().Context(), projectID)
	if err != nil {
		return writeJSON(ctx.Response(), http.StatusInternalServerError,
			map[string]any{"error": err.Error()})
	}
	if nodes == nil {
		nodes = []map[string]any{}
	}
	return writeJSON(ctx.Response(), http.StatusOK, map[string]any{
		"data": nodes, "project_id": projectID,
	})
}

// GetProjectEdges — GET /api/v1/graph/edges?project_id=<id>
func (h *GraphHTTPHandler) GetProjectEdges(ctx kratoshttp.Context) error {
	projectID := ctx.Request().URL.Query().Get("project_id")
	if projectID == "" {
		return writeJSON(ctx.Response(), http.StatusBadRequest,
			map[string]any{"error": "project_id is required"})
	}

	_, edges, err := h.uc.GetProjectGraph(ctx.Request().Context(), projectID)
	if err != nil {
		return writeJSON(ctx.Response(), http.StatusInternalServerError,
			map[string]any{"error": err.Error()})
	}
	if edges == nil {
		edges = []map[string]any{}
	}
	return writeJSON(ctx.Response(), http.StatusOK, map[string]any{
		"data": edges, "project_id": projectID,
	})
}

// DeleteProjectGraph — DELETE /api/v1/graph?project_id=<id>
// Cascade-deletes all KG nodes and relationships for the project.
// Called by preview-backend during project deletion (SOL-011 T03).
func (h *GraphHTTPHandler) DeleteProjectGraph(ctx kratoshttp.Context) error {
	projectID := ctx.Request().URL.Query().Get("project_id")
	if projectID == "" {
		return writeJSON(ctx.Response(), http.StatusBadRequest,
			map[string]any{"error": "project_id is required"})
	}

	nodesDeleted, err := h.uc.DeleteProjectGraph(ctx.Request().Context(), projectID)
	if err != nil {
		return writeJSON(ctx.Response(), http.StatusInternalServerError,
			map[string]any{"error": err.Error()})
	}
	return writeJSON(ctx.Response(), http.StatusOK, map[string]any{
		"project_id":    projectID,
		"nodes_deleted": nodesDeleted,
		"status":        "deleted",
	})
}

// writeJSON writes a JSON response to the HTTP response writer.
func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
