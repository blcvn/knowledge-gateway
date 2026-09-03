// Package main — obs-service entry point.
//
// Observability Service: aggregates metrics, traces, errors, costs,
// and service infrastructure topology.
//
// Merged from: vnp-observability, vnp-infra, sm-engine
// (MERGE-P3-T2)
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

	dockeradapter "vnp-memory/services/obs-service/internal/adapter/docker"
	obsgrpc "vnp-memory/services/obs-service/internal/adapter/grpc"
	obspg "vnp-memory/services/obs-service/internal/infra/pg"
	ucinfra "vnp-memory/services/obs-service/internal/usecase/infra"
	ucobs "vnp-memory/services/obs-service/internal/usecase/observability"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := loadConfig()
	telemetry.InitLogger(cfg.logLevel)
	slog.Info("Starting obs-service", "grpc_port", cfg.grpcPort)

	if st, err := telemetry.InitProvider(ctx, "obs-service"); err == nil {
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

	// ─── Repos ─────────────────────────────────────────────────────────────
	errorRepo := obspg.NewErrorRepo(pool)
	costRepo := obspg.NewCostRepo(pool)
	metricsRepo := obspg.NewMetricsRepo(pool)

	// ─── Optional Backends ─────────────────────────────────────────────────
	// Prometheus: noop if PROMETHEUS_URL not set (fallback to PG)
	scraper := &obspg.NoopScraper{}

	// Jaeger: noop if TRACING_ENABLED=false (default)
	traceClient := &obspg.NoopTraceClient{}

	// Docker: noop if DOCKER_ENABLED=false or socket not mounted
	var dockerClient ucinfra.DockerClientInterface
	if cfg.dockerEnabled {
		dockerClient = dockeradapter.NewClient(cfg.dockerSocket)
		slog.Info("Docker introspection enabled", "socket", cfg.dockerSocket)
	} else {
		dockerClient = dockeradapter.NewNoopClient()
		slog.Info("Docker introspection disabled (set DOCKER_ENABLED=true)")
	}

	// ─── Usecases ──────────────────────────────────────────────────────────
	obsSvc := ucobs.NewObservabilityService(scraper, traceClient, errorRepo, costRepo, metricsRepo)
	infraSvc := ucinfra.NewInfraService(dockerClient, nil, nil)

	// ─── Handler + Router ──────────────────────────────────────────────────
	handler := obsgrpc.NewObsHandler(obsSvc, infraSvc)
	logger := slog.Default()
	router := forward.NewRouter(logger)
	obsgrpc.RegisterRoutes(router, handler)

	// ─── gRPC Server ────────────────────────────────────────────────────────
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)
	forward.RegisterForwardService(grpcServer, router)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("obs-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// ─── HTTP Health ────────────────────────────────────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","service":"obs-service"}`))
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
	slog.Info("Shutting down obs-service...")
	grpcServer.GracefulStop()
}

type svcConfig struct {
	grpcPort      int
	healthPort    int
	databaseURL   string
	logLevel      string
	dockerEnabled bool
	dockerSocket  string
}

func loadConfig() svcConfig {
	return svcConfig{
		grpcPort:      envInt("GRPC_PORT", 9090),
		healthPort:    envInt("HEALTH_PORT", 9170),
		databaseURL:   envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/obs_service?sslmode=disable"),
		logLevel:      envStr("LOG_LEVEL", "info"),
		dockerEnabled: envBool("DOCKER_ENABLED", false),
		dockerSocket:  envStr("DOCKER_SOCKET", "/var/run/docker.sock"),
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
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
