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

	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"
	consoleHandler "vnp-memory/services/sm-memory/adapter/grpc"
	"vnp-memory/shared/pkg/forward"
)

// Enterprise-grade bootstrap for sm-memory
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	telemetry.InitLogger("info")
	slog.Info("Initializing sm-memory at enterprise-grade...")

	shutdownTracer, err := telemetry.InitProvider(ctx, "sm-memory")
	if err != nil {
		slog.Error("failed to initialize OTel provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			slog.Error("failed to shutdown OTel provider gracefully", slog.String("error", err.Error()))
		}
	}()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)

	// Register ForwardService with console routes
	handler := consoleHandler.NewSmMemoryHandler()
	router := forward.NewRouter(slog.Default())
	router.Handle("GET", "/v1/console/adaptive/memories", wrapHandler(handler.ListMemories))
	router.Handle("GET", "/v1/console/adaptive/memories/*", wrapIDHandler(handler.GetVersions))
	router.Handle("GET", "/v1/console/memory/*/versions", wrapIDHandler(handler.GetVersions))
	forward.RegisterForwardService(grpcServer, router)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

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

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		slog.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}
	healthCheck.SetServingStatus("sm-memory", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		slog.Info("Starting gRPC server on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
	slog.Info("Server exited properly")
}

func wrapHandler(fn func(ctx context.Context) ([]byte, error)) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		return fn(ctx)
	}
}

func wrapIDHandler(fn func(ctx context.Context, id string) ([]byte, error)) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		return fn(ctx, params["id"])
	}
}
