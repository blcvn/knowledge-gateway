// Package main is the entry point for ov-storage — the unified
// VikingFS + Encryption + Resource ingestion service.
//
// Consolidated from: ov-fs + ov-crypto + ov-resource (3 → 1).
// Encryption is now transparent within FS operations via local call.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/ov-storage/internal/infra/config"
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
	// registerFsService(srv, ...)       // OvFsService
	// registerCryptoService(srv, ...)   // OvCryptoService
	// registerResourceService(srv, ...) // OvResourceService

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("listen failed", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("ov-storage starting", "grpc_port", cfg.GRPCPort,
			"services", []string{"OvFsService", "OvCryptoService", "OvResourceService"})
		if err := srv.Serve(lis); err != nil {
			slog.Error("serve failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("ov-storage shutting down...")
	srv.GracefulStop()
}
