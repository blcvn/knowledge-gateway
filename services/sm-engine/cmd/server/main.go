// Package main is the entry point for sm-engine — the unified
// Document + Memory + Profile adaptive KG engine.
//
// Consolidated from: sm-document + sm-memory + sm-profile (3 → 1).
// Document → memory → profile chain is now a local in-process workflow.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/sm-engine/internal/infra/config"
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
	// registerDocumentService(srv, ...)  // SmDocumentService
	// registerMemoryService(srv, ...)    // SmMemoryService
	// registerProfileService(srv, ...)   // SmProfileService

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("listen failed", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("sm-engine starting", "grpc_port", cfg.GRPCPort,
			"services", []string{"SmDocumentService", "SmMemoryService", "SmProfileService"})
		if err := srv.Serve(lis); err != nil {
			slog.Error("serve failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("sm-engine shutting down...")
	srv.GracefulStop()
}
