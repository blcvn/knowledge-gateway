package main

import (
	"log"
	"net"

	"graphiti-pipeline/internal/infra/config"
	"graphiti-pipeline/internal/infra/server"
)

func main() {
	cfg := config.Config{}
	srv := server.NewGRPCServer(cfg)
	
	lis, err := net.Listen("tcp", ":9021")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	
	log.Println("gRPC Server listening on :9021")
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
