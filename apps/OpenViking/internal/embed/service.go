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

	"vnp-memory/pkg/telemetry"
	"vnp-memory/pkg/tenant"
)

// StartGRPCService is a generic bootstrap function that replicates the common
// service startup pattern used by all OpenViking microservices without modifying their code.
//
// Reference: services/ov-*/cmd/server/main.go
func StartGRPCService(ctx context.Context, serviceName string, grpcPort int) error {
	logger := slog.Default()

	// 1. Initialize OTel tracing
	shutdownTracer, err := telemetry.InitProvider(ctx, serviceName)
	if err != nil {
		logger.Warn("OTel init failed, continuing without tracing",
			"service", serviceName,
			"error", err,
		)
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

	// 4. Start HTTP health endpoint
	healthPort := grpcPort + 100 // e.g., 9130 for ov-admin (port 9030)
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
