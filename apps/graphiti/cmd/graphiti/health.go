// Package main — aggregated health server for the graphiti monolith app.
//
// Provides /healthz (liveness) and /readyz (readiness) on a dedicated
// health port, separate from the gateway REST port.
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/apps/graphiti/internal/config"
	"github.com/vnp-community/vnp-memory/apps/graphiti/internal/supervisor"
)

// startHealthServer runs a dedicated HTTP health server on cfg.Server.HealthPort.
// It polls the supervisor for service status and reports aggregated readiness.
func startHealthServer(cfg *config.Config, sv *supervisor.Supervisor) {
	mux := http.NewServeMux()

	// /healthz — liveness probe (process is alive)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	// /readyz — readiness probe (all services SERVING)
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		status := sv.HealthCheck()
		allReady := true
		for _, s := range status {
			if s != "serving" {
				allReady = false
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if !allReady {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"status":   boolToStatus(allReady),
			"services": status,
		})
	})

	// /status — detailed per-service health (JSON)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		status := sv.HealthCheck()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"app":      "graphiti-app",
			"services": status,
		})
	})

	addr := fmt.Sprintf(":%d", cfg.Server.HealthPort)
	slog.Info("health server starting", "port", cfg.Server.HealthPort)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("health server failed", "error", err)
	}
}

func boolToStatus(ready bool) string {
	if ready {
		return "ready"
	}
	return "not_ready"
}
