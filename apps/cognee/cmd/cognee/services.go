// Package main — service start functions for embedded cognee services.
//
// Each function wraps the generic StartGRPCService from internal/embed,
// providing service-specific configuration. These are registered as
// supervisor.ServiceSpec.StartFn entries in main.go.
package main

import (
	"context"

	"github.com/vnp-community/vnp-memory/apps/cognee/internal/config"
	"github.com/vnp-community/vnp-memory/apps/cognee/internal/embed"
)

// startIngestionService starts the cognee-ingestion gRPC server
// on the configured port. Follows the same bootstrap pattern as
// services/cognee-ingestion/cmd/server/main.go.
func startIngestionService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "cognee-ingestion", cfg.Services.IngestionPort)
}

// startCognifyService starts the cognee-cognify gRPC server
// on the configured port. Follows the same bootstrap pattern as
// services/cognee-cognify/cmd/server/main.go.
func startCognifyService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "cognee-cognify", cfg.Services.CognifyPort)
}

// startSearchService starts the cognee-search gRPC server
// on the configured port. Follows the same bootstrap pattern as
// services/cognee-search/cmd/server/main.go.
func startSearchService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "cognee-search", cfg.Services.SearchPort)
}
