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

	consoleHandler "vnp-memory/services/vnp-observability/internal/adapter/grpc"
	"vnp-memory/services/pkg/forward"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	slog.Info("Starting vnp-observability service...")

	handler := consoleHandler.NewObservabilityHandler()

	router := forward.NewRouter(logger)
	router.Handle("GET", "/v1/console/observability/metrics", wrapHandler(handler.GetMetrics))
	router.Handle("GET", "/v1/console/observability/traces", wrapHandler(handler.ListTraces))
	router.Handle("GET", "/v1/console/observability/traces/*", wrapIDHandler(handler.GetTrace))
	router.Handle("GET", "/v1/console/observability/errors", wrapHandler(handler.GetErrors))
	router.Handle("GET", "/v1/console/observability/costs", wrapHandler(handler.GetCosts))

	grpcServer := grpc.NewServer()
	forward.RegisterForwardService(grpcServer, router)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		port := envOr("HEALTH_PORT", "9146")
		slog.Info("HTTP probe server", "port", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			slog.Error("HTTP probe server failed", "error", err)
		}
	}()

	port := envOr("GRPC_PORT", "9046")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		os.Exit(1)
	}
	healthCheck.SetServingStatus("vnp-observability", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		slog.Info("gRPC server starting", "port", port)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down vnp-observability...")
	grpcServer.GracefulStop()
	cancel()
	slog.Info("vnp-observability stopped")
}

func wrapHandler(fn func(ctx context.Context) ([]byte, error)) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		return fn(ctx)
	}
}

func wrapIDHandler(fn func(ctx context.Context, id string) ([]byte, error)) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		id := params["id"]
		if id == "" {
			id = "unknown"
		}
		return fn(ctx, id)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
