// Package grpc implements the ForwardService router for obs-service.
//
// Routes cover: metrics, traces, errors, costs, topology, services,
//               databases, resources, deployments, engine status.
// (MERGE-P3-T2)
package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"vnp-memory/shared/pkg/forward"
	domobs "vnp-memory/services/obs-service/internal/domain/observability"
	ucinfra "vnp-memory/services/obs-service/internal/usecase/infra"
	ucobs "vnp-memory/services/obs-service/internal/usecase/observability"
)

// ObsHandler handles all obs-service HTTP endpoints.
type ObsHandler struct {
	obs   *ucobs.ObservabilityService
	infra *ucinfra.InfraService
}

// NewObsHandler creates an ObsHandler.
func NewObsHandler(obs *ucobs.ObservabilityService, infra *ucinfra.InfraService) *ObsHandler {
	return &ObsHandler{obs: obs, infra: infra}
}

// RegisterRoutes registers all obs-service routes.
func RegisterRoutes(router *forward.Router, h *ObsHandler) {
	// ── Observability (vnp-observability + sm-engine) ──
	router.Handle("GET", "/v1/console/observability/metrics", h.adapt(h.Metrics))
	router.Handle("GET", "/v1/console/observability/traces", h.adapt(h.ListTraces))
	router.Handle("GET", "/v1/console/observability/traces/*", h.adapt(h.GetTrace))
	router.Handle("GET", "/v1/console/observability/errors", h.adapt(h.Errors))
	router.Handle("GET", "/v1/console/observability/costs", h.adapt(h.Costs))
	router.Handle("GET", "/v1/console/observability/engines", h.adapt(h.EngineStatus))

	// ── Infrastructure (vnp-infra) ──
	router.Handle("GET", "/v1/console/infra/topology", h.adapt(h.Topology))
	router.Handle("GET", "/v1/console/infra/services", h.adapt(h.ListServices))
	router.Handle("GET", "/v1/console/infra/services/*", h.adapt(h.GetService))
	router.Handle("GET", "/v1/console/infra/databases", h.adapt(h.Databases))
	router.Handle("GET", "/v1/console/infra/resources", h.adapt(h.Resources))
	router.Handle("GET", "/v1/console/infra/deployments", h.adapt(h.Deployments))
}

// ─── Observability Handlers ─────────────────────────────────────────────────

func (h *ObsHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	summary, err := h.obs.Metrics(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *ObsHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domobs.TraceFilter{
		Service: q.Get("service"),
		Status:  q.Get("status"),
		Limit:   20,
	}
	traces, total, err := h.obs.ListTraces(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"traces": traces, "total": total})
}

func (h *ObsHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := r.PathValue("id")
	trace, err := h.obs.GetTrace(r.Context(), traceID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

func (h *ObsHandler) Errors(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domobs.ErrorFilter{
		Service: q.Get("service"),
		Limit:   50,
	}
	errors, total, err := h.obs.Errors(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"errors": errors, "total": total})
}

func (h *ObsHandler) Costs(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	costs, err := h.obs.Costs(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"costs": costs})
}

func (h *ObsHandler) EngineStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.obs.EngineStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// ─── Infrastructure Handlers ─────────────────────────────────────────────────

func (h *ObsHandler) Topology(w http.ResponseWriter, r *http.Request) {
	graph, err := h.infra.Topology(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (h *ObsHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.infra.ListServices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": services})
}

func (h *ObsHandler) GetService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("id")
	svc, err := h.infra.GetService(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ObsHandler) Databases(w http.ResponseWriter, r *http.Request) {
	dbs, err := h.infra.Databases(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"databases": dbs})
}

func (h *ObsHandler) Resources(w http.ResponseWriter, r *http.Request) {
	resources, err := h.infra.Resources(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resources": resources})
}

func (h *ObsHandler) Deployments(w http.ResponseWriter, r *http.Request) {
	deps, err := h.infra.Deployments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deployments": deps})
}

// ─── adapt helper ────────────────────────────────────────────────────────────

func (h *ObsHandler) adapt(hf http.HandlerFunc) forward.HandlerFunc {
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
