// Package main — storage-service entry point.
//
// Merged service: ov-fs + ov-crypto + ov-resource + ov-session + ov-storage
// (MERGE-P1-T4)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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

	storagegrpc "vnp-memory/services/storage-service/internal/adapter/grpc"
	"vnp-memory/services/storage-service/internal/infra/localfs"
	"vnp-memory/services/storage-service/internal/infra/pg"
	fsuc "vnp-memory/services/storage-service/internal/usecase/fs"
	resourceuc "vnp-memory/services/storage-service/internal/usecase/resource"
	sessionuc "vnp-memory/services/storage-service/internal/usecase/session"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ─── Config ────────────────────────────────────────────────────────────
	cfg := loadConfig()

	// ─── Telemetry ─────────────────────────────────────────────────────────
	telemetry.InitLogger(cfg.logLevel)
	slog.Info("Starting storage-service", "grpc_port", cfg.grpcPort)

	shutdownTracer, err := telemetry.InitProvider(ctx, "storage-service")
	if err != nil {
		slog.Error("failed to initialize OTel", "error", err)
		os.Exit(1)
	}
	defer func() { _ = shutdownTracer(context.Background()) }()

	// ─── Database ──────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.databaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL")

	// ─── Infra ─────────────────────────────────────────────────────────────
	fsRepo := localfs.NewRepo(cfg.fsBaseDir)
	sessionRepo := pg.NewSessionRepo(pool)
	resourceRepo := pg.NewResourceRepo(pool)

	// ─── Usecases ──────────────────────────────────────────────────────────
	fsSvc := fsuc.NewService(fsRepo, cfg.fsBaseDir)
	sessionSvc := sessionuc.NewService(sessionRepo)
	resourceSvc := resourceuc.NewService(resourceRepo)

	// ─── Handler ───────────────────────────────────────────────────────────
	handler := storagegrpc.NewStorageHandler(fsSvc, sessionSvc, resourceSvc)

	// ─── ForwardService Router ─────────────────────────────────────────────
	logger := slog.Default()
	router := forward.NewRouter(logger)
	registerRoutes(router, handler)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)
	forward.RegisterForwardService(grpcServer, router)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)
	healthCheck.SetServingStatus("storage-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// ─── HTTP Health Server ─────────────────────────────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		addr := fmt.Sprintf(":%d", cfg.healthPort)
		slog.Info("Starting HTTP health server", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("health server failed", "error", err)
		}
	}()

	// ─── gRPC Server ────────────────────────────────────────────────────────
	grpcAddr := fmt.Sprintf(":%d", cfg.grpcPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("failed to listen", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("Starting gRPC ForwardService", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	// ─── Graceful Shutdown ──────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down storage-service...")
	grpcServer.GracefulStop()
}

// registerRoutes wires all storage-service routes.
func registerRoutes(router *forward.Router, h *storagegrpc.StorageHandler) {
	// File System (ov-fs)
	router.Handle("GET", "/v1/ov/files/*", adapt(h.ReadFile))
	router.Handle("PUT", "/v1/ov/files/*", adapt(h.WriteFile))
	router.Handle("DELETE", "/v1/ov/files/*", adapt(h.DeleteFile))
	router.Handle("GET", "/v1/ov/tree/*", adapt(h.Tree))
	router.Handle("POST", "/v1/ov/grep", adapt(h.Grep))

	// Sessions (ov-session)
	router.Handle("POST", "/v1/ov/sessions", adapt(h.CreateSession))
	router.Handle("POST", "/v1/ov/sessions/*/messages", adapt(h.AddMessage))
	router.Handle("POST", "/v1/ov/sessions/*/commit", adapt(h.CommitSession))

	// Resources (ov-resource)
	router.Handle("POST", "/v1/ov/resources/ingest", adapt(h.IngestResource))
	router.Handle("GET", "/v1/ov/resources/*", adapt(h.GetResourceStatus))
}

// adapt converts http.HandlerFunc to forward.HandlerFunc.
func adapt(h http.HandlerFunc) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		path := "/"
		if p, ok := params["__path"]; ok {
			path = p
		}
		method := "GET"
		if m, ok := params["__method"]; ok {
			method = m
		}
		u, _ := url.Parse(path)
		req, _ := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range params {
			if k[0] != '_' {
				req.SetPathValue(k, v)
			}
		}

		rw := &responseCapture{header: make(http.Header)}
		h(rw, req)

		if rw.code >= 500 {
			return rw.body.Bytes(), fmt.Errorf("HTTP %d", rw.code)
		}
		return rw.body.Bytes(), nil
	}
}

type responseCapture struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func (rc *responseCapture) Header() http.Header        { return rc.header }
func (rc *responseCapture) WriteHeader(code int)        { rc.code = code }
func (rc *responseCapture) Write(b []byte) (int, error) { return rc.body.Write(b) }

// serviceConfig holds runtime configuration.
type serviceConfig struct {
	grpcPort    int
	healthPort  int
	databaseURL string
	fsBaseDir   string
	logLevel    string
}

func loadConfig() serviceConfig {
	return serviceConfig{
		grpcPort:    envInt("GRPC_PORT", 9090),
		healthPort:  envInt("HEALTH_PORT", 9140),
		databaseURL: envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/storage_service?sslmode=disable"),
		fsBaseDir:   envStr("FS_BASE_DIR", "/data/storage"),
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
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

var _ = json.Marshal // suppress import
