package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/infra/config"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/infra/wire"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv, err := wire.InitializeServer(cfg)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	srv.Stop()
}
