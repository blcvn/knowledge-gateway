package handler

import (
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
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

// ── Governance & Audit routes ─────────────────────────────────────────────

// GovernanceDelete handles DELETE /v1/memory/agent/{id}/governance
func (h *AgentMemoryHandler) GovernanceDelete(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// ListAudit handles GET /v1/memory/audit
func (h *AgentMemoryHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// ── Consolidation routes ──────────────────────────────────────────────────

// CompressObservation handles POST /v1/memory/compress
func (h *AgentMemoryHandler) CompressObservation(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// SummarizeSession handles POST /v1/memory/summarize
func (h *AgentMemoryHandler) SummarizeSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// RunConsolidationPipeline handles POST /v1/memory/consolidate
func (h *AgentMemoryHandler) RunConsolidationPipeline(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// ListProcedural handles GET /v1/memory/procedural
func (h *AgentMemoryHandler) ListProcedural(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// GetProcedural handles GET /v1/memory/procedural/{id}
func (h *AgentMemoryHandler) GetProcedural(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// ListLessons handles GET /v1/memory/lessons
func (h *AgentMemoryHandler) ListLessons(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// GetLesson handles GET /v1/memory/lessons/{id}
func (h *AgentMemoryHandler) GetLesson(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// LessonDecaySweep handles POST /v1/memory/lessons/decay-sweep
func (h *AgentMemoryHandler) LessonDecaySweep(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// ListInsights handles GET /v1/memory/insights
func (h *AgentMemoryHandler) ListInsights(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "consolidation-service", h.logger)(w, r)
}

// ── Observe hook alt paths ────────────────────────────────────────────────

// ObserveHook handles POST /v1/observe (short-form hook)
func (h *AgentMemoryHandler) ObserveHook(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// StartObserveSession handles POST /v1/observe/session/start
func (h *AgentMemoryHandler) StartObserveSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// EndObserveSession handles POST /v1/observe/session/end
func (h *AgentMemoryHandler) EndObserveSession(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// ── Replay routes ─────────────────────────────────────────────────────────

// ListReplaySessions handles GET /v1/observe/replay/sessions
func (h *AgentMemoryHandler) ListReplaySessions(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// LoadReplayTimeline handles GET /v1/observe/replay/{id}/timeline
func (h *AgentMemoryHandler) LoadReplayTimeline(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// StreamSSE handles GET /v1/stream (SSE subscription)
func (h *AgentMemoryHandler) StreamSSE(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-service", h.logger)(w, r)
}

// ── Observe-Search routes ─────────────────────────────────────────────────

// SmartSearch handles POST /v1/observe/search/smart
func (h *AgentMemoryHandler) SmartSearch(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// BM25Search handles POST /v1/observe/search/bm25
func (h *AgentMemoryHandler) BM25Search(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// VectorSearch handles POST /v1/observe/search/vector
func (h *AgentMemoryHandler) VectorSearch(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// BuildSearchContext handles POST /v1/observe/search/context
func (h *AgentMemoryHandler) BuildSearchContext(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// SearchIndexAdd handles POST /v1/observe/search/index
func (h *AgentMemoryHandler) SearchIndexAdd(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// SearchIndexRemove handles DELETE /v1/observe/search/index/{docId}
func (h *AgentMemoryHandler) SearchIndexRemove(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// RebuildSearchIndex handles POST /v1/observe/search/rebuild
func (h *AgentMemoryHandler) RebuildSearchIndex(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// GetSearchIndexStats handles GET /v1/observe/search/stats
func (h *AgentMemoryHandler) GetSearchIndexStats(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "observe-search", h.logger)(w, r)
}

// ── Orchestration: Actions ────────────────────────────────────────────────

// CreateAction handles POST /v1/orchestration/actions
func (h *AgentMemoryHandler) CreateAction(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ListActions handles GET /v1/orchestration/actions
func (h *AgentMemoryHandler) ListActions(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// GetAction handles GET /v1/orchestration/actions/{id}
func (h *AgentMemoryHandler) GetAction(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// UpdateAction handles PATCH /v1/orchestration/actions/{id}
func (h *AgentMemoryHandler) UpdateAction(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// DeleteAction handles DELETE /v1/orchestration/actions/{id}
func (h *AgentMemoryHandler) DeleteAction(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ── Orchestration: Leases ─────────────────────────────────────────────────

// AcquireLease handles POST /v1/orchestration/leases/acquire
func (h *AgentMemoryHandler) AcquireLease(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// RenewLease handles POST /v1/orchestration/leases/renew
func (h *AgentMemoryHandler) RenewLease(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ReleaseLease handles POST /v1/orchestration/leases/release
func (h *AgentMemoryHandler) ReleaseLease(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// GetLease handles GET /v1/orchestration/leases/{actionId}
func (h *AgentMemoryHandler) GetLease(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ── Orchestration: Signals ────────────────────────────────────────────────

// SendSignal handles POST /v1/orchestration/signals/send
func (h *AgentMemoryHandler) SendSignal(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ListSignals handles GET /v1/orchestration/signals
func (h *AgentMemoryHandler) ListSignals(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// MarkSignalRead handles POST /v1/orchestration/signals/{id}/read
func (h *AgentMemoryHandler) MarkSignalRead(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// DeleteSignal handles DELETE /v1/orchestration/signals/{id}
func (h *AgentMemoryHandler) DeleteSignal(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ── Orchestration: Routines ───────────────────────────────────────────────

// CreateRoutine handles POST /v1/orchestration/routines
func (h *AgentMemoryHandler) CreateRoutine(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ListRoutines handles GET /v1/orchestration/routines
func (h *AgentMemoryHandler) ListRoutines(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ExecuteRoutine handles POST /v1/orchestration/routines/{id}/execute
func (h *AgentMemoryHandler) ExecuteRoutine(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ── Orchestration: Checkpoints ────────────────────────────────────────────

// CreateCheckpoint handles POST /v1/orchestration/checkpoints
func (h *AgentMemoryHandler) CreateCheckpoint(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ListCheckpoints handles GET /v1/orchestration/checkpoints
func (h *AgentMemoryHandler) ListCheckpoints(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ApproveCheckpoint handles POST /v1/orchestration/checkpoints/{id}/approve
func (h *AgentMemoryHandler) ApproveCheckpoint(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// RejectCheckpoint handles POST /v1/orchestration/checkpoints/{id}/reject
func (h *AgentMemoryHandler) RejectCheckpoint(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ── Orchestration: Sentinels ──────────────────────────────────────────────

// CreateSentinel handles POST /v1/orchestration/sentinels
func (h *AgentMemoryHandler) CreateSentinel(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ListSentinels handles GET /v1/orchestration/sentinels
func (h *AgentMemoryHandler) ListSentinels(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// DeleteSentinel handles DELETE /v1/orchestration/sentinels/{id}
func (h *AgentMemoryHandler) DeleteSentinel(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ── Orchestration: Sketches & Crystals ───────────────────────────────────

// CreateSketch handles POST /v1/orchestration/sketches
func (h *AgentMemoryHandler) CreateSketch(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ListSketches handles GET /v1/orchestration/sketches
func (h *AgentMemoryHandler) ListSketches(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// AddActionToSketch handles POST /v1/orchestration/sketches/{id}/add-action
func (h *AgentMemoryHandler) AddActionToSketch(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// PromoteSketch handles POST /v1/orchestration/sketches/{id}/promote
func (h *AgentMemoryHandler) PromoteSketch(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ListCrystals handles GET /v1/orchestration/crystals
func (h *AgentMemoryHandler) ListCrystals(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// GetCrystal handles GET /v1/orchestration/crystals/{id}
func (h *AgentMemoryHandler) GetCrystal(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "orchestration-service", h.logger)(w, r)
}

// ── Health & Admin routes ─────────────────────────────────────────────────

// GetHealthSnapshot handles GET /v1/health
func (h *AgentMemoryHandler) GetHealthSnapshot(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-platform", h.logger)(w, r)
}

// DoctorCheck handles GET /v1/admin/doctor
func (h *AgentMemoryHandler) DoctorCheck(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-platform", h.logger)(w, r)
}

// CreateSnapshot handles POST /v1/admin/snapshot
func (h *AgentMemoryHandler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-platform", h.logger)(w, r)
}

// ListSnapshots handles GET /v1/admin/snapshots
func (h *AgentMemoryHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-platform", h.logger)(w, r)
}

// GetPluginConfig handles GET /v1/admin/plugin/{name}
func (h *AgentMemoryHandler) GetPluginConfig(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-platform", h.logger)(w, r)
}

// InstallPlugin handles POST /v1/admin/plugin/install
func (h *AgentMemoryHandler) InstallPlugin(w http.ResponseWriter, r *http.Request) {
	ForwardToService(h.registry, "memory-platform", h.logger)(w, r)
}
