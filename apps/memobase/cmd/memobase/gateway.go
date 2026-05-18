// Package main — gateway embedding function.
//
// Replicates the gateway startup pattern from gateway/cmd/main.go
// with the key difference: service addresses point to localhost
// (embedded gRPC services) instead of remote Kubernetes DNS names.
//
// The gateway exposes REST + MCP endpoints that proxy to gRPC services.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vnp-community/vnp-memory/apps/memobase/internal/config"
	"github.com/vnp-community/vnp-memory/apps/memobase/internal/gateway"
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
// Reference: gateway/cmd/main.go
func startGateway(ctx context.Context, cfg *config.Config) error {
	logger := slog.Default()

	// 1. Create gRPC registry with localhost services
	servicesMap := cfg.GatewayServicesMap()
	logger.Info("gateway starting with embedded services",
		"services", servicesMap,
		"rest_port", cfg.Server.RESTPort,
		"mcp_port", cfg.Server.MCPPort,
	)

	grpcReg := gateway.NewGRPCRegistry(servicesMap, logger)
	if err := grpcReg.Connect(ctx); err != nil {
		return fmt.Errorf("gateway gRPC registry connect: %w", err)
	}
	defer grpcReg.Close()

	// 2. Build router with middleware
	mux := http.NewServeMux()

	// --- Health endpoints ---
	mux.HandleFunc("GET /api/v1/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	// --- Memobase API v1 routes ---
	// These are placeholder routes that will be replaced with actual
	// gRPC transcoding in the full implementation.

	// Ingestion routes
	mux.HandleFunc("POST /api/v1/blobs/insert/", grpcProxy("memobase-ingestion.InsertBlob", grpcReg))
	mux.HandleFunc("GET /api/v1/blobs/", grpcProxy("memobase-ingestion.GetBlob", grpcReg))

	// Context/Profile routes
	mux.HandleFunc("GET /api/v1/users/profile/", grpcProxy("memobase-context.GetProfiles", grpcReg))
	mux.HandleFunc("POST /api/v1/users/profile/", grpcProxy("memobase-context.AddProfile", grpcReg))

	// Default 404
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

// --- Middleware Stack ---

func recoveryMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered in gateway",
					"error", err,
					"path", r.URL.Path,
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
		)
	})
}

func corsMiddleware(allowedOrigins string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
