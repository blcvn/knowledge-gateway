package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/apps/supermemory/internal/config"
	"github.com/vnp-community/vnp-memory/apps/supermemory/internal/supervisor"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	sup := supervisor.New()

	// Phase 0 — Platform Layer
	sup.Register(supervisor.Task{Name: "sm-auth", Phase: 0, StartFn: func(ctx context.Context) error { return startAuthService(ctx, cfg) }})
	sup.Register(supervisor.Task{Name: "sm-analytics", Phase: 0, StartFn: func(ctx context.Context) error { return startAnalyticsService(ctx, cfg) }})

	// Phase 1 — Data Layer
	sup.Register(supervisor.Task{Name: "sm-project", Phase: 1, StartFn: func(ctx context.Context) error { return startProjectService(ctx, cfg) }})
	sup.Register(supervisor.Task{Name: "sm-profile", Phase: 1, StartFn: func(ctx context.Context) error { return startProfileService(ctx, cfg) }})

	// Phase 2 — Intelligence Layer
	sup.Register(supervisor.Task{Name: "sm-engine", Phase: 2, StartFn: func(ctx context.Context) error { return startEngineService(ctx, cfg) }})
	sup.Register(supervisor.Task{Name: "sm-document", Phase: 2, StartFn: func(ctx context.Context) error { return startDocumentService(ctx, cfg) }})
	sup.Register(supervisor.Task{Name: "sm-memory", Phase: 2, StartFn: func(ctx context.Context) error { return startMemoryService(ctx, cfg) }})
	sup.Register(supervisor.Task{Name: "sm-search", Phase: 2, StartFn: func(ctx context.Context) error { return startSearchService(ctx, cfg) }})

	// Phase 3 — Integrations
	sup.Register(supervisor.Task{Name: "sm-connector", Phase: 3, StartFn: func(ctx context.Context) error { return startConnectorService(ctx, cfg) }})
	sup.Register(supervisor.Task{Name: "sm-mcp", Phase: 3, StartFn: func(ctx context.Context) error { return startMCPService(ctx, cfg) }})

	// Phase 4 — Gateway & Health
	sup.Register(supervisor.Task{Name: "vnp-gateway", Phase: 4, StartFn: func(ctx context.Context) error { return startGateway(ctx, cfg) }})
	sup.Register(supervisor.Task{Name: "health-aggregator", Phase: 4, StartFn: func(ctx context.Context) error { return startHealthAggregator(ctx, cfg) }})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Starting Supermemory monolith supervisor...")
	if err := sup.Start(ctx); err != nil {
		log.Fatalf("Failed to start supervisor: %v", err)
	}

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Interrupt signal received. Shutting down gracefully...")
	cancel() // Cancel context to stop all supervisor tasks
}
