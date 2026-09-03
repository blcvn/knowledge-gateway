package embed

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// StartGRPCService is a generic bootstrap function that replicates the common
// service startup pattern.
func StartGRPCService(ctx context.Context, serviceName string, grpcPort int) error {
	grpcServer := grpc.NewServer()
	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	healthPort := grpcPort + 100
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Printf("Starting health probe for %s on port %d", serviceName, healthPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", healthPort), mux); err != nil {
			log.Printf("Health probe for %s failed: %v", serviceName, err)
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		return fmt.Errorf("listen %s on :%d: %w", serviceName, grpcPort, err)
	}
	healthCheck.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)

	log.Printf("Starting embedded gRPC service %s on port %d", serviceName, grpcPort)

	errCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("%s gRPC serve: %w", serviceName, err)
	case <-ctx.Done():
		log.Printf("Shutting down embedded service %s", serviceName)
		grpcServer.GracefulStop()
		return nil
	}
}
