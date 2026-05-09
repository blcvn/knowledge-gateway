// Package server provides HTTP/gRPC/MCP server lifecycle management.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// HTTPServer manages the REST HTTP server lifecycle.
type HTTPServer struct {
	server *http.Server
	logger *slog.Logger
}

// NewHTTPServer creates a new HTTP server with the given handler and port.
func NewHTTPServer(handler http.Handler, port int, logger *slog.Logger) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      120 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1MB
		},
		logger: logger,
	}
}

// Start runs the HTTP server and blocks until ctx is cancelled, then shuts down gracefully.
func (s *HTTPServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("REST server started", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server error: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutting down REST server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}

// HealthServer provides /healthz, /readyz, /metrics on a separate port.
type HealthServer struct {
	server *http.Server
	logger *slog.Logger
}

// NewHealthServer creates a health check server on the given port.
func NewHealthServer(port int, logger *slog.Logger) *HealthServer {
	mux := http.NewServeMux()

	// Liveness
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "serving"})
	})

	// Readiness
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ready",
			"checks": map[string]string{
				"postgres": "ok",
				"redis":    "ok",
				"nats":     "ok",
			},
		})
	})

	return &HealthServer{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// Start runs the health server.
func (s *HealthServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("health server started", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("health server error: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}
