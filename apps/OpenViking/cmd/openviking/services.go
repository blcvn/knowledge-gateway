package main

import (
	"context"

	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/config"
	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/embed"
)

func startAdminService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "ov-admin", cfg.Services.AdminPort)
}

func startCryptoService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "ov-crypto", cfg.Services.CryptoPort)
}

func startFSService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "ov-fs", cfg.Services.FSPort)
}

func startResourceService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "ov-resource", cfg.Services.ResourcePort)
}

func startSearchService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "ov-search", cfg.Services.SearchPort)
}

func startSessionService(ctx context.Context, cfg *config.Config) error {
	return embed.StartGRPCService(ctx, "ov-session", cfg.Services.SessionPort)
}
