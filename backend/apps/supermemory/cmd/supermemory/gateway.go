package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/vnp-community/vnp-memory/apps/supermemory/internal/config"
)

func startGateway(ctx context.Context, cfg *config.Config) error {
	// Stub gateway initialization
	port := cfg.Services.GatewayPort
	log.Printf("Starting VNP Gateway on port %d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Gateway Route Working"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("gateway serve: %w", err)
	case <-ctx.Done():
		log.Printf("Shutting down Gateway")
		return server.Shutdown(context.Background())
	}
}
