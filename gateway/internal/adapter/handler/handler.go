// Package handler provides HTTP request handlers for all 8 API namespaces.
package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
	"github.com/vnp-community/vnp-memory/gateway/internal/infra/middleware"
	"github.com/vnp-community/vnp-memory/gateway/internal/usecase"
	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes a structured JSON error response.
func WriteError(w http.ResponseWriter, err error) {
	if gErr, ok := err.(*domain.GatewayError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(gErr.HTTPStatusCode())
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    gErr.Code,
				"message": gErr.Message,
				"details": gErr.Details,
			},
		})
		return
	}
	// Generic 500
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "INTERNAL",
			"message": err.Error(),
		},
	})
}

// ReadBody reads the request body.
func ReadBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

// ForwardToService is a generic handler that forwards requests to a named service.
func ForwardToService(registry port.ServiceRegistry, serviceName string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth, _ := middleware.AuthFromContext(r.Context())
		_ = auth

		body, err := ReadBody(r)
		if err != nil {
			WriteError(w, domain.ErrInvalidArgument.WithMessage("failed to read request body"))
			return
		}

		target, err := registry.Resolve(serviceName)
		if err != nil {
			logger.Error("resolve service failed", "service", serviceName, "error", err)
			WriteError(w, domain.ErrCircuitOpen)
			return
		}

		resp, err := registry.Forward(r.Context(), target, body)
		if err != nil {
			WriteError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}

// MemoryHandler handles /v1/memory/* routes.
type MemoryHandler struct {
	router   *usecase.RouteUseCase
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler(router *usecase.RouteUseCase, registry port.ServiceRegistry, logger *slog.Logger) *MemoryHandler {
	return &MemoryHandler{router: router, registry: registry, logger: logger}
}

// Store handles POST /v1/memory/store — auto-routing.
func (h *MemoryHandler) Store(w http.ResponseWriter, r *http.Request) {
	var req domain.StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, domain.ErrInvalidArgument.WithMessage("invalid JSON body"))
		return
	}
	defer r.Body.Close()

	result, err := h.router.Route(r.Context(), &req)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// Recall handles POST /v1/memory/recall — cross-engine search.
func (h *MemoryHandler) Recall(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "vnp-search-hub", h.logger)(w, r)
}

// Forget handles POST /v1/memory/forget — cascading delete.
func (h *MemoryHandler) Forget(w http.ResponseWriter, r *http.Request) {
	// TODO: Fan-out delete to all engines
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

// Timeline handles GET /v1/memory/timeline — temporal event query.
func (h *MemoryHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
}
