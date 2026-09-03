// Package main is the entry point for the Memobase monolith app.
//
// It loads configuration, sets up the supervisor, registers all embedded
// services in their correct startup phases, and blocks until termination.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/apps/memobase/internal/config"
	"github.com/vnp-community/vnp-memory/apps/memobase/internal/supervisor"
)

var version = "dev" // Overridden by build flags

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting memobase monolith", "version", version)

	// Load and validate unified configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Bridge: Set config as ENV vars so embedded services can read them
	cfg.SetServiceEnvVars()

	// Initialize supervisor
	sv := supervisor.New(logger)

	// Register services in phase order

	// Phase 0 (Data): Ingestion
	sv.Register(supervisor.ServiceSpec{
		Name:  "memobase-ingestion",
		Phase: supervisor.PhaseData,
		Port:  cfg.Services.IngestionPort,
		StartFn: func(ctx context.Context) error {
			return startIngestionService(ctx, cfg)
		},
	})

	// Phase 1 (Intelligence): Engine
	sv.Register(supervisor.ServiceSpec{
		Name:  "memobase-engine",
		Phase: supervisor.PhaseIntelligence,
		Port:  cfg.Services.EnginePort,
		StartFn: func(ctx context.Context) error {
			return startEngineService(ctx, cfg)
		},
	})

	// Phase 2 (Application): Context and Pipeline
	sv.Register(supervisor.ServiceSpec{
		Name:  "memobase-context",
		Phase: supervisor.PhaseApplication,
		Port:  cfg.Services.ContextPort,
		StartFn: func(ctx context.Context) error {
			return startContextService(ctx, cfg)
		},
	})

	sv.Register(supervisor.ServiceSpec{
		Name:  "memobase-pipeline",
		Phase: supervisor.PhaseApplication,
		Port:  cfg.Services.PipelinePort,
		StartFn: func(ctx context.Context) error {
			return startPipelineService(ctx, cfg)
		},
	})

	// Phase 3 (Gateway): REST / MCP proxy
	sv.Register(supervisor.ServiceSpec{
		Name:  "vnp-gateway",
		Phase: supervisor.PhaseGateway,
		Port:  cfg.Server.RESTPort, // Gateway handles REST+MCP
		StartFn: func(ctx context.Context) error {
			return startGateway(ctx, cfg)
		},
	})

	// Start health server in background
	go startHealthServer(cfg, sv)

	// Handle graceful shutdown via signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received termination signal", "signal", sig)
		cancel()
	}()

	// Start all services; blocks until ctx cancelled or fatal error
	if err := sv.StartAll(ctx); err != nil {
		logger.Error("supervisor failed", "error", err)
		// Try to shut down gracefully even on failure
		sv.Shutdown(cfg.Server.ShutdownTimeout)
		os.Exit(1)
	}

	// Normal shutdown
	sv.Shutdown(cfg.Server.ShutdownTimeout)
	logger.Info("memobase monolith exited cleanly")
}
