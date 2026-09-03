// Package main — service start functions for embedded graphiti services.
//
// Each function wraps the generic StartGRPCService from internal/embed,
// providing service-specific configuration. These are registered as
// supervisor.ServiceSpec.StartFn entries in main.go.
package main

import (
	"context"

	"github.com/vnp-community/vnp-memory/apps/graphiti/internal/config"
	"github.com/vnp-community/vnp-memory/apps/graphiti/internal/embed"
)

// startStoreService starts the graphiti-store gRPC server.
// Phase 0 (Data) — depends only on Neo4j.
//
// Reference: services/graphiti-store/cmd/server/main.go
func startStoreService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "graphiti-store", cfg.Services.StorePort)
}

// startKnowledgeService starts the graphiti-knowledge gRPC server.
// Phase 1 (Intelligence) — depends on store + LLM provider.
//
// Reference: services/graphiti-knowledge/cmd/server/main.go
func startKnowledgeService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "graphiti-knowledge", cfg.Services.KnowledgePort)
}

// startIngestionService starts the graphiti-ingestion gRPC server.
// Phase 2 (Application) — depends on knowledge + store.
//
// Reference: services/graphiti-ingestion/cmd/server/main.go
func startIngestionService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "graphiti-ingestion", cfg.Services.IngestionPort)
}

// startSearchService starts the graphiti-search gRPC server.
// Phase 2 (Application) — depends on knowledge + store.
//
// Reference: services/graphiti-search/cmd/server/main.go
func startSearchService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "graphiti-search", cfg.Services.SearchPort)
}

// startPipelineService starts the graphiti-pipeline gRPC server.
// Phase 2 (Application) — depends on ingestion + knowledge + store.
//
// Reference: services/graphiti-pipeline/cmd/server/main.go
func startPipelineService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "graphiti-pipeline", cfg.Services.PipelinePort)
}
