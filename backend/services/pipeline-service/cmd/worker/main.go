// Package main — pipeline-service worker binary.
//
// Background Redis queue processor consuming tasks:
//   - index_prd: index PRD into knowledge graph
//   - gen_outline: generate outline from PRD content
//   - graphiti.ingest: ingest content into Graphiti kg-service
//   - cognee.cognify: trigger Cognee cognification pipeline
//   - memobase.flush: flush memobase user buffer
//
// Merged from: ba-knowledge-worker + vnp-pipelines worker logic
// (MERGE-P3-T1)
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"vnp-memory/shared/pkg/telemetry"

	workeradapter "vnp-memory/services/pipeline-service/internal/adapter/worker"
	"vnp-memory/services/pipeline-service/internal/infra/config"
	"vnp-memory/services/pipeline-service/internal/infra/pg"
	redisqueue "vnp-memory/services/pipeline-service/internal/infra/redis"
	uck "vnp-memory/services/pipeline-service/internal/usecase/knowledge"
	ucp "vnp-memory/services/pipeline-service/internal/usecase/pipeline"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()
	telemetry.InitLogger(cfg.LogLevel)
	slog.Info("Starting pipeline-service worker",
		"concurrency", cfg.WorkerConcurrency,
		"redis_addr", cfg.RedisAddr,
		"redis_db", cfg.RedisDB,
	)

	// ─── PostgreSQL ────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL")

	// ─── Repos ─────────────────────────────────────────────────────────────
	jobRepo := pg.NewJobRepo(pool)
	workerRepo := pg.NewWorkerRepo(pool)
	prdRepo := pg.NewPRDRepo(pool)
	outlineRepo := pg.NewOutlineRepo(pool)

	// ─── Usecases ──────────────────────────────────────────────────────────
	indexUC := uck.NewIndexUseCase(prdRepo, outlineRepo)
	pipelineUC := ucp.NewPipelineUseCase(jobRepo, workerRepo, nil)

	// ─── Redis Consumer ────────────────────────────────────────────────────
	redisCfg := redisqueue.RedisConfig{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	consumer := redisqueue.NewConsumer(redisCfg, cfg.WorkerConcurrency)

	// ─── Register Handlers ─────────────────────────────────────────────────
	registry := workeradapter.NewRegistry(consumer, indexUC, pipelineUC, slog.Default())
	registry.RegisterAll()

	// ─── Start Consumer ────────────────────────────────────────────────────
	workerDone := make(chan error, 1)
	go func() {
		slog.Info("Pipeline worker consuming tasks...")
		workerDone <- consumer.Start(ctx)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("Shutting down pipeline worker...")
		cancel()
	case err := <-workerDone:
		if err != nil {
			slog.Error("Worker exited", "error", err)
			os.Exit(1)
		}
	}
	slog.Info("Pipeline worker stopped")
}
