package server

import (
	"log"
	"net"

	// "google.golang.org/grpc"
	// "google.golang.org/grpc/reflection"
	adapter "vnp-memory/services/memobase-engine/internal/adapter/grpc"
)

type GRPCServer struct {
	port    int
	handler *adapter.EngineHandler
}

// NewGRPCServer initializes the gRPC server.
func NewGRPCServer(port int, handler *adapter.EngineHandler) *GRPCServer {
	return &GRPCServer{
		port:    port,
		handler: handler,
	}
}

// Start runs the gRPC server
func (s *GRPCServer) Start() error {
	log.Printf("Starting gRPC server on port %d", s.port)
	// lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	// grpcServer := grpc.NewServer()
	// pb.RegisterMemobaseEngineServiceServer(grpcServer, s.handler)
	// return grpcServer.Serve(lis)
	return nil
}
