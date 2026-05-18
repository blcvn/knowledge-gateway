// Package main is the entry point for the cognee monolith app.
//
// It orchestrates all embedded cognee services (ingestion, cognify, search) and
// the gateway in a single process via the supervisor pattern.
//
// Architecture:
//
//	Client → [Gateway :8080] → gRPC localhost → [ingestion :9011]
//	                                          → [cognify   :9012]
//	                                          → [search    :9013]
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

	"github.com/vnp-community/vnp-memory/apps/cognee/internal/config"
	"github.com/vnp-community/vnp-memory/apps/cognee/internal/supervisor"
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

	logger.Info("cognee-app starting",
		"rest_port", cfg.Server.RESTPort,
		"health_port", cfg.Server.HealthPort,
		"ingestion_port", cfg.Services.IngestionPort,
		"cognify_port", cfg.Services.CognifyPort,
		"search_port", cfg.Services.SearchPort,
		"log_level", cfg.Server.LogLevel,
	)

	// 2. Inject config as ENV vars for embedded services
	cfg.SetServiceEnvVars()

	// 3. Create supervisor
	sv := supervisor.New(logger)

	// 4. Register Phase 0: Cognee gRPC services
	sv.Register(supervisor.ServiceSpec{
		Name:  "cognee-ingestion",
		Phase: supervisor.PhaseInfra,
		Port:  cfg.Services.IngestionPort,
		StartFn: func(ctx context.Context) error {
			return startIngestionService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "cognee-cognify",
		Phase: supervisor.PhaseInfra,
		Port:  cfg.Services.CognifyPort,
		StartFn: func(ctx context.Context) error {
			return startCognifyService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "cognee-search",
		Phase: supervisor.PhaseInfra,
		Port:  cfg.Services.SearchPort,
		StartFn: func(ctx context.Context) error {
			return startSearchService(ctx, cfg)
		},
	})

	// 5. Register Phase 1: Gateway (starts after services are ready)
	sv.Register(supervisor.ServiceSpec{
		Name:  "vnp-gateway",
		Phase: supervisor.PhaseGateway,
		Port:  cfg.Server.RESTPort,
		StartFn: func(ctx context.Context) error {
			return startGateway(ctx, cfg)
		},
	})

	// 6. Start aggregated health server (independent of supervisor lifecycle)
	go startHealthServer(cfg, sv)

	// 7. Start all with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.Info("cognee-app starting all services...")
	if err := sv.StartAll(ctx); err != nil {
		logger.Error("supervisor error", "error", err)
		os.Exit(1)
	}

	// 8. Graceful shutdown
	sv.Shutdown(cfg.Server.ShutdownTimeout)
	logger.Info("cognee-app stopped")
}
