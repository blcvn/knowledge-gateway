package main

import (
	"log"

	"vnp-memory/services/ov-fs/internal/infra/config"
	"vnp-memory/services/ov-fs/internal/infra/server"
	"vnp-memory/services/ov-fs/internal/infra/wire"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize dependencies
	_ = wire.InitializeHandler()

	go func() {
		if err := server.StartHealthServer(cfg); err != nil {
			log.Fatalf("health server failed: %v", err)
		}
	}()

	if err := server.StartGRPCServer(cfg); err != nil {
		log.Fatalf("grpc server failed: %v", err)
	}
}
