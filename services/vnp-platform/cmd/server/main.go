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

	// Adapters
	grpchandler "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/adapter/grpc"

	// Usecases
	analyticssvc "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/analytics"
	eventsvc "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/event"
	projectsvc "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/project"
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

	// --- Infrastructure ---
	// TODO: pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	// TODO: nc, err := nats.Connect(cfg.NatsURL)
	// TODO: publisher := natspub.NewPublisher(nc)

	// --- Repositories ---
	// TODO: Wire PostgreSQL repositories implementing port interfaces
	// tenantRepo := persistence.NewTenantRepo(pool)
	// userRepo := persistence.NewUserRepo(pool)
	// keyRepo := persistence.NewAPIKeyRepo(pool)
	// eventRepo := persistence.NewEventRepo(pool)
	// usageRepo := persistence.NewUsageRepo(pool)
	// spaceRepo := persistence.NewSpaceRepo(pool)

	// --- Usecase Services ---
	// TODO: adminSvc := adminsvc.NewService(tenantRepo, keyRepo, publisher)
	_ = eventsvc.NewService(nil)     // eventRepo
	_ = analyticssvc.NewService(nil) // usageRepo
	_ = projectsvc.NewService(nil)   // spaceRepo

	// --- gRPC Handlers ---
	_ = grpchandler.NewAdminHandler(nil, nil, nil, nil)
	_ = grpchandler.NewEventHandler(nil)
	_ = grpchandler.NewAnalyticsHandler(nil)
	_ = grpchandler.NewProjectHandler(nil)

	// --- gRPC Server ---
	grpcServer := grpc.NewServer()

	// Health check — register all 7 service names for backward compatibility
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSrv)
	for _, svcName := range []string{
		"vnp.admin.v1.AdminService",
		"vnp.event.v1.EventService",
		"ov.admin.v1.OvAdminService",
		"zep.admin.v1.ZepAdminService",
		"sm.auth.v1.AuthService",
		"sm.analytics.v1.AnalyticsService",
		"sm.project.v1.ProjectService",
	} {
		healthSrv.SetServingStatus(svcName, healthpb.HealthCheckResponse_SERVING)
	}

	// TODO: Register all 7 gRPC service handlers
	// pb.RegisterVnpAdminServiceServer(grpcServer, adminHandler)
	// pb.RegisterVnpEventServiceServer(grpcServer, eventHandler)
	// pb.RegisterOvAdminServiceServer(grpcServer, adminHandler)
	// pb.RegisterZepAdminServiceServer(grpcServer, adminHandler)
	// pb.RegisterSmAuthServiceServer(grpcServer, authHandler)
	// pb.RegisterSmAnalyticsServiceServer(grpcServer, analyticsHandler)
	// pb.RegisterSmProjectServiceServer(grpcServer, projectHandler)

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
	// TODO: pool.Close(), nc.Close()
	slog.Info("vnp-platform stopped")
}
