package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/vnp-community/vnp-memory/apps/supermemory/internal/config"
)

func startHealthAggregator(ctx context.Context, cfg *config.Config) error {
	port := 9090
	log.Printf("Starting Health Aggregator on port %d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "services_embedded": 10, "gateway": "ok"}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ready"}`))
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
		return fmt.Errorf("health aggregator serve: %w", err)
	case <-ctx.Done():
		log.Printf("Shutting down Health Aggregator")
		return server.Shutdown(context.Background())
	}
}
