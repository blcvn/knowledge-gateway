package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/infra/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // could parse from cfg.LogLevel
	}))
	slog.SetDefault(logger)

	// DB Init
	db, err := sql.Open("postgres", cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	db.SetMaxOpenConns(cfg.DBMaxConnections)

	// Normally here we use Wire to inject dependencies
	// e.g., app := wire.InitializeApp(db, cfg)

	// gRPC Server setup
	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Start gRPC
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		logger.Info("Starting gRPC server", slog.Int("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Start HTTP Probes
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"serving"}`))
	})
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Ping DB
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unready","checks":{"db":"down"}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","checks":{"db":"ok"}}`))
	})

	go func() {
		logger.Info("Starting HTTP health probes", slog.Int("port", cfg.HealthPort))
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HealthPort), nil); err != nil {
			log.Fatalf("failed to serve http health probes: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down gracefully...")
	_, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	logger.Info("Shutdown complete")
}
