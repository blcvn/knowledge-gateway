package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/adapter/event"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/adapter/repository/postgres"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/infra/config"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/infra/llm"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/infra/server"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/engine"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/ingestion"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// 1. Setup DB
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to db", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 2. Setup NATS
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	// 3. Init Repositories
	blobRepo := postgres.NewBlobRepo(dbPool)
	bufferRepo := postgres.NewBufferRepo(dbPool)
	profileRepo := postgres.NewProfileRepo(dbPool)
	gistRepo := postgres.NewGistRepo(dbPool)

	// 4. Init Infra Adapters
	bifrostClient := llm.NewBifrostClient("http://bifrost-gateway:8080")
	natsPublisher := event.NewNATSPublisher(nc)

	// 5. Init Usecases
	engineUsecase := engine.NewService(bifrostClient, profileRepo, gistRepo, natsPublisher)
	ingestUsecase := ingestion.NewService(blobRepo, bufferRepo, engineUsecase, natsPublisher)

	// 6. Init Server
	handler := server.NewIngestionHandler(ingestUsecase)
	grpcSrv := server.NewServer(cfg.GRPCPort, cfg.HealthPort, handler)

	// 7. Start Server with Graceful Shutdown
	errChan := make(chan error, 1)
	go func() {
		slog.Info("memobase-pipeline starting", "grpc_port", cfg.GRPCPort, "health_port", cfg.HealthPort)
		if err := grpcSrv.Start(); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		slog.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-sigChan:
		fmt.Printf("Received signal %v, initiating graceful shutdown...\n", sig)
		
		// 30s Grace Period as per runbook
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		grpcSrv.Stop()
		fmt.Println("Graceful shutdown completed")
	}
}
