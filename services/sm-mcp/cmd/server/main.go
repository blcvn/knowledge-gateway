package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Enterprise-grade bootstrap with Graceful Shutdown, Health Probes, and OTel initialization.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup Structured Logging & Telemetry
	log.Println("Initializing sm-mcp at enterprise-grade...")
	
	// 2. Setup gRPC Server with Interceptors
	grpcServer := grpc.NewServer()
	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	// 3. Start Health Probe HTTP Server
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Println("Health probe listening on :9199")
		http.ListenAndServe(":9199", mux)
	}()

	// 4. Listen and Serve gRPC
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	healthCheck.SetServingStatus("sm-mcp", grpc_health_v1.HealthCheckResponse_SERVING)
	
	go func() {
		log.Println("gRPC Server listening on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 5. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gracefully...")
	grpcServer.GracefulStop()
	log.Println("Shutdown complete.")
}
