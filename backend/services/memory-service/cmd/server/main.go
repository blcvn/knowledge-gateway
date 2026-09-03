// Package main — memory-service entry point.
//
// Merged services: memobase-context + memobase-engine + memobase-ingestion + memobase-pipeline +
//                  zep-user + zep-thread + zep-memory + zep-search + zep-graph + zep-admin +
//                  sm-memory + sm-document + sm-profile
// (MERGE-P2-T3)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/shared/pkg/forward"
	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"

	memgrpc "vnp-memory/services/memory-service/internal/adapter/grpc"
	zepnoop "vnp-memory/services/memory-service/internal/adapter/zep"
	"vnp-memory/services/memory-service/internal/infra/pgvector"
	ucmb "vnp-memory/services/memory-service/internal/usecase/memobase"
	ucsm "vnp-memory/services/memory-service/internal/usecase/sm"
	uczep "vnp-memory/services/memory-service/internal/usecase/zep"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := loadConfig()

	telemetry.InitLogger(cfg.logLevel)
	slog.Info("Starting memory-service",
		"grpc_port", cfg.grpcPort,
		"zep_enabled", cfg.zepEnabled,
	)

	if st, err := telemetry.InitProvider(ctx, "memory-service"); err == nil {
		defer func() { _ = st(context.Background()) }()
	}

	// ─── PostgreSQL ────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.databaseURL)
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

	// ─── Infra Repos ───────────────────────────────────────────────────────
	blobRepo := pgvector.NewBlobRepo(pool)
	profileRepo := pgvector.NewProfileRepo(pool)
	eventRepo := pgvector.NewEventRepo(pool)
	smMemRepo := pgvector.NewSMMemoryRepo(pool)
	smDocRepo := pgvector.NewSMDocumentRepo(pool)

	// ─── Memobase Usecases ─────────────────────────────────────────────────
	mbIngest := ucmb.NewIngestUseCase(blobRepo, nil, nil, nil)
	mbContext := ucmb.NewContextUseCase(blobRepo, profileRepo, eventRepo, nil)

	// ─── Zep Client ────────────────────────────────────────────────────────
	zepClient := &zepnoop.NoopClient{} // Default: no-op
	// TODO: wire real Zep SDK client when ZEP_API_KEY is set

	zUser := uczep.NewUserUseCase(zepClient, cfg.zepEnabled)
	zMem := uczep.NewMemoryUseCase(zepClient, cfg.zepEnabled)
	zGraph := uczep.NewGraphUseCase(zepClient, cfg.zepEnabled)

	// ─── SM Usecases ───────────────────────────────────────────────────────
	smMem := ucsm.NewMemoryUseCase(smMemRepo, nil)
	smDoc := ucsm.NewDocumentUseCase(smDocRepo)

	// ─── Handler + Router ──────────────────────────────────────────────────
	handler := memgrpc.NewMemoryHandler(mbIngest, mbContext, zUser, zMem, zGraph, smMem, smDoc)
	logger := slog.Default()
	router := forward.NewRouter(logger)
	memgrpc.RegisterRoutes(router, handler)

	// ─── gRPC Server ────────────────────────────────────────────────────────
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)
	forward.RegisterForwardService(grpcServer, router)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("memory-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// ─── HTTP Health ────────────────────────────────────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","service":"memory-service"}`))
		})
		addr := fmt.Sprintf(":%d", cfg.healthPort)
		slog.Info("HTTP health server", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("health server failed", "error", err)
		}
	}()

	// ─── Serve ──────────────────────────────────────────────────────────────
	grpcAddr := fmt.Sprintf(":%d", cfg.grpcPort)
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
	slog.Info("Shutting down memory-service...")
	grpcServer.GracefulStop()
}

type svcConfig struct {
	grpcPort    int
	healthPort  int
	databaseURL string
	zepEnabled  bool
	logLevel    string
}

func loadConfig() svcConfig {
	return svcConfig{
		grpcPort:    envInt("GRPC_PORT", 9090),
		healthPort:  envInt("HEALTH_PORT", 9130),
		databaseURL: envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/memory_service?sslmode=disable"),
		zepEnabled:  envBool("ZEP_ENABLED", false),
		logLevel:    envStr("LOG_LEVEL", "info"),
	}
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}
