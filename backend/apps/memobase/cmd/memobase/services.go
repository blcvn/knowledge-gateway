// Package main — service start functions for embedded memobase services.
//
// Each function wraps the generic StartGRPCService from internal/embed,
// providing service-specific configuration. These are registered as
// supervisor.ServiceSpec.StartFn entries in main.go.
package main

import (
	"context"

	"github.com/vnp-community/vnp-memory/apps/memobase/internal/config"
	"github.com/vnp-community/vnp-memory/apps/memobase/internal/embed"
)

// startIngestionService starts the memobase-ingestion gRPC server.
// Phase 0 (Data) — depends only on Postgres and NATS.
//
// Reference: services/memobase-ingestion/cmd/server/main.go
func startIngestionService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "memobase-ingestion", cfg.Services.IngestionPort)
}

// startEngineService starts the memobase-engine gRPC server.
// Phase 1 (Intelligence) — depends on ingestion + LLM.
//
// Reference: services/memobase-engine/cmd/server/main.go
func startEngineService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "memobase-engine", cfg.Services.EnginePort)
}

// startContextService starts the memobase-context gRPC server.
// Phase 2 (Application) — depends on engine.
//
// Reference: services/memobase-context/cmd/server/main.go
func startContextService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "memobase-context", cfg.Services.ContextPort)
}

// startPipelineService starts the memobase-pipeline gRPC server.
// Phase 2 (Application) — depends on engine and context.
//
// Reference: services/memobase-pipeline/cmd/server/main.go
func startPipelineService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "memobase-pipeline", cfg.Services.PipelinePort)
}
