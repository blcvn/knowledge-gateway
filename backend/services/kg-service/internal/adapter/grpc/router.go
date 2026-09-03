// Package grpc implements the ForwardService router for kg-service.
//
// Routes cover all graphiti-* and cognee-* endpoints.
// (MERGE-P2-T1 + MERGE-P2-T2)
package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"vnp-memory/shared/pkg/forward"
	uc_cognee "vnp-memory/services/kg-service/internal/usecase/cognee"
	uc_graphiti "vnp-memory/services/kg-service/internal/usecase/graphiti"

	"vnp-memory/services/kg-service/internal/domain/cognee"
	"vnp-memory/services/kg-service/internal/domain/graphiti"
)

// KGHandler handles all kg-service HTTP endpoints.
type KGHandler struct {
	ingest      *uc_graphiti.IngestUseCase
	store       *uc_graphiti.StoreUseCase
	search      *uc_graphiti.SearchUseCase
	knowledge   *uc_graphiti.KnowledgeUseCase
	dataset     *uc_cognee.DatasetUseCase
	cognify     *uc_cognee.CognifyUseCase
	csearch     *uc_cognee.CogneeSearchUseCase
	memify      *uc_cognee.MemifyUseCase       // CR-COGNEE-001
	nsSearch    *uc_cognee.NodeSetsSearchUseCase // CR-COGNEE-002
	datapoints  *uc_cognee.AddDataPointsUseCase  // CR-COGNEE-003
}

// NewKGHandler creates a KGHandler.
func NewKGHandler(
	ingest *uc_graphiti.IngestUseCase,
	store *uc_graphiti.StoreUseCase,
	search *uc_graphiti.SearchUseCase,
	knowledge *uc_graphiti.KnowledgeUseCase,
	dataset *uc_cognee.DatasetUseCase,
	cognify *uc_cognee.CognifyUseCase,
	csearch *uc_cognee.CogneeSearchUseCase,
	memify *uc_cognee.MemifyUseCase,
	nsSearch *uc_cognee.NodeSetsSearchUseCase,
	datapoints *uc_cognee.AddDataPointsUseCase,
) *KGHandler {
	return &KGHandler{
		ingest: ingest, store: store, search: search, knowledge: knowledge,
		dataset: dataset, cognify: cognify, csearch: csearch,
		memify: memify, nsSearch: nsSearch, datapoints: datapoints,
	}
}

// RegisterRoutes registers all kg-service routes on the ForwardService router.
func RegisterRoutes(router *forward.Router, h *KGHandler) {
	// ── Graphiti Episode Ingestion (graphiti-ingestion) ──
	router.Handle("POST", "/v1/graphiti/episodes", h.adaptHTTP(h.IngestEpisode))

	// ── Graphiti Store CRUD (graphiti-store) ──
	router.Handle("GET", "/v1/graphiti/nodes/*", h.adaptHTTP(h.GetNode))
	router.Handle("GET", "/v1/graphiti/edges/*", h.adaptHTTP(h.GetEdge))

	// ── Graphiti Search (graphiti-search) ──
	router.Handle("POST", "/v1/graphiti/search", h.adaptHTTP(h.Search))

	// ── Console Graph (graphiti-knowledge) ──
	router.Handle("POST", "/v1/console/graph/subgraph", h.adaptHTTP(h.Subgraph))
	router.Handle("GET", "/v1/console/graph/entity/*", h.adaptHTTP(h.GetEntity))
	router.Handle("POST", "/v1/console/graph/timeline", h.adaptHTTP(h.Timeline))
	router.Handle("GET", "/v1/console/graph/ontology", h.adaptHTTP(h.GetOntology))
	router.Handle("PUT", "/v1/console/graph/ontology", h.adaptHTTP(h.UpdateOntology))
	router.Handle("POST", "/v1/console/graph/query", h.adaptHTTP(h.QueryGraph))

	// ── Console Adaptive Memories (graphiti-knowledge) ──
	router.Handle("GET", "/v1/console/adaptive/memories", h.adaptHTTP(h.ListMemories))
	router.Handle("GET", "/v1/console/adaptive/memories/*/versions", h.adaptHTTP(h.GetVersions))

	// ── Cognee Dataset (cognee-ingestion / cognee-pipeline) ──
	router.Handle("POST", "/v1/cognee/datasets", h.adaptHTTP(h.CreateDataset))
	router.Handle("GET", "/v1/cognee/datasets", h.adaptHTTP(h.ListDatasets))
	router.Handle("POST", "/v1/cognee/datasets/*/data", h.adaptHTTP(h.UploadData))
	router.Handle("POST", "/v1/cognee/datasets/*/cognify", h.adaptHTTP(h.Cognify))

	// ── CR-COGNEE-001: Memify (non-destructive graph enrichment) ──
	router.Handle("POST", "/v1/cognee/datasets/*/memify", h.adaptHTTP(h.Memify))
	router.Handle("GET", "/v1/cognee/datasets/*/memify/status", h.adaptHTTP(h.GetMemifyStatus))

	// ── Cognee Search (cognee-search) ──
	router.Handle("POST", "/v1/cognee/search", h.adaptHTTP(h.CogneeSearch))

	// ── CR-COGNEE-003: DataPoints (schema-defined ingestion, zero LLM) ──
	router.Handle("POST", "/v1/cognee/datasets/*/datapoints", h.adaptHTTP(h.AddDataPoints))
}

// ─── Graphiti Handlers ──────────────────────────────────────────────────────

func (h *KGHandler) IngestEpisode(w http.ResponseWriter, r *http.Request) {
	var req graphiti.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TenantID == "" {
		req.TenantID = r.Header.Get("X-Tenant-ID")
	}
	ep, err := h.ingest.IngestEpisode(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ep)
}

func (h *KGHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	nodeUUID := r.PathValue("id")
	node, err := h.store.GetNode(r.Context(), tenantID, nodeUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (h *KGHandler) GetEdge(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	edgeUUID := r.PathValue("id")
	edge, err := h.store.GetEdge(r.Context(), tenantID, edgeUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, edge)
}

func (h *KGHandler) Search(w http.ResponseWriter, r *http.Request) {
	var q graphiti.SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if q.TenantID == "" {
		q.TenantID = r.Header.Get("X-Tenant-ID")
	}
	if q.Mode == "" {
		q.Mode = "hybrid"
	}
	result, err := h.search.Search(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *KGHandler) Subgraph(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query    string `json:"query"`
		TenantID string `json:"tenant_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.TenantID == "" {
		req.TenantID = r.Header.Get("X-Tenant-ID")
	}
	result, err := h.knowledge.QuerySubgraph(r.Context(), req.TenantID, req.Query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *KGHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	uuid := r.PathValue("id")
	node, err := h.store.GetNode(r.Context(), tenantID, uuid)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (h *KGHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct{ Query string `json:"query"` }
	_ = json.NewDecoder(r.Body).Decode(&req)
	result, err := h.knowledge.QuerySubgraph(r.Context(), tenantID, req.Query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"timeline": result.Nodes})
}

func (h *KGHandler) GetOntology(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	ont, err := h.knowledge.GetOntology(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ont)
}

func (h *KGHandler) UpdateOntology(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var ont graphiti.Ontology
	if err := json.NewDecoder(r.Body).Decode(&ont); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ont.TenantID = tenantID
	if err := h.knowledge.UpdateOntology(r.Context(), &ont); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *KGHandler) QueryGraph(w http.ResponseWriter, r *http.Request) {
	h.Subgraph(w, r)
}

func (h *KGHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	q := r.URL.Query().Get("query")
	result, _ := h.knowledge.QuerySubgraph(r.Context(), tenantID, q)
	writeJSON(w, http.StatusOK, map[string]any{"memories": result.Nodes, "total": len(result.Nodes)})
}

func (h *KGHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"versions": []any{}})
}

// ─── Cognee Handlers ────────────────────────────────────────────────────────

func (h *KGHandler) CreateDataset(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ds, err := h.dataset.CreateDataset(r.Context(), tenantID, req.Name)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ds)
}

func (h *KGHandler) ListDatasets(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	datasets, err := h.dataset.ListDatasets(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"datasets": datasets})
}

func (h *KGHandler) UploadData(w http.ResponseWriter, r *http.Request) {
	datasetID := r.PathValue("id")
	var item cognee.DataItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item.DatasetID = datasetID
	if err := h.dataset.UploadData(r.Context(), datasetID, &item); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"uploaded": true})
}

func (h *KGHandler) Cognify(w http.ResponseWriter, r *http.Request) {
	datasetID := r.PathValue("id")
	var cfg cognee.PipelineConfig
	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&cfg)
	}

	job, err := h.cognify.Cognify(r.Context(), datasetID, cfg)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (h *KGHandler) CogneeSearch(w http.ResponseWriter, r *http.Request) {
	// CR-COGNEE-002: Parse full SearchRequest to support NodeSets scoping.
	var req cognee.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	var results []*cognee.SearchResult
	var err error

	if len(req.NodeSets) > 0 {
		// CR-COGNEE-002: NodeSet-scoped search — filter by tag partitions.
		results, err = h.nsSearch.SearchWithNodeSets(r.Context(), req)
	} else {
		// Standard full-dataset search (unchanged behaviour, but passes full req).
		results, err = h.csearch.Search(r.Context(), req)
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "total": len(results)})
}

// ─── CR-COGNEE-001: Memify Handlers ──────────────────────────────────────────

// Memify handles POST /v1/cognee/datasets/{id}/memify.
// Returns 202 Accepted with pipeline_run_id for async polling.
func (h *KGHandler) Memify(w http.ResponseWriter, r *http.Request) {
	datasetID := r.PathValue("id")
	tenantID := r.Header.Get("X-Tenant-ID")

	var reqBody struct {
		Config *cognee.MemifyConfig `json:"config"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	cfg := cognee.MemifyConfig{
		DeriveFacts:   true,
		EmbedTriplets: true,
		BatchSize:     50,
	}
	if reqBody.Config != nil {
		if reqBody.Config.BatchSize > 0 {
			cfg.BatchSize = reqBody.Config.BatchSize
		}
		cfg.DeriveFacts = reqBody.Config.DeriveFacts
		cfg.EmbedTriplets = reqBody.Config.EmbedTriplets
	}

	job, err := h.memify.Memify(r.Context(), datasetID, tenantID, cfg)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

// GetMemifyStatus handles GET /v1/cognee/datasets/{id}/memify/status.
func (h *KGHandler) GetMemifyStatus(w http.ResponseWriter, r *http.Request) {
	datasetID := r.PathValue("id")
	pipelineRunID := r.URL.Query().Get("pipeline_run_id")

	job, err := h.memify.GetMemifyStatus(r.Context(), datasetID, pipelineRunID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// ─── CR-COGNEE-003: AddDataPoints Handler ─────────────────────────────────────

// AddDataPoints handles POST /v1/cognee/datasets/{id}/datapoints.
// Accepts schema-defined DataPoints and maps them directly to the knowledge graph
// without LLM entity extraction (zero token cost).
// Also supports CR-COGNEE-002 NodeSets tagging via the request payload.
func (h *KGHandler) AddDataPoints(w http.ResponseWriter, r *http.Request) {
	datasetID := r.PathValue("id")
	tenantID := r.Header.Get("X-Tenant-ID")

	var req cognee.AddDataPointsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.DatasetID = datasetID
	if req.TenantID == "" {
		req.TenantID = tenantID
	}

	resp, err := h.datapoints.AddDataPoints(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// ─── Adapter helper ─────────────────────────────────────────────────────────

// adaptHTTP converts http.HandlerFunc → forward.HandlerFunc.
func (h *KGHandler) adaptHTTP(hf http.HandlerFunc) forward.HandlerFunc {
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
