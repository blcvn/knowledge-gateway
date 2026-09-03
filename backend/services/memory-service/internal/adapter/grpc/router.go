// Package grpc implements the ForwardService router for memory-service.
//
// Routes cover all memobase, zep, and supermemory endpoints.
// (MERGE-P2-T3)
package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"vnp-memory/shared/pkg/forward"
	"vnp-memory/services/memory-service/internal/domain/memobase"
	"vnp-memory/services/memory-service/internal/domain/sm"
	"vnp-memory/services/memory-service/internal/domain/zep"
	ucmb "vnp-memory/services/memory-service/internal/usecase/memobase"
	ucsm "vnp-memory/services/memory-service/internal/usecase/sm"
	uczep "vnp-memory/services/memory-service/internal/usecase/zep"
)

// MemoryHandler handles all memory-service HTTP endpoints.
type MemoryHandler struct {
	mbIngest  *ucmb.IngestUseCase
	mbContext *ucmb.ContextUseCase
	zepUser   *uczep.UserUseCase
	zepMem    *uczep.MemoryUseCase
	zepGraph  *uczep.GraphUseCase
	smMemory  *ucsm.MemoryUseCase
	smDoc     *ucsm.DocumentUseCase
}

// NewMemoryHandler creates a MemoryHandler.
func NewMemoryHandler(
	mbIngest *ucmb.IngestUseCase, mbCtx *ucmb.ContextUseCase,
	zUser *uczep.UserUseCase, zMem *uczep.MemoryUseCase, zGraph *uczep.GraphUseCase,
	smMem *ucsm.MemoryUseCase, smDoc *ucsm.DocumentUseCase,
) *MemoryHandler {
	return &MemoryHandler{
		mbIngest: mbIngest, mbContext: mbCtx,
		zepUser: zUser, zepMem: zMem, zepGraph: zGraph,
		smMemory: smMem, smDoc: smDoc,
	}
}

// RegisterRoutes registers all memory-service routes on the ForwardService router.
func RegisterRoutes(router *forward.Router, h *MemoryHandler) {
	// ── Memobase (memobase-ingestion, memobase-context, memobase-engine) ──
	router.Handle("POST", "/v1/memobase/users/*/blobs", h.adapt(h.InsertBlob))
	router.Handle("POST", "/v1/memobase/users/*/flush", h.adapt(h.Flush))
	router.Handle("GET", "/v1/memobase/users/*/context", h.adapt(h.GetContext))
	router.Handle("GET", "/v1/memobase/users/*/profiles", h.adapt(h.GetProfiles))
	router.Handle("GET", "/v1/memobase/users/*/events", h.adapt(h.GetEvents))

	// ── Zep (zep-user, zep-thread, zep-memory, zep-search, zep-graph) ──
	router.Handle("POST", "/v1/zep/users", h.adapt(h.CreateZepUser))
	router.Handle("GET", "/v1/zep/users/*", h.adapt(h.GetZepUser))
	router.Handle("PATCH", "/v1/zep/users/*", h.adapt(h.UpdateZepUser))
	router.Handle("POST", "/v1/zep/sessions/*/memory", h.adapt(h.PutMemory))
	router.Handle("GET", "/v1/zep/sessions/*/memory", h.adapt(h.GetMemory))
	router.Handle("POST", "/v1/zep/sessions/*/search", h.adapt(h.SessionSearch))
	router.Handle("POST", "/v1/zep/graph/search", h.adapt(h.GraphSearch))
	router.Handle("POST", "/v1/zep/graph/facts", h.adapt(h.AddFact))

	// ── Supermemory (sm-memory, sm-document, sm-profile) ──
	router.Handle("POST", "/v1/sm/memories", h.adapt(h.CreateSMMemory))
	router.Handle("POST", "/v1/sm/rag", h.adapt(h.SMRag))
	router.Handle("GET", "/v1/sm/profiles/*", h.adapt(h.GetSMProfile))
	router.Handle("POST", "/v1/sm/documents", h.adapt(h.CreateSMDocument))
	router.Handle("GET", "/v1/sm/documents/*", h.adapt(h.GetSMDocument))
}

// ─── Memobase Handlers ──────────────────────────────────────────────────────

func (h *MemoryHandler) InsertBlob(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	tenantID := r.Header.Get("X-Tenant-ID")
	var blob memobase.Blob
	if err := json.NewDecoder(r.Body).Decode(&blob); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	result, err := h.mbIngest.InsertBlob(r.Context(), userID, tenantID, &blob)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *MemoryHandler) Flush(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if err := h.mbIngest.Flush(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"flushed": true, "user_id": userID})
}

func (h *MemoryHandler) GetContext(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	tenantID := r.Header.Get("X-Tenant-ID")
	ctx, err := h.mbContext.GetContext(r.Context(), userID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ctx)
}

func (h *MemoryHandler) GetProfiles(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	tenantID := r.Header.Get("X-Tenant-ID")
	profiles, err := h.mbContext.GetProfiles(r.Context(), userID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": profiles})
}

func (h *MemoryHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	events, err := h.mbContext.GetEvents(r.Context(), userID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// ─── Zep Handlers ───────────────────────────────────────────────────────────

func (h *MemoryHandler) CreateZepUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    string         `json:"user_id"`
		Email     string         `json:"email"`
		FirstName string         `json:"first_name"`
		LastName  string         `json:"last_name"`
		Metadata  map[string]any `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	user, err := h.zepUser.CreateUser(r.Context(), req.UserID, req.Email, req.FirstName, req.LastName, req.Metadata)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *MemoryHandler) GetZepUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	user, err := h.zepUser.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *MemoryHandler) UpdateZepUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	var updates map[string]any
	_ = json.NewDecoder(r.Body).Decode(&updates)
	user, err := h.zepUser.UpdateUser(r.Context(), userID, updates)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *MemoryHandler) PutMemory(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var mem zep.ZepMemory
	if err := json.NewDecoder(r.Body).Decode(&mem); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	mem.SessionID = sessionID
	if err := h.zepMem.PutMemory(r.Context(), sessionID, &mem); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"stored": true})
}

func (h *MemoryHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	mem, err := h.zepMem.GetMemory(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mem)
}

func (h *MemoryHandler) SessionSearch(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	msgs, err := h.zepMem.SessionSearch(r.Context(), sessionID, req.Query, req.Limit)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
}

func (h *MemoryHandler) GraphSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	facts, err := h.zepGraph.GraphSearch(r.Context(), req.UserID, req.Query, req.Limit)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"facts": facts})
}

func (h *MemoryHandler) AddFact(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string       `json:"user_id"`
		Fact   zep.GraphFact `json:"fact"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.zepGraph.AddFact(r.Context(), req.UserID, &req.Fact); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"added": true})
}

// ─── Supermemory Handlers ───────────────────────────────────────────────────

func (h *MemoryHandler) CreateSMMemory(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	mem, err := h.smMemory.CreateMemory(r.Context(), tenantID, req.Content, req.Tags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, mem)
}

func (h *MemoryHandler) SMRag(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	result, err := h.smMemory.RAG(r.Context(), tenantID, req.Query, req.Limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *MemoryHandler) GetSMProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	tenantID := r.Header.Get("X-Tenant-ID")
	// Return profile skeleton (memories fetched separately for perf)
	writeJSON(w, http.StatusOK, &sm.SMProfile{UserID: userID, TenantID: tenantID})
}

func (h *MemoryHandler) CreateSMDocument(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Type    string `json:"type"`
		URL     string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Type == "" {
		req.Type = "markdown"
	}
	doc, err := h.smDoc.CreateDocument(r.Context(), tenantID, req.Title, req.Content, req.Type, req.URL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *MemoryHandler) GetSMDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	doc, err := h.smDoc.GetDocument(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// ─── adapt helper ───────────────────────────────────────────────────────────

func (h *MemoryHandler) adapt(hf http.HandlerFunc) forward.HandlerFunc {
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
