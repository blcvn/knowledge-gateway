package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/config"
	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/supervisor"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	sv := supervisor.New(logger)

	// Phase 1: Domain Services
	sv.Register(supervisor.ServiceSpec{
		Name:  "ov-admin",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			return startAdminService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "ov-crypto",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			return startCryptoService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "ov-fs",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			return startFSService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "ov-resource",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			return startResourceService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "ov-search",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			return startSearchService(ctx, cfg)
		},
	})
	sv.Register(supervisor.ServiceSpec{
		Name:  "ov-session",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			return startSessionService(ctx, cfg)
		},
	})

	// Phase 2: Gateway
	sv.Register(supervisor.ServiceSpec{
		Name:  "gateway",
		Phase: 2,
		StartFn: func(ctx context.Context) error {
			return startGateway(ctx, cfg, logger)
		},
	})

	// Phase 3: Health Aggregation
	sv.Register(supervisor.ServiceSpec{
		Name:  "health-aggregation",
		Phase: 3,
		StartFn: func(ctx context.Context) error {
			return startHealthAggregation(ctx, cfg, logger)
		},
	})

	if err := sv.Run(); err != nil {
		logger.Error("supervisor failed", "error", err)
		os.Exit(1)
	}
}
