package server

import (
	"fmt"
	"net"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type GRPCServer struct {
	server *grpc.Server
	health *health.Server
	port   int
}

func NewGRPCServer(port int) *GRPCServer {
	logger := zap.L()

	// Setup interceptors
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		otelgrpc.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_zap.UnaryServerInterceptor(logger),
		grpc_recovery.UnaryServerInterceptor(),
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		otelgrpc.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
		grpc_zap.StreamServerInterceptor(logger),
		grpc_recovery.StreamServerInterceptor(),
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	)
	
	grpc_prometheus.Register(srv)
	
	hsrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, hsrv)
	
	return &GRPCServer{
		server: srv,
		health: hsrv,
		port:   port,
	}
}

func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	return s.server.Serve(lis)
}

func (s *GRPCServer) Stop() {
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.server.GracefulStop()
}
