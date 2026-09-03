package main

import (
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

	consoleHandler "github.com/vnp-community/vnp-memory/services/vnp-dashboard/adapter/grpc"
	"github.com/vnp-community/vnp-memory/shared/pkg/forward"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	slog.Info("Starting vnp-dashboard service...")

	// 1. Initialize handler
	handler := consoleHandler.NewDashboardHandler()

	// 2. Setup ForwardService router — maps HTTP paths to handler methods
	router := forward.NewRouter(logger)
	router.Handle("GET", "/v1/console/dashboard/health", wrapHandler(handler.GetHealth))
	router.Handle("GET", "/v1/console/dashboard/metrics", wrapHandler(handler.GetMetrics))
	router.Handle("GET", "/v1/console/dashboard/throughput", wrapHandler(handler.GetThroughput))
	router.Handle("GET", "/v1/console/dashboard/heatmap", wrapHandler(handler.GetHeatmap))

	// 3. Setup gRPC Server
	grpcServer := grpc.NewServer()
	forward.RegisterForwardService(grpcServer, router)

	// 4. Health probes
	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	// 5. HTTP health endpoint
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		port := envOr("HEALTH_PORT", "9143")
		slog.Info("HTTP probe server", "port", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			slog.Error("HTTP probe server failed", "error", err)
		}
	}()

	// 6. Start gRPC server
	port := envOr("GRPC_PORT", "9043")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}
	healthCheck.SetServingStatus("vnp-dashboard", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		slog.Info("gRPC server starting", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	// 7. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down vnp-dashboard...")
	grpcServer.GracefulStop()
	cancel()
	slog.Info("vnp-dashboard stopped")
}

// wrapHandler wraps a simple context handler into a forward.HandlerFunc.
func wrapHandler(fn func(ctx context.Context) ([]byte, error)) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		return fn(ctx)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
