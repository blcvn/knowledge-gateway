package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Enterprise-grade bootstrap for vnp-gateway
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Initializing vnp-gateway at enterprise-grade...")
	
	grpcServer := grpc.NewServer()
	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		http.ListenAndServe(":9199", mux)
	}()

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	healthCheck.SetServingStatus("vnp-gateway", grpc_health_v1.HealthCheckResponse_SERVING)
	
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gracefully...")
	grpcServer.GracefulStop()
}
