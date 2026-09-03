// Package grpc implements the ForwardService router for pipeline-service (server binary).
//
// Routes cover pipeline status, jobs, queues, workers, templates, and knowledge CRUD.
// (MERGE-P3-T1)
package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"vnp-memory/shared/pkg/forward"
	domp "vnp-memory/services/pipeline-service/internal/domain/pipeline"
	domk "vnp-memory/services/pipeline-service/internal/domain/knowledge"
	uck "vnp-memory/services/pipeline-service/internal/usecase/knowledge"
	ucp "vnp-memory/services/pipeline-service/internal/usecase/pipeline"
)

// PipelineHandler handles all pipeline-service HTTP endpoints.
type PipelineHandler struct {
	pipeline  *ucp.PipelineUseCase
	knowledge *uck.KnowledgeUseCase
	index     *uck.IndexUseCase
}

// NewPipelineHandler creates a PipelineHandler.
func NewPipelineHandler(pipeline *ucp.PipelineUseCase, knowledge *uck.KnowledgeUseCase, index *uck.IndexUseCase) *PipelineHandler {
	return &PipelineHandler{pipeline: pipeline, knowledge: knowledge, index: index}
}

// RegisterRoutes registers all pipeline-service routes.
func RegisterRoutes(router *forward.Router, h *PipelineHandler) {
	// ── Pipeline (from vnp-pipelines) ──
	router.Handle("GET", "/v1/console/pipelines/status", h.adapt(h.Status))
	router.Handle("GET", "/v1/console/pipelines/queues", h.adapt(h.Queues))
	router.Handle("GET", "/v1/console/pipelines/workers", h.adapt(h.Workers))
	router.Handle("GET", "/v1/console/pipelines/templates", h.adapt(h.Templates))
	router.Handle("POST", "/v1/console/pipelines/*/jobs", h.adapt(h.EnqueueJob))
	router.Handle("GET", "/v1/console/pipelines/*", h.adapt(h.GetEngine))
	router.Handle("GET", "/v1/console/pipelines/*/jobs", h.adapt(h.ListJobs))
	router.Handle("GET", "/v1/console/pipelines/*/jobs/*", h.adapt(h.GetJob))

	// ── Knowledge CRUD (from ba-knowledge-service) ──
	router.Handle("POST", "/v1/knowledge/prds", h.adapt(h.CreatePRD))
	router.Handle("GET", "/v1/knowledge/prds", h.adapt(h.ListPRDs))
	router.Handle("GET", "/v1/knowledge/prds/*", h.adapt(h.GetPRD))
	router.Handle("GET", "/v1/knowledge/prds/*/outline", h.adapt(h.GetOutline))
}

// ─── Pipeline Handlers ──────────────────────────────────────────────────────

func (h *PipelineHandler) Status(w http.ResponseWriter, r *http.Request) {
	pipelines, err := h.pipeline.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pipelines": pipelines})
}

func (h *PipelineHandler) GetEngine(w http.ResponseWriter, r *http.Request) {
	engine := r.PathValue("id")
	p, err := h.pipeline.GetEngine(r.Context(), engine)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *PipelineHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	engine := r.PathValue("id")
	q := r.URL.Query()
	filter := domp.JobFilter{
		Status: q.Get("status"),
		Type:   q.Get("type"),
		Limit:  20,
	}
	jobs, total, err := h.pipeline.ListJobs(r.Context(), engine, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs, "total": total})
}

func (h *PipelineHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	engine := r.PathValue("engine")
	jobID := r.PathValue("id")
	job, err := h.pipeline.GetJob(r.Context(), engine, jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *PipelineHandler) Queues(w http.ResponseWriter, r *http.Request) {
	queues, err := h.pipeline.Queues(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"queues": queues})
}

func (h *PipelineHandler) Workers(w http.ResponseWriter, r *http.Request) {
	workers, err := h.pipeline.Workers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workers": workers})
}

func (h *PipelineHandler) Templates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.pipeline.Templates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func (h *PipelineHandler) EnqueueJob(w http.ResponseWriter, r *http.Request) {
	engine := r.PathValue("id")
	var req struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	job, err := h.pipeline.EnqueueJob(r.Context(), engine, req.Type, req.Payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

// ─── Knowledge Handlers ─────────────────────────────────────────────────────

func (h *PipelineHandler) CreatePRD(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	prd, err := h.knowledge.CreatePRD(r.Context(), tenantID, req.Title, req.Content, req.Tags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, prd)
}

func (h *PipelineHandler) ListPRDs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	prds, total, err := h.knowledge.ListPRDs(r.Context(), tenantID, 20, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"prds": prds, "total": total})
}

func (h *PipelineHandler) GetPRD(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	prd, err := h.knowledge.GetPRD(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prd)
}

func (h *PipelineHandler) GetOutline(w http.ResponseWriter, r *http.Request) {
	prdID := r.PathValue("id")
	outline, err := h.knowledge.GetOutline(r.Context(), prdID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outline)
}

// ── suppress unused import warning ─────────────────────────────────────────
var _ = domk.PRD{}

// ─── adapt helper ───────────────────────────────────────────────────────────

func (h *PipelineHandler) adapt(hf http.HandlerFunc) forward.HandlerFunc {
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
			if len(k) > 0 && k[0] != '_' {
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
