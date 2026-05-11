// Package main is the entry point for cognee-ingestion — the multi-modal
// data ingestion service for the Cognee semantic knowledge graph engine.
//
// Architecture: Clean Architecture (Domain → Usecase → Adapter → Infra)
// Communication: gRPC (sync), NATS JetStream (async events)
// Storage: PostgreSQL (metadata), MinIO (files), NATS (events)
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/infra/config"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/infra/server"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/infra/telemetry"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	// 2. Initialize telemetry
	logger := telemetry.NewLogger(cfg.Telemetry.LogLevel, cfg.Telemetry.LogFormat)
	logger.Info("cognee-ingestion starting",
		"service", cfg.Service.Name,
		"version", cfg.Service.Version,
	)

	// 3. Initialize tracer (optional)
	tp := telemetry.NewTracerProvider(cfg.Telemetry.OTelEndpoint, cfg.Service.Name, logger)
	tracerShutdown, err := tp.Init(context.Background())
	if err != nil {
		logger.Error("tracer init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := tracerShutdown(context.Background()); err != nil {
			logger.Error("tracer shutdown failed", "error", err)
		}
	}()

	// 4. Create server
	srv := server.New(cfg, logger)

	// 5. Wire domain services
	// In production, this is replaced by Wire-generated code:
	//   app, err := wire.InitializeApp(ctx)
	//
	// For now, the gRPC server starts with health checks only.
	// Service registration happens after adapter layer is fully wired.
	//
	// Example of manual wiring:
	//   pool, _ := pgxpool.New(ctx, cfg.Postgres.URL)
	//   dsRepo := postgres.NewDatasetRepo(pool)
	//   itemRepo := postgres.NewDataItemRepo(pool)
	//   minioClient, _ := minio.New(cfg.MinIO.Endpoint, &minio.Options{...})
	//   fileStore := storage.NewMinIOAdapter(minioClient, cfg.MinIO.Bucket, logger)
	//   natsConn, _ := nats.Connect(cfg.NATS.URL)
	//   js, _ := natsConn.JetStream()
	//   eventPub := event.NewNATSPublisher(js, logger)
	//   extractorReg := extractor.NewRegistry()
	//   hashComp := hash.NewSHA256Computer()
	//   scraper := scraper.NewHTTPScraper(30*time.Second, logger)
	//
	//   fileUC := usecase.NewIngestFileUseCase(dsRepo, itemRepo, fileStore, extractorReg, eventPub, hashComp, logger)
	//   textUC := usecase.NewIngestTextUseCase(dsRepo, itemRepo, eventPub, logger)
	//   urlUC := usecase.NewIngestURLUseCase(dsRepo, itemRepo, scraper, eventPub, logger)
	//   dsUC := usecase.NewManageDatasetUseCase(dsRepo, itemRepo, fileStore, logger)
	//
	//   handler := grpc.NewHandler(fileUC, textUC, urlUC, dsUC, logger)
	//   Register handler on srv.GRPCServer()

	srv.SetServingStatus("cognee.ingestion.v1.CogneeIngestionService", true)

	// 6. Signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 7. Start server (blocking)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// 8. Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			logger.Error("server error", "error", err)
		}
	}

	// 9. Graceful shutdown
	srv.Shutdown()
	logger.Info("cognee-ingestion stopped")
}
