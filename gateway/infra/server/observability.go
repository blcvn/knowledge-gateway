package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vnp-community/vnp-memory/gateway/usecase/port"
)

// ObservabilityServer provides /healthz, /readyz, /healthz/deep, /metrics on a separate port.
type ObservabilityServer struct {
	server   *http.Server
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewObservabilityServer creates an observability server with health + metrics endpoints.
func NewObservabilityServer(port int, registry port.ServiceRegistry, logger *slog.Logger) *ObservabilityServer {
	mux := http.NewServeMux()
	s := &ObservabilityServer{
		registry: registry,
		logger:   logger,
	}

	// Liveness — always returns OK if process is running
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "serving"})
	})

	// Readiness — checks local dependencies
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// TODO: Real checks against postgres, redis, nats
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ready",
			"checks": map[string]string{
				"postgres": "ok",
				"redis":    "ok",
				"nats":     "ok",
			},
		})
	})

	// Deep health — cascade to all downstream services
	mux.HandleFunc("GET /healthz/deep", s.deepHealthHandler)

	// Prometheus metrics
	mux.Handle("GET /metrics", promhttp.Handler())

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

// deepHealthHandler checks health of all downstream services.
func (s *ObservabilityServer) deepHealthHandler(w http.ResponseWriter, r *http.Request) {
	services := []string{
		"cognee-ingestion", "cognee-search", "graphiti-ingestion", "graphiti-search",
		"memobase-ingestion", "memobase-context", "ov-fs", "ov-search",
		"zep-user", "zep-memory", "zep-search", "sm-document", "sm-search",
		"vnp-event", "vnp-search-hub", "vnp-admin",
	}

	statuses := make(map[string]string, len(services))
	healthy := 0
	for _, svc := range services {
		ok, err := s.registry.HealthCheck(svc)
		if ok && err == nil {
			statuses[svc] = "ok"
			healthy++
		} else {
			statuses[svc] = "unhealthy"
		}
	}

	overall := "healthy"
	if healthy < len(services) {
		overall = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":   overall,
		"healthy":  healthy,
		"total":    len(services),
		"services": statuses,
	})
}

// Start runs the observability server.
func (s *ObservabilityServer) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("observability server started", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("observability server error: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}
