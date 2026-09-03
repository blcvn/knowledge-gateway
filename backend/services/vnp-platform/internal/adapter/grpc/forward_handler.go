// Package grpc — HTTP/REST forward handlers for all vnp-platform routes.
//
// Implements the ForwardService pattern: incoming HTTP JSON → usecase → JSON response.
// Covers: admin, governance, dashboard, event, analytics, space, debugger, session console routes.
// Added in: MERGE-P1-T2 (admin), MERGE-P1-T3 (event, dashboard, analytics, space)
package grpc

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/port"
)

// PlatformForwardHandler exposes all vnp-platform HTTP endpoints.
type PlatformForwardHandler struct {
	tenants   port.TenantUseCase
	keys      port.APIKeyUseCase
	users     port.UserUseCase
	health    port.HealthUseCase
	events    port.EventUseCase
	analytics port.AnalyticsUseCase
	projects  port.ProjectUseCase
}

// NewPlatformForwardHandler creates the handler wiring all usecases.
func NewPlatformForwardHandler(
	t port.TenantUseCase, k port.APIKeyUseCase, u port.UserUseCase,
	h port.HealthUseCase, e port.EventUseCase, a port.AnalyticsUseCase, p port.ProjectUseCase,
) *PlatformForwardHandler {
	return &PlatformForwardHandler{
		tenants: t, keys: k, users: u, health: h,
		events: e, analytics: a, projects: p,
	}
}

// ═══════════════════════════════════════════════════════
// Admin / Tenant Routes (T2)
// ═══════════════════════════════════════════════════════

// CreateTenant — POST /v1/admin/tenants
func (h *PlatformForwardHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
		Tier string `json:"tier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Tier == "" {
		req.Tier = "free"
	}
	tenant, err := h.tenants.CreateTenant(r.Context(), req.Name, req.Slug, admin.SubscriptionTier(req.Tier))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tenant)
}

// ListTenants — GET /v1/admin/tenants
func (h *PlatformForwardHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	tenants, total, err := h.tenants.ListTenants(r.Context(), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenants": tenants, "total": total})
}

// GetTenant — GET /v1/admin/tenants/{id}
func (h *PlatformForwardHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "missing tenant id")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}
	tenant, err := h.tenants.GetTenant(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tenant)
}

// UpdateTenant — PUT /v1/admin/tenants/{id}
func (h *PlatformForwardHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tenant, err := h.tenants.UpdateTenant(r.Context(), id, updates)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tenant)
}

// IssueAPIKey — POST /v1/admin/tenants/{id}/keys
func (h *PlatformForwardHandler) IssueAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}
	var req struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	key, rawKey, err := h.keys.CreateKey(r.Context(), tenantID, req.Name, req.Permissions)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"key":     key,
		"raw_key": rawKey, // only shown once
	})
}

// Health — GET /v1/admin/health
func (h *PlatformForwardHandler) Health(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.health.AggregatedHealth(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "services": statuses})
}

// Metrics — GET /v1/admin/metrics
func (h *PlatformForwardHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "uptime_seconds": 0})
}

// ═══════════════════════════════════════════════════════
// Governance Console Routes (T2)
// ═══════════════════════════════════════════════════════

// ListPolicies — GET /v1/console/governance/policies (stub)
func (h *PlatformForwardHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"policies": []any{}, "total": 0})
}

// CreatePolicy — POST /v1/console/governance/policies (stub)
func (h *PlatformForwardHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	_ = json.NewDecoder(r.Body).Decode(&req)
	writeJSON(w, http.StatusCreated, map[string]any{"id": uuid.New().String(), "created": true})
}

// UpdatePolicy — PUT /v1/console/governance/policies/{id} (stub)
func (h *PlatformForwardHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}

// SearchAudit — GET /v1/console/governance/audit
func (h *PlatformForwardHandler) SearchAudit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"entries": []any{}, "total": 0})
}

// GDPRForget — POST /v1/console/governance/gdpr/forget
func (h *PlatformForwardHandler) GDPRForget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	// TODO: cascade delete user data from all engines via NATS
	writeJSON(w, http.StatusAccepted, map[string]any{"job_id": uuid.New().String(), "status": "queued"})
}

// GDPRForgetPreview — POST /v1/console/governance/gdpr/forget/preview
func (h *PlatformForwardHandler) GDPRForgetPreview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"affected_records": 0, "engines": []string{}})
}

// ═══════════════════════════════════════════════════════
// Dashboard Routes (T3 - vnp-dashboard domain)
// ═══════════════════════════════════════════════════════

// DashboardHealth — GET /v1/console/dashboard/health
func (h *PlatformForwardHandler) DashboardHealth(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.health.AggregatedHealth(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "engines": statuses})
}

// DashboardMetrics — GET /v1/console/dashboard/metrics
func (h *PlatformForwardHandler) DashboardMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"total_memories":     0,
		"active_sessions":    0,
		"requests_per_min":   0.0,
		"storage_used_bytes": 0,
		"last_updated_at":    time.Now().UTC(),
	})
}

// DashboardThroughput — GET /v1/console/dashboard/throughput
func (h *PlatformForwardHandler) DashboardThroughput(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"points": []any{}})
}

// DashboardHeatmap — GET /v1/console/dashboard/heatmap
func (h *PlatformForwardHandler) DashboardHeatmap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"heatmap": []any{}})
}

// ═══════════════════════════════════════════════════════
// Event Routes (T3 - vnp-event domain)
// ═══════════════════════════════════════════════════════

// GetUserEvents — GET /v1/memobase/users/{user_id}/events
func (h *PlatformForwardHandler) GetUserEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	userID := r.PathValue("user_id")
	if tenantID == "" || userID == "" {
		writeError(w, http.StatusBadRequest, "missing tenant or user id")
		return
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	timeline, err := h.events.GetTimeline(r.Context(), tid, uid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, timeline)
}

// ═══════════════════════════════════════════════════════
// Analytics Routes (T3 - sm-analytics domain)
// ═══════════════════════════════════════════════════════

// GetAnalytics — GET /v1/console/adaptive/analytics
func (h *PlatformForwardHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	if tenantIDStr == "" {
		writeJSON(w, http.StatusOK, map[string]any{"records": []any{}, "period": period})
		return
	}
	tid, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}
	records, err := h.analytics.GetUsageReport(r.Context(), tid, period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": records, "period": period})
}

// GetForgetRules — GET /v1/console/adaptive/forget-rules (stub)
func (h *PlatformForwardHandler) GetForgetRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"rules": []any{}})
}

// UpdateForgetRules — PUT /v1/console/adaptive/forget-rules (stub)
func (h *PlatformForwardHandler) UpdateForgetRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}

// ═══════════════════════════════════════════════════════
// Space Routes (T3 - sm-project domain)
// ═══════════════════════════════════════════════════════

// CreateSpace — POST /v1/sm/projects/spaces
func (h *PlatformForwardHandler) CreateSpace(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tid, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant id")
		return
	}
	space, err := h.projects.CreateSpace(r.Context(), tid, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, space)
}

// ═══════════════════════════════════════════════════════
// Debugger Console Routes (T3 - stub)
// ═══════════════════════════════════════════════════════

// CreateTrace — POST /v1/console/debugger/trace
func (h *PlatformForwardHandler) CreateTrace(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusCreated, map[string]any{
		"trace_id":   uuid.New().String(),
		"status":     "created",
		"created_at": time.Now().UTC(),
	})
}

// GetTrace — GET /v1/console/debugger/traces/{id}
func (h *PlatformForwardHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{"trace_id": id, "steps": []any{}})
}

// ListTraces — GET /v1/console/debugger/traces
func (h *PlatformForwardHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"traces": []any{}, "total": 0})
}

// ═══════════════════════════════════════════════════════
// Session Console Routes (T3 - stub, data from memory-service)
// ═══════════════════════════════════════════════════════

// ListSessions — GET /v1/console/sessions
func (h *PlatformForwardHandler) ListSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"sessions": []any{}, "total": 0})
}

// GetSession — GET /v1/console/sessions/{id}
func (h *PlatformForwardHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "messages": []any{}})
}

// GetSessionTimeline — GET /v1/console/sessions/{id}/timeline
func (h *PlatformForwardHandler) GetSessionTimeline(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"events": []any{}})
}

// GetSessionDiff — GET /v1/console/sessions/{id}/diff
func (h *PlatformForwardHandler) GetSessionDiff(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"additions": []any{}, "deletions": []any{}})
}

// GetWorkingMemory — GET /v1/console/sessions/{id}/working-memory
func (h *PlatformForwardHandler) GetWorkingMemory(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"memories": []any{}})
}

// GetUserSummary — GET /v1/console/sessions/{id}/user-summary
func (h *PlatformForwardHandler) GetUserSummary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"summary": ""})
}

// ListLiveSessions — GET /v1/console/sessions/live
func (h *PlatformForwardHandler) ListLiveSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"sessions": []any{}, "total": 0})
}

// ═══════════════════════════════════════════════════════
// Profile Console Routes (T3 - delegates to memory-service data)
// These are wired at platform side but serve data from memory stores.
// ═══════════════════════════════════════════════════════

// ListProfiles — GET /v1/console/profiles
func (h *PlatformForwardHandler) ListProfiles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"profiles": []any{}, "total": 0})
}

// GetProfile — GET /v1/console/profiles/{user_id}
func (h *PlatformForwardHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"user_id": r.PathValue("user_id"), "topics": []any{}})
}

// GetProfileEvents — GET /v1/console/profiles/{user_id}/events
func (h *PlatformForwardHandler) GetProfileEvents(w http.ResponseWriter, r *http.Request) {
	// Delegate to event usecase
	h.GetUserEvents(w, r)
}

// GetProfileContext — GET /v1/console/profiles/{user_id}/context
func (h *PlatformForwardHandler) GetProfileContext(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"context": ""})
}

// GetProfileBuffers — GET /v1/console/profiles/{user_id}/buffers
func (h *PlatformForwardHandler) GetProfileBuffers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"buffers": []any{}})
}

// GetProfileConfig — GET /v1/console/profiles/config
func (h *PlatformForwardHandler) GetProfileConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"schema": map[string]any{}})
}

// UpdateProfileConfig — PUT /v1/console/profiles/config
func (h *PlatformForwardHandler) UpdateProfileConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}

// ═══════════════════════════════════════════════════════
// Pipeline Console Routes (T3 - delegated, stub)
// ═══════════════════════════════════════════════════════

// PipelineStatus — GET /v1/console/pipelines/status
func (h *PlatformForwardHandler) PipelineStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"engines":     map[string]any{},
		"total_jobs":  0,
		"queue_depth": 0,
		"workers":     0,
		"updated_at":  time.Now().UTC(),
	})
}

// Helper to get context without unused import
var _ = context.Background
