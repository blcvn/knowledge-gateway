package bootstrap

import (
	"log/slog"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
	"github.com/vnp-community/vnp-memory/apps/memory/internal/config"
)

func Platform(bus *bus.GRPCBus, infra *Infra, nats *bus.NATSBus, cfg *config.Config, logger *slog.Logger) {
	logger.Info("Bootstrapping Platform services...")
	// vnp-admin, vnp-event, vnp-search-hub will be wired here
}
