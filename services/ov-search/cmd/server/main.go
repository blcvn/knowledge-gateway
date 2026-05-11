package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vnp-memory/ov-search/internal/infra/config"
)

func main() {
	// 1. Initialize Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting ov-search service...")

	// 2. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Setup Observability (OTel & Prometheus) - Pseudo implementation
	slog.Info("Configuring OpenTelemetry", "endpoint", cfg.OtelEndpoint)
	// mock initProvider()

	// 4. Initialize Dependencies via Wire (pseudo)
	// handler, subscriber, hotnessUC, err := wire.InitializeApp(cfg)
	
	// 5. Start Hotness Background Worker
	// ctx, cancel := context.WithCancel(context.Background())
	// go hotnessUC.StartWorker(ctx)

	// 6. Start Health & Metrics Server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "serving"}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Mock dependency checks (db, qdrant, nats, bifrost)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ready", "checks": {"db": "ok", "qdrant": "ok", "nats": "ok", "bifrost": "ok"}}`))
	})
	
	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HealthPort),
		Handler: mux,
	}
	
	go func() {
		slog.Info("Health server listening", "port", cfg.HealthPort)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Health server failed", "error", err)
		}
	}()

	// 7. Start gRPC Server
	// grpcServer := grpc.NewServer(...)
	// proto.RegisterOvSearchServiceServer(grpcServer, handler)
	slog.Info("gRPC server listening", "port", cfg.GRPCPort)

	// 8. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	slog.Info("Shutting down ov-search service...")

	// Create a context with 30-second timeout for graceful shutdown
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := healthServer.Shutdown(ctxShutdown); err != nil {
		slog.Error("Health server shutdown error", "error", err)
	}

	// grpcServer.GracefulStop()
	// cancel() // stop background workers

	slog.Info("ov-search service stopped gracefully")
}
