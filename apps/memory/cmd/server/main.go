package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bootstrap"
	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
	"github.com/vnp-community/vnp-memory/apps/memory/internal/config"
	embedui "github.com/vnp-community/vnp-memory/apps/memory/internal/ui"
	gwServer "github.com/vnp-community/vnp-memory/gateway/infra/server"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Server.LogLevel),
	}))
	slog.SetDefault(logger)

	logger.Info("Starting VNP Memory Monolith",
		"rest_port", cfg.Server.RESTPort,
		"mcp_port", cfg.Server.MCPPort,
		"health_port", cfg.Server.HealthPort,
	)

	// ── Shared Infrastructure ─────────────────────────
	infra, err := bootstrap.NewInfra(cfg, logger)
	if err != nil {
		logger.Error("Failed to init infra", "error", err)
		os.Exit(1)
	}
	defer infra.Close()

	// ── In-Process Communication Bus ──────────────────
	grpcBus := bus.NewGRPCBus()

	natsCfg := bus.NATSConfig{
		Mode:     cfg.NATS.Mode,
		URL:      cfg.NATS.URL,
		StoreDir: cfg.NATS.StoreDir,
	}
	natsBus, err := bus.NewNATSBus(natsCfg, logger)
	if err != nil {
		logger.Error("Failed to init NATS", "error", err)
		os.Exit(1)
	}
	defer natsBus.Close()

	// ── Bootstrap: Platform Services ──────────────────
	bootstrap.Platform(grpcBus, infra, natsBus, cfg, logger)

	// ── Bootstrap: Engine Services ────────────────────
	bootstrap.Cognee(grpcBus, infra, natsBus, logger)
	bootstrap.Graphiti(grpcBus, infra, natsBus, logger)
	bootstrap.Memobase(grpcBus, infra, natsBus, logger)
	bootstrap.OpenViking(grpcBus, infra, natsBus, logger)
	bootstrap.Zep(grpcBus, infra, natsBus, logger)
	bootstrap.Supermemory(grpcBus, infra, natsBus, logger)

	// ── Start gRPC Bus ────────────────────────────────
	go func() {
		if err := grpcBus.Serve(); err != nil {
			logger.Error("gRPC bus serve error", "error", err)
		}
	}()

	// ── Load Embedded UI ──────────────────────────────
	spaFS, uiErr := embedui.DistFS()
	if uiErr != nil {
		logger.Warn("UI console not embedded, running API-only mode", "error", uiErr)
		spaFS = nil
	} else {
		logger.Info("UI console embedded successfully")
	}

	registry := bus.NewInProcessRegistry(grpcBus, logger)
	gw := bootstrap.Gateway(registry, infra, cfg, logger, spaFS)

	// ── Server Lifecycle ──────────────────────────────
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// REST server (main API gateway)
	restSrv := gwServer.NewHTTPServer(gw.Router, cfg.Server.RESTPort, logger)
	go func() {
		if err := restSrv.Start(ctx); err != nil {
			logger.Error("REST server error", "error", err)
		}
	}()

	// MCP server (AI agent interface)
	mcpSrv := gwServer.NewHTTPServer(gw.MCP.Handler(), cfg.Server.MCPPort, logger)
	go func() {
		if err := mcpSrv.Start(ctx); err != nil {
			logger.Error("MCP server error", "error", err)
		}
	}()

	// Health server (observability)
	healthSrv := newHealthServer(registry, cfg.Server.HealthPort, logger)
	go func() {
		if err := healthSrv.Start(ctx); err != nil {
			logger.Error("Health server error", "error", err)
		}
	}()

	logger.Info("VNP Memory Monolith running",
		"rest", fmt.Sprintf(":%d", cfg.Server.RESTPort),
		"mcp", fmt.Sprintf(":%d", cfg.Server.MCPPort),
		"health", fmt.Sprintf(":%d", cfg.Server.HealthPort),
	)

	<-ctx.Done()
	logger.Info("Shutting down...")

	// Graceful shutdown order: servers → NATS drain → gRPC → DB pools
	grpcBus.Stop()
	logger.Info("VNP Memory Monolith stopped")
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// newHealthServer creates a health check server that reports status of all 35 services.
func newHealthServer(registry *bus.InProcessRegistry, port int, logger *slog.Logger) *gwServer.HTTPServer {
	allServices := []string{
		// Cognee (3)
		"cognee-ingestion", "cognee-cognify", "cognee-search",
		// Graphiti (4)
		"graphiti-ingestion", "graphiti-search", "graphiti-knowledge", "graphiti-store",
		// Memobase (3)
		"memobase-ingestion", "memobase-engine", "memobase-context",
		// OpenViking (6)
		"ov-fs", "ov-search", "ov-session", "ov-resource", "ov-crypto", "ov-admin",
		// Zep (6)
		"zep-user", "zep-thread", "zep-memory", "zep-graph", "zep-search", "zep-admin",
		// Supermemory (9)
		"sm-document", "sm-memory", "sm-search", "sm-profile", "sm-connector",
		"sm-mcp", "sm-auth", "sm-analytics", "sm-project",
		// Platform (4)
		"vnp-admin", "vnp-event", "vnp-search-hub", "vnp-platform",
	}

	mux := http.NewServeMux()

	// GET /healthz — list all 35 services with status
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		healthy := 0
		services := make([]map[string]string, 0, len(allServices))
		for _, svc := range allServices {
			ok, _ := registry.HealthCheck(svc)
			status := "unhealthy"
			if ok {
				status = "healthy"
				healthy++
			}
			services = append(services, map[string]string{"name": svc, "status": status})
		}
		overall := "healthy"
		if healthy < len(allServices) {
			overall = "degraded"
		}

		fmt.Fprintf(w, `{"status":"%s","healthy":%d,"total":%d,"services":[`, overall, healthy, len(allServices))
		for i, svc := range services {
			if i > 0 {
				w.Write([]byte(","))
			}
			fmt.Fprintf(w, `{"name":"%s","status":"%s"}`, svc["name"], svc["status"])
		}
		w.Write([]byte("]}\n"))
	})

	// GET /readyz — simple readiness check
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ready"}` + "\n"))
	})

	return gwServer.NewHTTPServer(mux, port, logger)
}
