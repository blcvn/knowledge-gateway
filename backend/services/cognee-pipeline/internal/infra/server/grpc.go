package server

import (
	"context"
	"fmt"
	"net"

	cogneev1 "github.com/vnp-community/vnp-memory/gateway/gen/go/cognee/v1"
	pipelinegrpc "github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/adapter/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	server  *grpc.Server
	handler *pipelinegrpc.Handler
	logger  *zap.Logger
	port    int
}

func NewGRPCServer(handler *pipelinegrpc.Handler, logger *zap.Logger) *GRPCServer {
	// Setup gRPC Server with Interceptors for Logging, Recovery, Metrics, Tracing
	opts := []grpc.ServerOption{
		// grpc.UnaryInterceptor(...)
	}
	srv := grpc.NewServer(opts...)

	cogneev1.RegisterCogneeIngestionServiceServer(srv, handler)

	// Health Check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Reflection for debugging
	reflection.Register(srv)

	return &GRPCServer{
		server:  srv,
		handler: handler,
		logger:  logger,
		port:    9011,
	}
}

func (s *GRPCServer) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Starting gRPC server", zap.Int("port", s.port))
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}

func (s *GRPCServer) Stop() {
	s.logger.Info("Gracefully stopping gRPC server")
	s.server.GracefulStop()
}
