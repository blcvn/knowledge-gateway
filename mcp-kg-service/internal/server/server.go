package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/internal/service"
)

func New(port int, svc *service.Service, readTimeout, writeTimeout time.Duration) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	mux.HandleFunc("/v1/kg/documents", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			documents, err := svc.ListDocuments(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, documents)
		default:
			writeMethodNotAllowed(w, http.MethodGet)
		}
	})
	mux.HandleFunc("/v1/kg/features/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "q is required"})
			return
		}

		limit := 10
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 || parsed > 50 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be between 1 and 50"})
				return
			}
			limit = parsed
		}

		results, err := svc.SearchFeatures(r.Context(), query, r.URL.Query().Get("document_id"), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, results)
	})
	mux.HandleFunc("/v1/kg/documents/", func(w http.ResponseWriter, r *http.Request) {
		documentID, featureID, action, ok := parseDocumentAction(r.URL.Path)
		if !ok {
			writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
			return
		}

		switch action {
		case "subgraph":
			if r.Method != http.MethodGet {
				writeMethodNotAllowed(w, http.MethodGet)
				return
			}
			opts, err := parseSubgraphOptions(r)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}

			subgraph, err := svc.GetDocumentSubgraph(r.Context(), documentID, opts)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, subgraph)
		case "feature_subgraph":
			if r.Method != http.MethodGet {
				writeMethodNotAllowed(w, http.MethodGet)
				return
			}
			if featureID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "feature_id is required"})
				return
			}
			subgraph, err := svc.GetFeatureSubgraph(r.Context(), documentID, featureID)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, subgraph)
		case "feature_detail":
			if r.Method != http.MethodGet {
				writeMethodNotAllowed(w, http.MethodGet)
				return
			}
			if featureID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "feature_id is required"})
				return
			}
			detail, err := svc.GetFeatureDetail(r.Context(), documentID, featureID)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, detail)
		case "upsert_nodes":
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w, http.MethodPost)
				return
			}
			var payload struct {
				Nodes []service.UpsertNodePayload `json:"nodes"`
				Edges []service.UpsertEdgePayload `json:"edges"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
				return
			}
			upsertedNodes, upsertedEdges, err := svc.UpsertNodes(r.Context(), documentID, payload.Nodes, payload.Edges)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"document_id":    documentID,
				"upserted_nodes": upsertedNodes,
				"upserted_edges": upsertedEdges,
			})
		case "save_graph":
			if r.Method != http.MethodPut {
				writeMethodNotAllowed(w, http.MethodPut)
				return
			}
			var payload struct {
				Graph service.GraphSnapshot `json:"graph"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
				return
			}
			if err := svc.SaveGraph(r.Context(), documentID, payload.Graph); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"document_id": documentID,
				"saved":       true,
			})
		case "save_document":
			if r.Method != http.MethodPut {
				writeMethodNotAllowed(w, http.MethodPut)
				return
			}
			var payload struct {
				DocKind string `json:"doc_kind"`
				Content string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
				return
			}
			result, err := svc.SaveDocument(r.Context(), documentID, payload.DocKind, payload.Content)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, result)
		default:
			writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
		}
	})

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

func registerNotImplemented(mux *http.ServeMux, pattern string) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
	})
}

func parseDocumentAction(path string) (documentID, featureID, action string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/v1/kg/documents/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	switch {
	case len(parts) == 2 && parts[0] != "" && parts[1] == "subgraph":
		return parts[0], "", "subgraph", true
	case len(parts) == 2 && parts[0] != "" && parts[1] == "nodes":
		return parts[0], "", "upsert_nodes", true
	case len(parts) == 2 && parts[0] != "" && parts[1] == "graph":
		return parts[0], "", "save_graph", true
	case len(parts) == 2 && parts[0] != "" && parts[1] == "document":
		return parts[0], "", "save_document", true
	case len(parts) == 4 && parts[0] != "" && parts[1] == "feature" && parts[2] != "" && parts[3] == "subgraph":
		return parts[0], parts[2], "feature_subgraph", true
	case len(parts) == 4 && parts[0] != "" && parts[1] == "feature" && parts[2] != "" && parts[3] == "detail":
		return parts[0], parts[2], "feature_detail", true
	default:
		return "", "", "", false
	}
}

func parseSubgraphOptions(r *http.Request) (service.DocumentSubgraphOptions, error) {
	query := r.URL.Query()

	maxNodes := 200
	if raw := strings.TrimSpace(query.Get("max_nodes")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return service.DocumentSubgraphOptions{}, fmt.Errorf("invalid max_nodes")
		}
		if parsed < 1 || parsed > 500 {
			return service.DocumentSubgraphOptions{}, fmt.Errorf("max_nodes must be between 1 and 500")
		}
		maxNodes = parsed
	}

	includeEdges := true
	if raw := strings.TrimSpace(query.Get("include_edges")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return service.DocumentSubgraphOptions{}, fmt.Errorf("invalid include_edges")
		}
		includeEdges = parsed
	}

	nodeTypes := query["node_types"]
	if len(nodeTypes) == 1 && strings.Contains(nodeTypes[0], ",") {
		nodeTypes = strings.Split(nodeTypes[0], ",")
	}
	for i := range nodeTypes {
		nodeTypes[i] = strings.TrimSpace(nodeTypes[i])
	}

	return service.DocumentSubgraphOptions{
		MaxNodes:     maxNodes,
		NodeTypes:    compact(nodeTypes),
		IncludeEdges: includeEdges,
	}, nil
}

func compact(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func writeMethodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrFeatureNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
