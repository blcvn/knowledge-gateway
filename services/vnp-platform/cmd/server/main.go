// Package main is the entry point for vnp-platform — the unified
// Admin + Event + Auth + Analytics + Project platform service.
//
// Consolidated from: vnp-admin, vnp-event, ov-admin, zep-admin,
// sm-auth, sm-analytics, sm-project (7 services → 1).
//
// Exposes 7 gRPC service definitions on a single port for proto
// backward compatibility.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/infra/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// --- gRPC Server ---
	grpcServer := grpc.NewServer()

	// Health check
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)

	// Register all 7 consolidated gRPC services
	// TODO: Wire up actual handlers after domain/usecase implementation
	// registerAdminService(grpcServer, ...)     // VnpAdminService
	// registerEventService(grpcServer, ...)     // VnpEventService
	// registerOvAdminService(grpcServer, ...)   // OvAdminService
	// registerZepAdminService(grpcServer, ...)  // ZepAdminService
	// registerAuthService(grpcServer, ...)      // SmAuthService
	// registerAnalyticsService(grpcServer, ...) // SmAnalyticsService
	// registerProjectService(grpcServer, ...)   // SmProjectService

	reflection.Register(grpcServer)

	// --- Start Listener ---
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("failed to listen", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	// --- Graceful Shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("vnp-platform gRPC server starting",
			"grpc_port", cfg.GRPCPort,
			"services", []string{
				"VnpAdminService", "VnpEventService", "OvAdminService",
				"ZepAdminService", "SmAuthService", "SmAnalyticsService",
				"SmProjectService",
			},
		)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down vnp-platform gracefully...")
	grpcServer.GracefulStop()
	slog.Info("vnp-platform stopped")
}
