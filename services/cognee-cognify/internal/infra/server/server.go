package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/adapter/grpc"
	grpc_adapter "github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/adapter/grpc"
)

// StartGRPCServer initializes and starts the gRPC server.
func StartGRPCServer(ctx context.Context, port int, handler grpc_adapter.CogneeCognifyServiceServer) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	
	// In a real application, we would register the generated server.
	// cogneev1.RegisterCogneeCognifyServiceServer(grpcServer, handler)
	
	healthcheck := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthcheck)
	healthcheck.SetServingStatus("cognee-cognify", healthpb.HealthCheckResponse_SERVING)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			panic(fmt.Sprintf("failed to serve: %v", err))
		}
	}()

	return grpcServer, nil
}
