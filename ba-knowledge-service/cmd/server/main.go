package main

import (
	"fmt"
	"log"
	"net"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/server"
	knowledgepb "github.com/blcvn/backend/services/proto/knowledge"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize Knowledge Service dependencies

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 50053)) // Port 50053
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	knowledgeServer := server.NewKnowledgeServer()
	knowledgepb.RegisterKnowledgeServiceServer(s, knowledgeServer)

	// Enable reflection
	reflection.Register(s)

	log.Printf("Knowledge Service listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
