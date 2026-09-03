package server

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	
	grpcAdapter "vnp-memory/services/graphiti-knowledge/adapter/grpc"
	"vnp-memory/services/graphiti-knowledge/adapter/grpc/pb"
)

func StartGRPCServer(grpcPort string, healthPort string, handler *grpcAdapter.Handler) {
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		slog.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterGraphitiKnowledgeServiceServer(grpcServer, handler)

	// Setup Health Check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start Health HTTP Endpoint
	go func() {
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		slog.Info("Starting health HTTP server", slog.String("port", healthPort))
		if err := http.ListenAndServe(":"+healthPort, nil); err != nil {
			slog.Error("failed to serve health check", slog.String("error", err.Error()))
		}
	}()

	// Start gRPC server
	go func() {
		slog.Info("Starting gRPC server", slog.String("port", grpcPort))
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", slog.String("error", err.Error()))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	slog.Info("gRPC server stopped gracefully")
}
