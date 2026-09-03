// Package server provides gRPC + HTTP server lifecycle management.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"vnp-memory/services/cognee-ingestion/internal/infra/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const gracefulShutdownTimeout = 30 * time.Second

// Server encapsulates both the gRPC server and HTTP health server.
type Server struct {
	grpcServer   *grpc.Server
	healthServer *health.Server
	httpServer   *http.Server
	cfg          *config.Config
	logger       *slog.Logger
}

// New creates a new Server with gRPC and HTTP health check.
func New(cfg *config.Config, logger *slog.Logger, opts ...grpc.ServerOption) *Server {
	// Default gRPC server options
	defaultOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.GRPC.MaxSendMsgSize),
		grpc.ConnectionTimeout(cfg.GRPC.ConnectionTimeout),
	}
	allOpts := append(defaultOpts, opts...)

	grpcSrv := grpc.NewServer(allOpts...)
	healthSrv := health.NewServer()

	// Register health service
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	// Enable reflection for development tools (grpcurl, etc.)
	reflection.Register(grpcSrv)

	// HTTP health check
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler(cfg))
	mux.HandleFunc("/readyz", readyHandler(healthSrv))

	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Health.Port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	return &Server{
		grpcServer:   grpcSrv,
		healthServer: healthSrv,
		httpServer:   httpSrv,
		cfg:          cfg,
		logger:       logger.With("component", "server"),
	}
}

// GRPCServer returns the underlying grpc.Server for service registration.
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

// SetServingStatus sets the health status for a specific service.
func (s *Server) SetServingStatus(service string, serving bool) {
	if serving {
		s.healthServer.SetServingStatus(service, healthpb.HealthCheckResponse_SERVING)
	} else {
		s.healthServer.SetServingStatus(service, healthpb.HealthCheckResponse_NOT_SERVING)
	}
}

// Start begins listening on both gRPC and HTTP ports.
// This is a blocking call — use in a goroutine.
func (s *Server) Start(ctx context.Context) error {
	// Start HTTP health server in background
	go func() {
		s.logger.Info("http health server starting", "port", s.cfg.Health.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("http health server failed", "error", err)
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GRPC.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", s.cfg.GRPC.Port, err)
	}

	s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	s.logger.Info("grpc server starting",
		"port", s.cfg.GRPC.Port,
		"service", s.cfg.Service.Name,
		"version", s.cfg.Service.Version,
	)

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

// Shutdown performs graceful shutdown of both servers.
func (s *Server) Shutdown() {
	s.logger.Info("initiating graceful shutdown", "timeout", gracefulShutdownTimeout)

	// Mark health as not serving
	s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	// Graceful stop gRPC (drains in-flight RPCs)
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or timeout
	timer := time.NewTimer(gracefulShutdownTimeout)
	defer timer.Stop()

	select {
	case <-stopped:
		s.logger.Info("grpc server stopped gracefully")
	case <-timer.C:
		s.logger.Warn("graceful shutdown timed out, forcing stop")
		s.grpcServer.Stop()
	}

	// Shutdown HTTP server
	httpCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(httpCtx); err != nil {
		s.logger.Error("http shutdown failed", "error", err)
	}

	s.logger.Info("server shutdown complete")
}

// healthHandler returns service identity and status.
func healthHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": cfg.Service.Name,
			"version": cfg.Service.Version,
		})
	}
}

// readyHandler checks if gRPC health is SERVING.
func readyHandler(hs *health.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := hs.Check(r.Context(), &healthpb.HealthCheckRequest{})
		if err != nil || resp.Status != healthpb.HealthCheckResponse_SERVING {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}
