// Package main — gateway embedding function.
//
// Replicates the gateway startup pattern from gateway/cmd/main.go
// with the key difference: service addresses point to localhost
// (embedded gRPC services) instead of remote Kubernetes DNS names.
//
// The gateway exposes REST endpoints that proxy to gRPC services.
// In v1, routes are thin wrappers. Full gRPC-JSON transcoding can be
// added when protobuf definitions are wired.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vnp-community/vnp-memory/apps/cognee/internal/config"
	"github.com/vnp-community/vnp-memory/apps/cognee/internal/gateway"
)

// startGateway starts the HTTP gateway that routes REST requests to
// embedded gRPC services via localhost connections.
//
// Startup sequence:
//  1. Create gRPC registry → connect to localhost services
//  2. Build HTTP router with middleware
//  3. Start HTTP server
//  4. Block until ctx cancelled → graceful shutdown
//
// Reference: gateway/cmd/main.go (264 lines)
func startGateway(ctx context.Context, cfg *config.Config) error {
	logger := slog.Default()

	// 1. Create gRPC registry with localhost services
	servicesMap := cfg.GatewayServicesMap()
	logger.Info("gateway starting with embedded services",
		"services", servicesMap,
		"rest_port", cfg.Server.RESTPort,
	)

	grpcReg := gateway.NewGRPCRegistry(servicesMap, logger)
	if err := grpcReg.Connect(ctx); err != nil {
		return fmt.Errorf("gateway gRPC registry connect: %w", err)
	}
	defer grpcReg.Close()

	// 2. Build router with middleware
	mux := http.NewServeMux()

	// --- Health endpoints ---
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		// Probe all embedded gRPC services
		status := grpcReg.HealthCheckAll(r.Context())
		allReady := true
		for _, s := range status {
			if s != "SERVING" {
				allReady = false
				break
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if !allReady {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"status":   readyStr(allReady),
			"services": status,
		})
	})

	// --- Cognee API v1 routes ---
	// These are placeholder routes that will be replaced with gRPC-to-JSON
	// transcoding once protobuf service definitions are fully wired.
	// The gRPC connections are already established above.

	// Ingestion routes
	mux.HandleFunc("POST /v1/cognee/datasets", grpcProxy("cognee.createDataset", grpcReg))
	mux.HandleFunc("POST /v1/cognee/datasets/{id}/data", grpcProxy("cognee.uploadData", grpcReg))
	mux.HandleFunc("GET /v1/cognee/datasets", grpcProxy("cognee.listDatasets", grpcReg))
	mux.HandleFunc("DELETE /v1/cognee/datasets/{id}", grpcProxy("cognee.deleteDataset", grpcReg))

	// Cognify routes
	mux.HandleFunc("POST /v1/cognee/datasets/{id}/cognify", grpcProxy("cognee.startCognify", grpcReg))
	mux.HandleFunc("GET /v1/cognee/cognify/{id}/status", grpcProxy("cognee.cognifyStatus", grpcReg))

	// Search routes
	mux.HandleFunc("POST /v1/cognee/search", grpcProxy("cognee.search", grpcReg))
	mux.HandleFunc("POST /v1/cognee/search/rag", grpcProxy("cognee.searchRAG", grpcReg))
	mux.HandleFunc("GET /v1/cognee/search/explore", grpcProxy("cognee.explore", grpcReg))

	// --- Default 404 ---
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "NOT_FOUND",
				"message": "route not found: " + r.Method + " " + r.URL.Path,
			},
		})
	})

	// 3. Wrap with middleware stack
	handler := corsMiddleware(
		cfg.CORS.AllowedOrigins,
		recoveryMiddleware(logger,
			loggingMiddleware(logger, mux),
		),
	)

	// 4. HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.RESTPort),
		Handler:      handler,
		ReadTimeout:  cfg.Timeout.Default,
		WriteTimeout: cfg.Timeout.Default + 5*time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 5. Serve until context cancelled
	errCh := make(chan error, 1)
	go func() {
		logger.Info("gateway HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("gateway serve: %w", err)
	case <-ctx.Done():
		logger.Info("gateway shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// grpcProxy returns a handler that will proxy REST requests to gRPC services.
// In v1, this responds with 501 + metadata about the intended route.
// In v2, this will use gRPC-JSON transcoding with the established connections.
func grpcProxy(route string, reg *gateway.GRPCRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "NOT_IMPLEMENTED",
				"message": fmt.Sprintf("Route %s: gRPC proxy wiring pending protobuf definitions", route),
			},
			"_meta": map[string]string{
				"route":    route,
				"registry": "connected",
			},
		})
	}
}

// readyStr returns "ready" or "not_ready" for JSON output.
func readyStr(ready bool) string {
	if ready {
		return "ready"
	}
	return "not_ready"
}

// --- Middleware Stack ---

// recoveryMiddleware wraps an http.Handler with panic recovery.
func recoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered in gateway",
					"error", err,
					"path", r.URL.Path,
					"method", r.Method,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "INTERNAL_ERROR",
						"message": "internal server error",
					},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs each HTTP request with structured fields.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// corsMiddleware adds CORS headers.
func corsMiddleware(allowedOrigins string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// statusWriter wraps http.ResponseWriter to capture status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
