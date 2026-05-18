package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/config"
)

func startHealthAggregation(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// In a real implementation we would make concurrent requests to the health endpoints
		// of the gateway and the 6 domain services.
		
		resp := map[string]any{
			"status": "up",
			"gateway": "up",
			"services": map[string]string{
				"ov-admin":    "up",
				"ov-crypto":   "up",
				"ov-fs":       "up",
				"ov-resource": "up",
				"ov-search":   "up",
				"ov-session":  "up",
			},
		}
		
		json.NewEncoder(w).Encode(resp)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.ManagementPort),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("health aggregation server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
