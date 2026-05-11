package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/infra/config"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/infra/server"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/infra/telemetry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	shutdownTelemetry, err := telemetry.Init(ctx, cfg.Service.Name, cfg.Telemetry.OTLPEndpoint)
	if err != nil {
		slog.Error("Failed to initialize telemetry", "error", err)
		os.Exit(1)
	}
	defer shutdownTelemetry(context.Background())

	slog.Info("Starting service", "name", cfg.Service.Name, "version", cfg.Service.Version)

	// Since we mock wire initialization, we simulate starting the server
	// grpcHandler := &grpc_adapter.handler{} // Mock handler
	grpcSrv, err := server.StartGRPCServer(ctx, cfg.GRPC.Port, nil)
	if err != nil {
		slog.Error("Failed to start gRPC server", "error", err)
		os.Exit(1)
	}
	
	slog.Info("gRPC Server started", "port", cfg.GRPC.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down gracefully...")
	grpcSrv.GracefulStop()
	time.Sleep(1 * time.Second)
	slog.Info("Shutdown complete")
}
