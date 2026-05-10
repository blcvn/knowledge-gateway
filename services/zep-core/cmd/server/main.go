// Package main is the entry point for zep-core — the unified
// User + Thread + Memory context engineering service.
//
// Consolidated from: zep-user + zep-thread + zep-memory (3 → 1).
// PutMemory now calls Thread.UpsertSession locally (sub-200ms hot path).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/zep-core/internal/infra/config"
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

	// Register consolidated gRPC services
	// TODO: Wire domain/usecase after implementation
	// registerUserService(srv, ...)    // ZepUserService
	// registerThreadService(srv, ...)  // ZepThreadService
	// registerMemoryService(srv, ...)  // ZepMemoryService

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("listen failed", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("zep-core starting", "grpc_port", cfg.GRPCPort,
			"services", []string{"ZepUserService", "ZepThreadService", "ZepMemoryService"})
		if err := srv.Serve(lis); err != nil {
			slog.Error("serve failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("zep-core shutting down...")
	srv.GracefulStop()
}
