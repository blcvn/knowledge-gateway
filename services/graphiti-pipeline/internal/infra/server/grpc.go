package server

import (
	"graphiti-pipeline/internal/infra/config"
	"google.golang.org/grpc"
	grpc_adapter "graphiti-pipeline/internal/adapter/grpc"
)

func NewGRPCServer(cfg config.Config) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// otelgrpc.UnaryServerInterceptor(),
			// grpc_recovery.UnaryServerInterceptor(),
			// grpc_logging.UnaryServerInterceptor(),
			grpc_adapter.TenantExtractorInterceptor(),
		),
	)
	return server
}
