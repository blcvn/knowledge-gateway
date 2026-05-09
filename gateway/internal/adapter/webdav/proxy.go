// Package webdav provides a WebDAV reverse proxy to the ov-fs service.
package webdav

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	mw "github.com/vnp-community/vnp-memory/gateway/internal/infra/middleware"
	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// Proxy forwards WebDAV requests to the ov-fs service.
type Proxy struct {
	registry port.ServiceRegistry
	client   *http.Client
	logger   *slog.Logger
}

// NewProxy creates a new WebDAV proxy.
func NewProxy(registry port.ServiceRegistry, logger *slog.Logger) *Proxy {
	return &Proxy{
		registry: registry,
		client: &http.Client{
			Timeout: 120 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // don't follow redirects
			},
		},
		logger: logger,
	}
}

// webdavHeaders are WebDAV-specific headers that must be propagated.
var webdavHeaders = []string{
	"Depth", "Destination", "Lock-Token", "If", "Overwrite",
	"Timeout", "Content-Type", "Content-Length",
}

// ServeHTTP handles all WebDAV requests by proxying to ov-fs.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip /webdav prefix
	path := strings.TrimPrefix(r.URL.Path, "/webdav")
	if path == "" {
		path = "/"
	}

	// Resolve ov-fs target
	target, err := p.registry.Resolve("ov-fs")
	if err != nil {
		p.logger.Error("failed to resolve ov-fs", "error", err)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Build proxy request
	proxyURL := "http://" + target.Address + path
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL, r.Body)
	if err != nil {
		p.logger.Error("failed to create proxy request", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Propagate WebDAV-specific headers
	for _, h := range webdavHeaders {
		if v := r.Header.Get(h); v != "" {
			proxyReq.Header.Set(h, v)
		}
	}

	// Propagate auth context
	auth, ok := mw.AuthFromContext(r.Context())
	if ok && auth != nil {
		proxyReq.Header.Set("X-Tenant-ID", auth.TenantID)
		proxyReq.Header.Set("X-User-ID", auth.UserID)
	}

	// Propagate request ID
	if reqID := mw.RequestIDFromContext(r.Context()); reqID != "" {
		proxyReq.Header.Set("X-Request-ID", reqID)
	}

	// Execute proxy request
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		p.logger.Error("webdav proxy failed",
			"method", r.Method,
			"path", path,
			"error", err,
		)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Stream response body
	written, _ := io.Copy(w, resp.Body)

	p.logger.Debug("webdav proxied",
		"method", r.Method,
		"path", path,
		"status", resp.StatusCode,
		"bytes", written,
	)
}
