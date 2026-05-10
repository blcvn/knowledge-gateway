// Package main is the entry point for graphiti-pipeline — the unified
// Graphiti ingestion + knowledge extraction pipeline service.
//
// Consolidated from: graphiti-ingestion + graphiti-knowledge (2 → 1).
// Saga orchestrator now calls knowledge RPCs locally instead of cross-service gRPC.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/graphiti-pipeline/internal/infra/config"
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
	// registerIngestionService(srv, ...)   // GraphitiIngestionService (saga orchestrator)
	// registerKnowledgeService(srv, ...)   // GraphitiKnowledgeService (LLM extraction)

	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("listen failed", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("graphiti-pipeline starting", "grpc_port", cfg.GRPCPort,
			"services", []string{"GraphitiIngestionService", "GraphitiKnowledgeService"})
		if err := srv.Serve(lis); err != nil {
			slog.Error("serve failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("graphiti-pipeline shutting down...")
	srv.GracefulStop()
}
