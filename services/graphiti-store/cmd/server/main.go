package main

import (
	"vnp-memory/pkg/forward"
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/pkg/telemetry"
	"vnp-memory/pkg/tenant"
)

// Enterprise-grade bootstrap for graphiti-store
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize Structured Logging
	telemetry.InitLogger("info")
	slog.Info("Initializing graphiti-store at enterprise-grade...")

	// 2. Initialize OpenTelemetry Distributed Tracing
	shutdownTracer, err := telemetry.InitProvider(ctx, "graphiti-store")
	if err != nil {
		slog.Error("failed to initialize OTel provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			slog.Error("failed to shutdown OTel provider gracefully", slog.String("error", err.Error()))
		}
	}()

	// 3. Setup gRPC Server with Interceptors (Tenant isolation & OTel traces)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)

	// 4. Setup Health Probes
	// Setup ForwardService Router
	router := forward.NewRouter()
	forward.RegisterForwardService(grpcServer, router)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	// 5. Start HTTP Health/Metrics Server
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		slog.Info("Starting HTTP probe server on :9199")
		if err := http.ListenAndServe(":9199", mux); err != nil {
			slog.Error("HTTP probe server failed", slog.String("error", err.Error()))
		}
	}()

	// 6. Start gRPC Server
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		slog.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}
	healthCheck.SetServingStatus("graphiti-store", grpc_health_v1.HealthCheckResponse_SERVING)
	
	go func() {
		slog.Info("Starting gRPC server on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", slog.String("error", err.Error()))
		}
	}()

	// 7. Graceful Shutdown Management
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
	slog.Info("Server exited properly")
}
