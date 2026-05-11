package grpc

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"vnp-memory/services/cognee-search/internal/infrastructure/config"
	"vnp-memory/services/cognee-search/internal/infrastructure/grpc/pb"
)

type Server struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	handler    pb.CogneeSearchServiceServer
}

func NewServer(cfg *config.Config, handler pb.CogneeSearchServiceServer) *Server {
	return &Server{
		cfg:        cfg,
		grpcServer: grpc.NewServer(),
		handler:    handler,
	}
}

func (s *Server) Start() error {
	// Register service
	// In a real implementation we would call:
	// pb.RegisterCogneeSearchServiceServer(s.grpcServer, s.handler)
	
	addr := fmt.Sprintf(":%d", s.cfg.GRPC.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	log.Printf("gRPC server listening at %v", lis.Addr())
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	if s.grpcServer != nil {
		log.Println("Gracefully stopping gRPC server...")
		s.grpcServer.GracefulStop()
	}
}
