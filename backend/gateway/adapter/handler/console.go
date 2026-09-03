// Package handler — Console API handlers for VNP Memory Console UI.
// All console endpoints require admin role and proxy to downstream services.
// Implements FEAT-006 through FEAT-017 (SOL-002).
package handler

import (
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/infra/middleware"
	"github.com/vnp-community/vnp-memory/gateway/usecase/port"
)

// requireAdmin checks that the request has admin or super_admin role.
func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		WriteError(w, domain.ErrUnauthenticated)
		return false
	}
	for _, role := range auth.Roles {
		if role == "admin" || role == "super_admin" {
			return true
		}
	}
	WriteError(w, domain.ErrForbidden.WithMessage("admin role required for console APIs"))
	return false
}

// requireSuperAdmin checks that the request has super_admin role.
func requireSuperAdmin(w http.ResponseWriter, r *http.Request) bool {
	auth, ok := middleware.AuthFromContext(r.Context())
	if !ok {
		WriteError(w, domain.ErrUnauthenticated)
		return false
	}
	for _, role := range auth.Roles {
		if role == "super_admin" {
			return true
		}
	}
	WriteError(w, domain.ErrForbidden.WithMessage("super_admin role required"))
	return false
}

// ──── T01: Dashboard Handler (FEAT-006) ────────────────────────

// DashboardHandler handles /v1/console/dashboard/* routes.
type DashboardHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewDashboardHandler(registry port.ServiceRegistry, logger *slog.Logger) *DashboardHandler {
	return &DashboardHandler{registry: registry, logger: logger}
}

// Health handles GET /v1/console/dashboard/health — aggregated engine health.
func (h *DashboardHandler) Health(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	// Fan-out health checks to all engine groups via vnp-platform
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// Metrics handles GET /v1/console/dashboard/metrics — KPI cards.
func (h *DashboardHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// Throughput handles GET /v1/console/dashboard/throughput — per-engine throughput.
func (h *DashboardHandler) Throughput(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// Heatmap handles GET /v1/console/dashboard/heatmap — memory density heatmap.
func (h *DashboardHandler) Heatmap(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// ──── T02: Explorer Handler (FEAT-007) ─────────────────────────

// ExplorerHandler handles /v1/console/memory/* routes.
type ExplorerHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewExplorerHandler(registry port.ServiceRegistry, logger *slog.Logger) *ExplorerHandler {
	return &ExplorerHandler{registry: registry, logger: logger}
}

// Search handles POST /v1/console/memory/search — unified cross-engine search.
func (h *ExplorerHandler) Search(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// GetMemory handles GET /v1/console/memory/{id} — memory detail.
func (h *ExplorerHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// GetNeighbors handles GET /v1/console/memory/{id}/neighbors — graph neighbors.
func (h *ExplorerHandler) GetNeighbors(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// GetVersions handles GET /v1/console/memory/{id}/versions — version chain.
func (h *ExplorerHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// ──── T17: Graph Studio Handler (FEAT-013) ─────────────────────

// GraphHandler handles /v1/console/graph/* routes.
type GraphHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewGraphHandler(registry port.ServiceRegistry, logger *slog.Logger) *GraphHandler {
	return &GraphHandler{registry: registry, logger: logger}
}

// Subgraph handles POST /v1/console/graph/subgraph — query subgraph by entity.
func (h *GraphHandler) Subgraph(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "kg-service", h.logger)(w, r)
}

// GetEntity handles GET /v1/console/graph/entity/{id} — entity detail.
func (h *GraphHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "kg-service", h.logger)(w, r)
}

// Timeline handles POST /v1/console/graph/timeline — temporal subgraph.
func (h *GraphHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "kg-service", h.logger)(w, r)
}

// GetOntology handles GET /v1/console/graph/ontology — get ontology schema.
func (h *GraphHandler) GetOntology(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "kg-service", h.logger)(w, r)
}

// UpdateOntology handles PUT /v1/console/graph/ontology — update ontology.
func (h *GraphHandler) UpdateOntology(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "kg-service", h.logger)(w, r)
}

// Query handles POST /v1/console/graph/query — execute Cypher/NL query.
func (h *GraphHandler) Query(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "kg-service", h.logger)(w, r)
}

// ──── T03: Profile Handler (FEAT-008) ──────────────────────────

// ProfileHandler handles /v1/console/profiles/* routes.
type ProfileHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewProfileHandler(registry port.ServiceRegistry, logger *slog.Logger) *ProfileHandler {
	return &ProfileHandler{registry: registry, logger: logger}
}

// ListProfiles handles GET /v1/console/profiles — list user profiles.
func (h *ProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetProfile handles GET /v1/console/profiles/{user_id} — profile detail.
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetEvents handles GET /v1/console/profiles/{user_id}/events — event timeline.
func (h *ProfileHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// GetContext handles GET /v1/console/profiles/{user_id}/context — context preview.
func (h *ProfileHandler) GetContext(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetBuffers handles GET /v1/console/profiles/{user_id}/buffers — buffer status.
func (h *ProfileHandler) GetBuffers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetConfig handles GET /v1/console/profiles/config — profile schema config.
func (h *ProfileHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// UpdateConfig handles PUT /v1/console/profiles/config — update profile schema.
func (h *ProfileHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// ──── T04: Adaptive Handler (FEAT-009) ─────────────────────────

// AdaptiveHandler handles /v1/console/adaptive/* routes.
type AdaptiveHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewAdaptiveHandler(registry port.ServiceRegistry, logger *slog.Logger) *AdaptiveHandler {
	return &AdaptiveHandler{registry: registry, logger: logger}
}

// ListMemories handles GET /v1/console/adaptive/memories.
func (h *AdaptiveHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// GetVersions handles GET /v1/console/adaptive/memories/{id}/versions.
func (h *AdaptiveHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// ListConnectors handles GET /v1/console/adaptive/connectors.
func (h *AdaptiveHandler) ListConnectors(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// CreateConnector handles POST /v1/console/adaptive/connectors.
func (h *AdaptiveHandler) CreateConnector(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// SyncConnector handles POST /v1/console/adaptive/connectors/{id}/sync.
func (h *AdaptiveHandler) SyncConnector(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "search-service", h.logger)(w, r)
}

// GetAnalytics handles GET /v1/console/adaptive/analytics.
func (h *AdaptiveHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// GetForgetRules handles GET /v1/console/adaptive/forget-rules.
func (h *AdaptiveHandler) GetForgetRules(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// UpdateForgetRules handles PUT /v1/console/adaptive/forget-rules.
func (h *AdaptiveHandler) UpdateForgetRules(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// ──── T05: Debugger Handler (FEAT-010) ─────────────────────────

// DebuggerHandler handles /v1/console/debugger/* routes.
type DebuggerHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewDebuggerHandler(registry port.ServiceRegistry, logger *slog.Logger) *DebuggerHandler {
	return &DebuggerHandler{registry: registry, logger: logger}
}

// CreateTrace handles POST /v1/console/debugger/trace — simulate context assembly.
func (h *DebuggerHandler) CreateTrace(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// GetTrace handles GET /v1/console/debugger/traces/{id} — get saved trace.
func (h *DebuggerHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// ListTraces handles GET /v1/console/debugger/traces — list recent traces.
func (h *DebuggerHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// ──── T18: Session Handler (FEAT-014) ──────────────────────────

// SessionHandler handles /v1/console/sessions/* routes.
type SessionHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewSessionHandler(registry port.ServiceRegistry, logger *slog.Logger) *SessionHandler {
	return &SessionHandler{registry: registry, logger: logger}
}

// ListSessions handles GET /v1/console/sessions — list sessions.
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetSession handles GET /v1/console/sessions/{id} — session detail.
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetTimeline handles GET /v1/console/sessions/{id}/timeline.
func (h *SessionHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetDiff handles GET /v1/console/sessions/{id}/diff — memory diff.
func (h *SessionHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// GetWorkingMemory handles GET /v1/console/sessions/{id}/working-memory.
func (h *SessionHandler) GetWorkingMemory(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "storage-service", h.logger)(w, r)
}

// GetUserSummary handles GET /v1/console/sessions/{id}/user-summary.
func (h *SessionHandler) GetUserSummary(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// ListLiveSessions handles GET /v1/console/sessions/live — active sessions.
func (h *SessionHandler) ListLiveSessions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}

// ──── T06: Governance Handler (FEAT-011) ───────────────────────

// GovernanceHandler handles /v1/console/governance/* routes.
type GovernanceHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewGovernanceHandler(registry port.ServiceRegistry, logger *slog.Logger) *GovernanceHandler {
	return &GovernanceHandler{registry: registry, logger: logger}
}

// ListTenants handles GET /v1/console/governance/tenants.
func (h *GovernanceHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// CreateTenant handles POST /v1/console/governance/tenants.
func (h *GovernanceHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// UpdateTenant handles PUT /v1/console/governance/tenants/{id}.
func (h *GovernanceHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// ListPolicies handles GET /v1/console/governance/policies.
func (h *GovernanceHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// CreatePolicy handles POST /v1/console/governance/policies.
func (h *GovernanceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// UpdatePolicy handles PUT /v1/console/governance/policies/{id}.
func (h *GovernanceHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// SearchAudit handles GET /v1/console/governance/audit.
func (h *GovernanceHandler) SearchAudit(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// GDPRForget handles POST /v1/console/governance/gdpr/forget.
func (h *GovernanceHandler) GDPRForget(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// GDPRForgetPreview handles POST /v1/console/governance/gdpr/forget/preview.
func (h *GovernanceHandler) GDPRForgetPreview(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// ──── T19: Pipeline Handler (FEAT-015) ─────────────────────────

// PipelineHandler handles /v1/console/pipelines/* routes.
type PipelineHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewPipelineHandler(registry port.ServiceRegistry, logger *slog.Logger) *PipelineHandler {
	return &PipelineHandler{registry: registry, logger: logger}
}

// Status handles GET /v1/console/pipelines/status — all engines overview.
func (h *PipelineHandler) Status(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
}

// GetEngine handles GET /v1/console/pipelines/{engine} — engine pipeline.
func (h *PipelineHandler) GetEngine(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
}

// ListJobs handles GET /v1/console/pipelines/{engine}/jobs.
func (h *PipelineHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
}

// GetJob handles GET /v1/console/pipelines/{engine}/jobs/{id}.
func (h *PipelineHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
}

// Queues handles GET /v1/console/pipelines/queues.
func (h *PipelineHandler) Queues(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
}

// Workers handles GET /v1/console/pipelines/workers.
func (h *PipelineHandler) Workers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
}

// Templates handles GET /v1/console/pipelines/templates.
func (h *PipelineHandler) Templates(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
}

// ──── T20: Infrastructure Handler (FEAT-016) ───────────────────

// InfraHandler handles /v1/console/infra/* routes.
type InfraHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewInfraHandler(registry port.ServiceRegistry, logger *slog.Logger) *InfraHandler {
	return &InfraHandler{registry: registry, logger: logger}
}

// Topology handles GET /v1/console/infra/topology — service topology graph.
func (h *InfraHandler) Topology(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// ListServices handles GET /v1/console/infra/services.
func (h *InfraHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// GetService handles GET /v1/console/infra/services/{name}.
func (h *InfraHandler) GetService(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// Databases handles GET /v1/console/infra/databases — DB health.
func (h *InfraHandler) Databases(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// Resources handles GET /v1/console/infra/resources — resource usage.
func (h *InfraHandler) Resources(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// Deployments handles GET /v1/console/infra/deployments — deployment timeline.
func (h *InfraHandler) Deployments(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// ──── T21: Observability Handler (FEAT-017) ────────────────────

// ObservabilityHandler handles /v1/console/observability/* routes.
type ObservabilityHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewObservabilityHandler(registry port.ServiceRegistry, logger *slog.Logger) *ObservabilityHandler {
	return &ObservabilityHandler{registry: registry, logger: logger}
}

// Metrics handles GET /v1/console/observability/metrics.
func (h *ObservabilityHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// ListTraces handles GET /v1/console/observability/traces.
func (h *ObservabilityHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// GetTrace handles GET /v1/console/observability/traces/{id}.
func (h *ObservabilityHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// Errors handles GET /v1/console/observability/errors.
func (h *ObservabilityHandler) Errors(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// Costs handles GET /v1/console/observability/costs — cost analytics.
func (h *ObservabilityHandler) Costs(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "obs-service", h.logger)(w, r)
}

// ──── Org Handler (FEAT-019 / TASK-BE-013) ─────────────────────

// OrgHandler handles /v1/console/org/* routes.
// Cung cấp tenant settings, members, roles cho quản trị tổ chức.
type OrgHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewOrgHandler(registry port.ServiceRegistry, logger *slog.Logger) *OrgHandler {
	return &OrgHandler{registry: registry, logger: logger}
}

// GetSettings handles GET /v1/console/org/settings — tenant configuration.
func (h *OrgHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// UpdateSettings handles PUT /v1/console/org/settings — update tenant config.
func (h *OrgHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// GetMembers handles GET /v1/console/org/members — list console users in tenant.
func (h *OrgHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// GetRoles handles GET /v1/console/org/roles — available roles + permissions.
func (h *OrgHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	// Trả về static roles config — không cần gọi downstream
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(`{"roles":[` +
		`{"id":"owner","name":"Owner","permissions":["*"]},` +
		`{"id":"admin","name":"Admin","permissions":["console:*","api:*"]},` +
		`{"id":"editor","name":"Editor","permissions":["console:read","console:write"]},` +
		`{"id":"viewer","name":"Viewer","permissions":["console:read"]}` +
		`]}`))
}

// ──── SDK Handler (FEAT-019 / TASK-BE-013) ─────────────────────

// SDKHandler handles /v1/console/sdk/* routes.
// Quản lý API keys (sdk_api_keys) và webhooks cho developer integration.
type SDKHandler struct {
	registry port.ServiceRegistry
	logger   *slog.Logger
}

func NewSDKHandler(registry port.ServiceRegistry, logger *slog.Logger) *SDKHandler {
	return &SDKHandler{registry: registry, logger: logger}
}

// ListKeys handles GET /v1/console/sdk/keys — list tenant API keys (no hash returned).
func (h *SDKHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// CreateKey handles POST /v1/console/sdk/keys — generate new API key (shown once).
func (h *SDKHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// RevokeKey handles DELETE /v1/console/sdk/keys/{id} — revoke an API key.
func (h *SDKHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// GetRateLimits handles GET /v1/console/sdk/rate-limits — rate limit config.
func (h *SDKHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// ListWebhooks handles GET /v1/console/sdk/webhooks — list webhooks.
func (h *SDKHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// CreateWebhook handles POST /v1/console/sdk/webhooks — register new webhook.
func (h *SDKHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

// DeleteWebhook handles DELETE /v1/console/sdk/webhooks/{id} — remove webhook.
func (h *SDKHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
}

