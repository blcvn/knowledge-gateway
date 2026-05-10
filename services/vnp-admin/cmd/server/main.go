// vnp-admin — Central Administration Service
// gRPC port: 9050 | Health port: 9100
//
// Provides: Tenant CRUD, API key lifecycle, User management, Health aggregation
// Publishes: admin.tenant.created, admin.tenant.deleted, admin.apikey.revoked
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/infra/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.Load()
	log.Printf("vnp-admin starting on :%s (health :%s)", cfg.GRPCPort, cfg.HealthPort)

	// --- Infrastructure ---
	// TODO: pgxpool.New(ctx, cfg.DatabaseURL)
	// TODO: nats.Connect(cfg.NatsURL)

	// --- Dependency Injection ---
	// TODO: Wire repositories, publishers, usecases, and handlers
	// tenantRepo := persistence.NewTenantRepo(pool)
	// keyRepo := persistence.NewAPIKeyRepo(pool)
	// userRepo := persistence.NewUserRepo(pool)
	// publisher := natspub.NewPublisher(nc)
	// tenantSvc := usecase.NewTenantService(tenantRepo, publisher)
	// keySvc := usecase.NewAPIKeyService(keyRepo, tenantRepo, publisher)
	// handler := grpcadapter.NewAdminHandler(tenantSvc, keySvc, nil, nil)

	// --- gRPC Server ---
	grpcServer := grpc.NewServer()

	// Register health service
	healthSvc := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthSvc)
	healthSvc.SetServingStatus("vnp.admin.v1.AdminService", healthpb.HealthCheckResponse_SERVING)

	// Enable reflection for service discovery
	reflection.Register(grpcServer)

	// TODO: Register VNPAdminService handler
	// pb.RegisterVNPAdminServiceServer(grpcServer, handler)

	// --- Start Listener ---
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	// --- Graceful Shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("vnp-admin gRPC server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("vnp-admin shutting down...")
	grpcServer.GracefulStop()
	// TODO: pool.Close(), nc.Close()
	log.Println("vnp-admin stopped")
}
