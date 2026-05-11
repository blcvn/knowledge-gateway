package server

import (
	"fmt"
	"log/slog"
	"net"

	grpcHandler "github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// GRPCServer wraps the standard grpc server.
type GRPCServer struct {
	server  *grpc.Server
	handler *grpcHandler.Handler
	logger  *slog.Logger
	port    int
}

// NewGRPCServer creates a new gRPC server.
func NewGRPCServer(handler *grpcHandler.Handler, logger *slog.Logger) *GRPCServer {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor(logger),
			LoggingInterceptor(logger),
		),
	}
	s := grpc.NewServer(opts...)
	
	// Normally we would register the handler here
	// pb.RegisterGraphitiStoreServer(s, handler)

	// Health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)

	return &GRPCServer{
		server:  s,
		handler: handler,
		logger:  logger.With("component", "grpc_server"),
		port:    50051,
	}
}

// Start runs the gRPC server.
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("gRPC server listening", "port", s.port)
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}

// Stop gracefully stops the server.
func (s *GRPCServer) Stop() {
	s.logger.Info("stopping gRPC server gracefully")
	s.server.GracefulStop()
}
