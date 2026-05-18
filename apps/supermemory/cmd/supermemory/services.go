package main

import (
	"context"

	"github.com/vnp-community/vnp-memory/apps/supermemory/internal/config"
	"github.com/vnp-community/vnp-memory/apps/supermemory/internal/embed"
)

func startAuthService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-auth", cfg.Services.AuthPort)
}

func startAnalyticsService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-analytics", cfg.Services.AnalyticsPort)
}

func startProjectService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-project", cfg.Services.ProjectPort)
}

func startProfileService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-profile", cfg.Services.ProfilePort)
}

func startEngineService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-engine", cfg.Services.EnginePort)
}

func startDocumentService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-document", cfg.Services.DocumentPort)
}

func startMemoryService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-memory", cfg.Services.MemoryPort)
}

func startSearchService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-search", cfg.Services.SearchPort)
}

func startConnectorService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-connector", cfg.Services.ConnectorPort)
}

func startMCPService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "sm-mcp", cfg.Services.MCPPort)
}
