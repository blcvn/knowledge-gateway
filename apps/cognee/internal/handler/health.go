package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
)

// HealthChecker is a function that checks health of a dependency.
type HealthChecker func() error

// HealthHandler provides liveness and readiness probe endpoints.
type HealthHandler struct {
	checkers map[string]HealthChecker
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewHealthHandler creates the health check handler.
func NewHealthHandler(logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		checkers: make(map[string]HealthChecker),
		logger:   logger.With("handler", "health"),
	}
}

// Register adds a named health checker for the readiness probe.
func (h *HealthHandler) Register(name string, checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checker
}

// Liveness handles GET /healthz — simple liveness probe (always healthy if process runs).
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// Readiness handles GET /readyz — checks all registered dependencies.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := make(map[string]string)
	allHealthy := true

	for name, checker := range h.checkers {
		if err := checker(); err != nil {
			status[name] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			status[name] = "healthy"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":       map[bool]string{true: "ready", false: "not_ready"}[allHealthy],
		"dependencies": status,
	})
}
