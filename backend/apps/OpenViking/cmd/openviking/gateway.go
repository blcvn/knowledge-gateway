package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/config"
	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/gateway"
)

// startGateway starts the gateway service embedding
func startGateway(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	// Configure the gateway to route to the localhost ports
	for name, port := range cfg.GatewayServicesMap() {
		// Use environment variables to point gateway to these services
		envKey := fmt.Sprintf("SERVICES_%s", name) // Adjust according to gateway's config parsing
		os.Setenv(envKey, "127.0.0.1"+port)
	}

	// Set Gateway REST Port
	os.Setenv("SERVER_REST_PORT", fmt.Sprintf("%d", cfg.Server.RESTPort))
	os.Setenv("SERVER_MCP_PORT", fmt.Sprintf("%d", cfg.Server.MCPPort))

	return gateway.Run(ctx, logger)
}
