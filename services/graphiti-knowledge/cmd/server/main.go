package main

import (
	"log/slog"
	"os"
	"vnp-memory/services/graphiti-knowledge/internal/infra/config"
	"vnp-memory/services/graphiti-knowledge/internal/infra/server"
	"vnp-memory/services/graphiti-knowledge/internal/infra/wire"
)

func main() {
	// Setup structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Warn("failed to load complete config, using fallbacks", slog.String("error", err.Error()))
		cfg = &config.Config{
			GRPCPort:   "9023",
			HealthPort: "9096",
		}
	}

	// Initialize dependencies
	handler := wire.InitializeHandler(cfg)
	
	// Start the server
	slog.Info("Starting graphiti-knowledge service", slog.String("grpc_port", cfg.GRPCPort))
	server.StartGRPCServer(cfg.GRPCPort, cfg.HealthPort, handler)
}
