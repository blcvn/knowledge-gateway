// Package handler — SPA static file server for the embedded VNP Memory Console UI.
package handler

import (
	"io/fs"
	"net/http"
	"strings"
)

// SPAHandler serves static files from an embedded filesystem and falls back
// to index.html for any path that doesn't match a real file. This enables
// client-side routing in the React SPA.
type SPAHandler struct {
	staticFS   fs.FS
	fileServer http.Handler
}

// NewSPAHandler creates a handler that serves static files from the given fs.FS.
// Paths not matching a real file are served index.html so the SPA can handle routing.
func NewSPAHandler(staticFS fs.FS) *SPAHandler {
	return &SPAHandler{
		staticFS:   staticFS,
		fileServer: http.FileServer(http.FS(staticFS)),
	}
}

// ServeHTTP implements http.Handler.
//
// Routing logic:
//  1. If the path starts with /v1/ → skip (handled by API router before this)
//  2. If a file exists at the requested path → serve it (JS, CSS, images, etc.)
//  3. Otherwise → serve index.html (SPA fallback for client-side routes)
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip leading slash for fs.Open
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try to open the file — if it exists and is not a directory, serve it
	if f, err := h.staticFS.Open(path); err == nil {
		defer f.Close()
		if stat, err := f.Stat(); err == nil && !stat.IsDir() {
			h.fileServer.ServeHTTP(w, r)
			return
		}
	}

	// SPA fallback: serve index.html for client-side routing
	indexFile, err := fs.ReadFile(h.staticFS, "index.html")
	if err != nil {
		http.Error(w, "UI not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(indexFile)
}
