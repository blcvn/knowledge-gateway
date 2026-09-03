// Package embed provides startup functions for embedding cognee microservices
// as goroutines within the monolith process.
//
// Each function replicates the bootstrap pattern from the corresponding
// service's cmd/server/main.go WITHOUT modifying those files. The pattern is:
//
//  1. Init OTel tracing via pkg/telemetry
//  2. Create gRPC server with tenant interceptor via pkg/tenant
//  3. Register gRPC health check
//  4. Listen on configured port
//  5. Block until context cancelled → GracefulStop
//
// All embedded services communicate via gRPC localhost and NATS, identical
// to the microservice deployment topology.
package embed

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"
)

// StartGRPCService is a generic bootstrap function that replicates the common
// service startup pattern used by all cognee microservices.
//
// Reference: services/cognee-ingestion/cmd/server/main.go (lines 21-85)
//
// This pattern is identical across cognee-ingestion, cognee-cognify, and
// cognee-search because they all follow the same enterprise bootstrap template.
func StartGRPCService(ctx context.Context, serviceName string, grpcPort int) error {
	logger := slog.Default()

	// 1. Initialize OTel tracing (same as each service does)
	shutdownTracer, err := telemetry.InitProvider(ctx, serviceName)
	if err != nil {
		logger.Warn("OTel init failed, continuing without tracing",
			"service", serviceName,
			"error", err,
		)
		// Non-fatal: service can operate without tracing
	} else {
		defer func() {
			if err := shutdownTracer(context.Background()); err != nil {
				logger.Error("OTel shutdown failed",
					"service", serviceName,
					"error", err,
				)
			}
		}()
	}

	// 2. Create gRPC server with tenant isolation interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)

	// 3. Register gRPC health check service
	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	// 4. Start HTTP health endpoint (matches service pattern)
	healthPort := grpcPort + 100 // e.g., 9111 for ingestion (port 9011)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		logger.Info("starting health probe",
			"service", serviceName,
			"port", healthPort,
		)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", healthPort), mux); err != nil {
			logger.Error("health probe server failed",
				"service", serviceName,
				"error", err,
			)
		}
	}()

	// 5. Listen on gRPC port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		return fmt.Errorf("listen %s on :%d: %w", serviceName, grpcPort, err)
	}
	healthCheck.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)

	logger.Info("starting embedded gRPC service",
		"service", serviceName,
		"grpc_port", grpcPort,
		"health_port", healthPort,
	)

	// 6. Serve until context is cancelled
	errCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// 7. Wait for shutdown signal
	select {
	case err := <-errCh:
		return fmt.Errorf("%s gRPC serve: %w", serviceName, err)
	case <-ctx.Done():
		logger.Info("shutting down embedded service", "service", serviceName)
		grpcServer.GracefulStop()
		return nil
	}
}
