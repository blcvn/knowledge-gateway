// Package grpc implements the ForwardService handler for storage-service.
//
// All routes use ForwardService pattern: forward.HandlerFunc via HTTP JSON.
// Absorbed from: ov-fs, ov-resource, ov-session (MERGE-P1-T4)
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	fsuc "vnp-memory/services/storage-service/internal/usecase/fs"
	resourceuc "vnp-memory/services/storage-service/internal/usecase/resource"
	sessionuc "vnp-memory/services/storage-service/internal/usecase/session"

	"vnp-memory/services/storage-service/internal/domain/resource"
	"vnp-memory/services/storage-service/internal/domain/session"
)

// StorageHandler handles all storage-service HTTP endpoints.
type StorageHandler struct {
	fs       *fsuc.Service
	sessions *sessionuc.Service
	resources *resourceuc.Service
}

// NewStorageHandler creates a StorageHandler.
func NewStorageHandler(fs *fsuc.Service, sessions *sessionuc.Service, res *resourceuc.Service) *StorageHandler {
	return &StorageHandler{fs: fs, sessions: sessions, resources: res}
}

// ═══════════════════════════════════════════════════════
// File System Handlers (ov-fs)
// ═══════════════════════════════════════════════════════

// ReadFile — GET /v1/ov/files/*
func (h *StorageHandler) ReadFile(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	path := r.PathValue("path")
	if path == "" {
		path = r.URL.Path
	}

	file, err := h.fs.ReadFile(r.Context(), tenantID, path)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Return raw content if not JSON request
	if r.Header.Get("Accept") == "application/octet-stream" {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(file.Content)
		return
	}

	writeJSON(w, http.StatusOK, file)
}

// WriteFile — PUT /v1/ov/files/*
func (h *StorageHandler) WriteFile(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	path := r.PathValue("path")

	var req struct {
		Content string `json:"content"`
		Path    string `json:"path,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if path == "" {
		path = req.Path
	}

	if err := h.fs.WriteFile(r.Context(), tenantID, path, []byte(req.Content)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path, "written": true})
}

// DeleteFile — DELETE /v1/ov/files/*
func (h *StorageHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	path := r.PathValue("path")

	if err := h.fs.DeleteFile(r.Context(), tenantID, path); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// Tree — GET /v1/ov/tree/*
func (h *StorageHandler) Tree(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	path := r.PathValue("path")
	if path == "" {
		path = "/"
	}

	dir, err := h.fs.Tree(r.Context(), tenantID, path, 3)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dir)
}

// Grep — POST /v1/ov/grep
func (h *StorageHandler) Grep(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		Path    string `json:"path"`
		Pattern string `json:"pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Path == "" {
		req.Path = "/"
	}

	results, err := h.fs.Grep(r.Context(), tenantID, req.Path, req.Pattern)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "count": len(results)})
}

// ═══════════════════════════════════════════════════════
// Session Handlers (ov-session)
// ═══════════════════════════════════════════════════════

// CreateSession — POST /v1/ov/sessions
func (h *StorageHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		BaseDir string `json:"base_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BaseDir == "" {
		req.BaseDir = "/"
	}

	sess, err := h.sessions.Create(r.Context(), tenantID, req.BaseDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

// AddMessage — POST /v1/ov/sessions/{id}/messages
func (h *StorageHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var req struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	msg := &session.Message{
		Role:    req.Role,
		Content: req.Content,
	}
	if err := h.sessions.AddMessage(r.Context(), sessionID, msg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

// CommitSession — POST /v1/ov/sessions/{id}/commit
func (h *StorageHandler) CommitSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	record, err := h.sessions.Commit(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

// ═══════════════════════════════════════════════════════
// Resource Handlers (ov-resource)
// ═══════════════════════════════════════════════════════

// IngestResource — POST /v1/ov/resources/ingest
func (h *StorageHandler) IngestResource(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	var req struct {
		URI     string                   `json:"uri"`
		Options resource.IngestOptions   `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	job := &resource.IngestJob{
		URI:     req.URI,
		Options: req.Options,
	}

	res, err := h.resources.Ingest(r.Context(), tenantID, job)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, res)
}

// GetResourceStatus — GET /v1/ov/resources/{id}
func (h *StorageHandler) GetResourceStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.resources.GetStatus(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ─── helpers ──────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

var _ = fmt.Sprintf // suppress unused import
var _ = context.Background
