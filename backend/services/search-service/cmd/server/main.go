// Package main — search-service entry point.
//
// Merged from: vnp-search-hub + ov-search + sm-search + sm-connector + sm-mcp
// (MERGE-P2-T4)
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

	searchgrpc "vnp-memory/services/search-service/internal/adapter/grpc"
	kgadapter "vnp-memory/services/search-service/internal/adapter/kg"
	memadapter "vnp-memory/services/search-service/internal/adapter/memory"
	storeadapter "vnp-memory/services/search-service/internal/adapter/storage"
	searchpg "vnp-memory/services/search-service/internal/infra/pgvector"
	ucconn "vnp-memory/services/search-service/internal/usecase/connector"
	ucorch "vnp-memory/services/search-service/internal/usecase/orchestrator"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := loadConfig()
	telemetry.InitLogger(cfg.logLevel)
	slog.Info("Starting search-service", "grpc_port", cfg.grpcPort)

	if st, err := telemetry.InitProvider(ctx, "search-service"); err == nil {
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

	// ─── Infra ─────────────────────────────────────────────────────────────
	connectorRepo := searchpg.NewConnectorRepo(pool)

	// ─── Downstream Clients ────────────────────────────────────────────────
	var kgClient ucorch.KGClientInterface
	if cfg.kgServiceURL != "" {
		kgClient = kgadapter.NewClient(cfg.kgServiceURL)
	} else {
		kgClient = kgadapter.NewNoopClient()
	}

	var memClient ucorch.MemoryClientInterface
	if cfg.memoryServiceURL != "" {
		memClient = memadapter.NewClient(cfg.memoryServiceURL)
	} else {
		memClient = memadapter.NewNoopClient()
	}

	var storeClient ucorch.StorageClientInterface
	if cfg.storageServiceURL != "" {
		storeClient = storeadapter.NewClient(cfg.storageServiceURL)
	} else {
		storeClient = storeadapter.NewNoopClient()
	}

	// ─── Usecases ──────────────────────────────────────────────────────────
	searchOrch := ucorch.NewSearchOrchestrator(kgClient, memClient, storeClient)
	connectorSvc := ucconn.NewService(connectorRepo, nil)

	// ─── Handler + Router ──────────────────────────────────────────────────
	handler := searchgrpc.NewSearchHandler(searchOrch, connectorSvc)
	logger := slog.Default()
	router := forward.NewRouter(logger)
	searchgrpc.RegisterRoutes(router, handler)

	// ─── gRPC Server ────────────────────────────────────────────────────────
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)
	forward.RegisterForwardService(grpcServer, router)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("search-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// ─── HTTP Health ────────────────────────────────────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","service":"search-service"}`))
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
	slog.Info("Shutting down search-service...")
	grpcServer.GracefulStop()
}

type svcConfig struct {
	grpcPort          int
	healthPort        int
	databaseURL       string
	logLevel          string
	kgServiceURL      string
	memoryServiceURL  string
	storageServiceURL string
}

func loadConfig() svcConfig {
	return svcConfig{
		grpcPort:          envInt("GRPC_PORT", 9090),
		healthPort:        envInt("HEALTH_PORT", 9140),
		databaseURL:       envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/search_service?sslmode=disable"),
		logLevel:          envStr("LOG_LEVEL", "info"),
		kgServiceURL:      envStr("KG_SERVICE_URL", ""),
		memoryServiceURL:  envStr("MEMORY_SERVICE_URL", ""),
		storageServiceURL: envStr("STORAGE_SERVICE_URL", ""),
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
