// Package main — pipeline-service server binary.
//
// Provides gRPC ForwardService for pipeline status, job management,
// and knowledge CRUD endpoints.
//
// Merged from: vnp-pipelines, ba-knowledge-service
// (MERGE-P3-T1)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/shared/pkg/forward"
	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"

	pgrpc "vnp-memory/services/pipeline-service/internal/adapter/grpc"
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
	slog.Info("Starting pipeline-service server", "grpc_port", cfg.GRPCPort)

	if st, err := telemetry.InitProvider(ctx, "pipeline-service"); err == nil {
		defer func() { _ = st(context.Background()) }()
	}

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

	// ─── Queue (for enqueuing tasks from server) ───────────────────────────
	redisCfg := redisqueue.RedisConfig{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	queue := redisqueue.NewQueue(redisCfg)

	// ─── Usecases ──────────────────────────────────────────────────────────
	pipelineUC := ucp.NewPipelineUseCase(jobRepo, workerRepo, nil)
	knowledgeUC := uck.NewKnowledgeUseCase(prdRepo, outlineRepo, queue, nil)
	indexUC := uck.NewIndexUseCase(prdRepo, outlineRepo)

	// ─── Handler + Router ──────────────────────────────────────────────────
	handler := pgrpc.NewPipelineHandler(pipelineUC, knowledgeUC, indexUC)
	logger := slog.Default()
	router := forward.NewRouter(logger)
	pgrpc.RegisterRoutes(router, handler)

	// ─── gRPC Server ────────────────────────────────────────────────────────
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)
	forward.RegisterForwardService(grpcServer, router)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("pipeline-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// ─── HTTP Health ────────────────────────────────────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","service":"pipeline-service"}`))
		})
		addr := fmt.Sprintf(":%d", cfg.HealthPort)
		slog.Info("HTTP health server", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("health server failed", "error", err)
		}
	}()

	// ─── Serve ──────────────────────────────────────────────────────────────
	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("listen failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}
	go func() {
		slog.Info("gRPC ForwardService listening", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC serve failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down pipeline-service server...")
	grpcServer.GracefulStop()
}
