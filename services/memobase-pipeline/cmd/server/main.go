// Package main is the entry point for memobase-pipeline — the unified
// Buffer Zone FSM + YOLO merge engine service.
//
// Consolidated from: memobase-ingestion + memobase-engine (2 → 1).
// Buffer flush now triggers YOLO merge via local function call.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/infra/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, health.NewServer())

	// Register consolidated gRPC service
	// TODO: Wire domain/usecase after implementation
	// registerIngestionService(srv, ...)  // MemobaseIngestionService (Buffer + Engine)

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("listen failed", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("memobase-pipeline starting", "grpc_port", cfg.GRPCPort,
			"services", []string{"MemobaseIngestionService"})
		if err := srv.Serve(lis); err != nil {
			slog.Error("serve failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("memobase-pipeline shutting down...")
	srv.GracefulStop()
}
