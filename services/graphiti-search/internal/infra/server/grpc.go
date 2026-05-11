package server

import (
	"log"
	"net"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	pb "vnp-memory/services/graphiti-search/internal/adapter/grpc/pb"
)

type GRPCServer struct {
	server *grpc.Server
	addr   string
}

func NewGRPCServer(addr string, searchServer pb.GraphitiSearchServiceServer) *GRPCServer {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_prometheus.UnaryServerInterceptor,
		),
		grpc.ChainStreamInterceptor(
			grpc_prometheus.StreamServerInterceptor,
		),
	)
	pb.RegisterGraphitiSearchServiceServer(s, searchServer)
	grpc_prometheus.Register(s)
	return &GRPCServer{
		server: s,
		addr:   addr,
	}
}

func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	log.Printf("gRPC server listening on %s", s.addr)
	return s.server.Serve(lis)
}

func (s *GRPCServer) Stop() {
	s.server.GracefulStop()
}
