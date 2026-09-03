// Package main is the entry point for the graphiti monolith app.
//
// It orchestrates all embedded graphiti services (store, knowledge, ingestion,
// search, pipeline) and the gateway in a single process via the supervisor
// pattern.
//
// Architecture:
//
//	Client → [Gateway :8080] → gRPC localhost → [store      :9024]
//	                                          → [knowledge  :9023]
//	                                          → [ingestion  :9021]
//	                                          → [search     :9022]
//	                                          → [pipeline   :9025]
//
// Services communicate via gRPC localhost + NATS, identical to microservice
// deployment but within a single binary.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/apps/graphiti/internal/config"
	"github.com/vnp-community/vnp-memory/apps/graphiti/internal/supervisor"
)

func main() {
	// Structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 1. Load unified configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.Error("configuration validation failed", "error", err)
		os.Exit(1)
	}

	logger.Info("graphiti-app starting",
		"rest_port", cfg.Server.RESTPort,
		"mcp_port", cfg.Server.MCPPort,
		"health_port", cfg.Server.HealthPort,
		"store_port", cfg.Services.StorePort,
		"knowledge_port", cfg.Services.KnowledgePort,
		"ingestion_port", cfg.Services.IngestionPort,
		"search_port", cfg.Services.SearchPort,
		"pipeline_port", cfg.Services.PipelinePort,
		"log_level", cfg.Server.LogLevel,
	)

	// 2. Inject config as ENV vars for embedded services
	cfg.SetServiceEnvVars()

	// 3. Create supervisor
	sv := supervisor.New(logger)

	// 4. Register Phase 0 (Data): Store — must start first
	sv.Register(supervisor.ServiceSpec{
		Name:  "graphiti-store",
		Phase: supervisor.PhaseData,
		Port:  cfg.Services.StorePort,
		StartFn: func(ctx context.Context) error {
			return startStoreService(ctx, cfg)
		},
	})

	// 5. Register Phase 1 (Intelligence): Knowledge — depends on store
	sv.Register(supervisor.ServiceSpec{
		Name:  "graphiti-knowledge",
		Phase: supervisor.PhaseIntelligence,
		Port:  cfg.Services.KnowledgePort,
		StartFn: func(ctx context.Context) error {
			return startKnowledgeService(ctx, cfg)
		},
	})

	// 6. Register Phase 2 (Application): Ingestion, Search, Pipeline
	sv.Register(supervisor.ServiceSpec{
		Name:  "graphiti-ingestion",
		Phase: supervisor.PhaseApplication,
		Port:  cfg.Services.IngestionPort,
		StartFn: func(ctx context.Context) error {
			return startIngestionService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "graphiti-search",
		Phase: supervisor.PhaseApplication,
		Port:  cfg.Services.SearchPort,
		StartFn: func(ctx context.Context) error {
			return startSearchService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "graphiti-pipeline",
		Phase: supervisor.PhaseApplication,
		Port:  cfg.Services.PipelinePort,
		StartFn: func(ctx context.Context) error {
			return startPipelineService(ctx, cfg)
		},
	})

	// 7. Register Phase 3 (Gateway): starts after all services are ready
	sv.Register(supervisor.ServiceSpec{
		Name:  "vnp-gateway",
		Phase: supervisor.PhaseGateway,
		Port:  cfg.Server.RESTPort,
		StartFn: func(ctx context.Context) error {
			return startGateway(ctx, cfg)
		},
	})

	// 8. Start aggregated health server (independent of supervisor lifecycle)
	go startHealthServer(cfg, sv)

	// 9. Start all with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.Info("graphiti-app starting all services...")
	if err := sv.StartAll(ctx); err != nil {
		logger.Error("supervisor error", "error", err)
		os.Exit(1)
	}

	// 10. Graceful shutdown
	sv.Shutdown(cfg.Server.ShutdownTimeout)
	logger.Info("graphiti-app stopped")
}
