// Package grpc implements the ForwardService router for search-service.
//
// Routes cover: cross-engine search, RAG, connectors, MCP tools
// (MERGE-P2-T4)
package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"vnp-memory/shared/pkg/forward"
	"vnp-memory/services/search-service/internal/domain/search"
	ucconn "vnp-memory/services/search-service/internal/usecase/connector"
	ucorch "vnp-memory/services/search-service/internal/usecase/orchestrator"
)

// SearchHandler handles all search-service HTTP endpoints.
type SearchHandler struct {
	search    *ucorch.SearchOrchestrator
	connector *ucconn.Service
}

// NewSearchHandler creates a SearchHandler.
func NewSearchHandler(search *ucorch.SearchOrchestrator, connector *ucconn.Service) *SearchHandler {
	return &SearchHandler{search: search, connector: connector}
}

// RegisterRoutes registers all search-service routes.
func RegisterRoutes(router *forward.Router, h *SearchHandler) {
	// ── Unified Search (vnp-search-hub + ov-search + sm-search) ──
	router.Handle("POST", "/v1/search", h.adapt(h.Search))
	router.Handle("POST", "/v1/search/rag", h.adapt(h.RAG))
	router.Handle("POST", "/v1/search/agents", h.adapt(h.AgentSearch))

	// ── OV Console Search (ov-search) ──
	router.Handle("GET", "/v1/ov/search", h.adapt(h.OVSearch))

	// ── SM Search (sm-search) ──
	router.Handle("POST", "/v1/sm/search", h.adapt(h.Search))

	// ── Connectors (sm-connector) ──
	router.Handle("GET", "/v1/connectors", h.adapt(h.ListConnectors))
	router.Handle("POST", "/v1/connectors", h.adapt(h.CreateConnector))
	router.Handle("POST", "/v1/connectors/*/sync", h.adapt(h.SyncConnector))

	// ── MCP Tools (sm-mcp) ──
	router.Handle("GET", "/v1/mcp/tools", h.adapt(h.ListMCPTools))
	router.Handle("POST", "/v1/mcp/search", h.adapt(h.MCPSearch))
}

// ─── Search Handlers ────────────────────────────────────────────────────────

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var q search.Query
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if q.TenantID == "" {
		q.TenantID = tenantID
	}
	result, err := h.search.Search(r.Context(), &q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *SearchHandler) RAG(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	result, err := h.search.RAG(r.Context(), req.Query, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *SearchHandler) AgentSearch(w http.ResponseWriter, r *http.Request) {
	h.Search(w, r)
}

func (h *SearchHandler) OVSearch(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	q := &search.Query{
		Query:    r.URL.Query().Get("q"),
		TenantID: tenantID,
		Engines:  []string{"graphiti", "memobase"},
		Limit:    20,
	}
	result, err := h.search.Search(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ─── Connector Handlers ──────────────────────────────────────────────────────

func (h *SearchHandler) ListConnectors(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	connectors, err := h.connector.ListConnectors(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connectors": connectors})
}

func (h *SearchHandler) CreateConnector(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Name          string         `json:"name"`
		Type          string         `json:"type"`
		SyncFrequency string         `json:"sync_frequency"`
		Config        map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	conn, err := h.connector.CreateConnector(r.Context(), tenantID, req.Name, req.Type, req.SyncFrequency, req.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, conn)
}

func (h *SearchHandler) SyncConnector(w http.ResponseWriter, r *http.Request) {
	connectorID := r.PathValue("id")
	job, err := h.connector.SyncConnector(r.Context(), connectorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

// ─── MCP Tool Handlers (sm-mcp) ─────────────────────────────────────────────

var mcpTools = []map[string]any{
	{"name": "search_memory", "description": "Search across all memory engines", "input_schema": map[string]any{
		"type": "object", "required": []string{"query"},
		"properties": map[string]any{
			"query":   map[string]any{"type": "string"},
			"engines": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"limit":   map[string]any{"type": "integer", "default": 10},
		},
	}},
	{"name": "recall_context", "description": "Retrieve RAG context for a query", "input_schema": map[string]any{
		"type": "object", "required": []string{"query"},
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}},
}

func (h *SearchHandler) ListMCPTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tools": mcpTools})
}

func (h *SearchHandler) MCPSearch(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Tool  string         `json:"tool"`
		Input map[string]any `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	query, _ := req.Input["query"].(string)
	switch req.Tool {
	case "recall_context":
		result, err := h.search.RAG(r.Context(), query, tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	default: // search_memory
		limit := 10
		if l, ok := req.Input["limit"].(float64); ok {
			limit = int(l)
		}
		result, err := h.search.Search(r.Context(), &search.Query{
			Query:    query,
			TenantID: tenantID,
			Limit:    limit,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

// ─── adapt helper ─────────────────────────────────────────────────────────

func (h *SearchHandler) adapt(hf http.HandlerFunc) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		method, path := "GET", "/"
		if m, ok := params["__method"]; ok {
			method = m
		}
		if p, ok := params["__path"]; ok {
			path = p
		}
		u, _ := url.Parse(path)
		req, _ := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range params {
			if k[0] != '_' {
				req.SetPathValue(k, v)
			}
		}
		rw := &responseCapture{header: make(http.Header)}
		hf(rw, req)
		if rw.code >= 500 {
			return rw.body.Bytes(), fmt.Errorf("HTTP %d", rw.code)
		}
		return rw.body.Bytes(), nil
	}
}

type responseCapture struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func (rc *responseCapture) Header() http.Header        { return rc.header }
func (rc *responseCapture) WriteHeader(code int)        { rc.code = code }
func (rc *responseCapture) Write(b []byte) (int, error) { return rc.body.Write(b) }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
