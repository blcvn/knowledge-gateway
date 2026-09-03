package handler

import (
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/usecase/port"
)

// AgentMemoryHandler handles /v1/observe/* and /v1/memory/agent/* and /v1/memory/slots/* routes.
// Routes are forwarded to the appropriate backend services: observe-service and memory-service.
type AgentMemoryHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

// NewAgentMemoryHandler creates a new AgentMemoryHandler
func NewAgentMemoryHandler(registry port.ServiceRegistry, logger *slog.Logger) *AgentMemoryHandler {
	return &AgentMemoryHandler{registry: registry, logger: logger}
}

// ── Observe routes ────────────────────────────────────────────────────────

// StartSession handles POST /v1/observe/sessions
func (h *AgentMemoryHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// Observe handles POST /v1/observe/sessions/{id}/observe
func (h *AgentMemoryHandler) Observe(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// EndSession handles POST /v1/observe/sessions/{id}/end
func (h *AgentMemoryHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// GetSession handles GET /v1/observe/sessions/{id}
func (h *AgentMemoryHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// ListSessions handles GET /v1/observe/sessions
func (h *AgentMemoryHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// DeleteSession handles DELETE /v1/observe/sessions/{id}
func (h *AgentMemoryHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// GetObservations handles GET /v1/observe/sessions/{id}/observations
func (h *AgentMemoryHandler) GetObservations(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// StreamEvents handles GET /v1/observe/stream
func (h *AgentMemoryHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// ── AgentMemory routes ────────────────────────────────────────────────────

// RememberAgent handles POST /v1/memory/agent/remember
func (h *AgentMemoryHandler) RememberAgent(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// ListAgentMemories handles GET /v1/memory/agent/list
func (h *AgentMemoryHandler) ListAgentMemories(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetAgentMemory handles GET /v1/memory/agent/{id}
func (h *AgentMemoryHandler) GetAgentMemory(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// DeleteAgentMemory handles DELETE /v1/memory/agent/{id}
func (h *AgentMemoryHandler) DeleteAgentMemory(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetRetentionScore handles GET /v1/memory/agent/{id}/retention
func (h *AgentMemoryHandler) GetRetentionScore(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// EvictMemories handles POST /v1/memory/agent/evict
func (h *AgentMemoryHandler) EvictMemories(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// AutoForgetSweep handles POST /v1/memory/agent/auto-forget
func (h *AgentMemoryHandler) AutoForgetSweep(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// ── Slots routes ──────────────────────────────────────────────────────────

// ListSlots handles GET /v1/memory/slots
func (h *AgentMemoryHandler) ListSlots(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetSlot handles GET /v1/memory/slots/{scope}/{label}
func (h *AgentMemoryHandler) GetSlot(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// WriteSlot handles POST /v1/memory/slots/{scope}/{label}
func (h *AgentMemoryHandler) WriteSlot(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// DeleteSlot handles DELETE /v1/memory/slots/{scope}/{label}
func (h *AgentMemoryHandler) DeleteSlot(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}
