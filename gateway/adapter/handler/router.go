// Package handler provides the main HTTP router for vnp-gateway.
package handler

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/infra/middleware"
)

// Router creates the main HTTP router with all API namespaces and middleware.
// When spaFS is non-nil, non-API routes serve the embedded SPA (UI console).
// When spaFS is nil, non-API routes return a JSON 404 (standalone gateway mode).
func Router(
	memory *MemoryHandler,
	cognee *CogneeHandler,
	graphiti *GraphitiHandler,
	memobase *MemobaseHandler,
	ov *OpenVikingHandler,
	zep *ZepHandler,
	sm *SMHandler,
	admin *AdminHandler,
	// Console handlers (SOL-002)
	dashboard *DashboardHandler,
	explorer *ExplorerHandler,
	graph *GraphHandler,
	profile *ProfileHandler,
	adaptive *AdaptiveHandler,
	debugger *DebuggerHandler,
	session *SessionHandler,
	governance *GovernanceHandler,
	pipeline *PipelineHandler,
	infra *InfraHandler,
	observability *ObservabilityHandler,
	ws *WSHandler,
	logger *slog.Logger,
	spaFS fs.FS,
) http.Handler {
	mux := http.NewServeMux()

	// Middleware chain (applied in order)
	chain := func(h http.Handler) http.Handler {
		h = middleware.Logger(logger)(h)
		h = middleware.CORS("*", "true")(h)
		h = middleware.RequestID()(h)
		h = middleware.Recovery(logger)(h)
		return h
	}

	// === /v1/memory/* ===
	mux.HandleFunc("POST /v1/memory/store", memory.Store)
	mux.HandleFunc("POST /v1/memory/recall", memory.Recall)
	mux.HandleFunc("POST /v1/memory/forget", memory.Forget)
	mux.HandleFunc("GET /v1/memory/timeline", memory.Timeline)

	// === /v1/cognee/* ===
	mux.HandleFunc("POST /v1/cognee/datasets", cognee.CreateDataset)
	mux.HandleFunc("POST /v1/cognee/datasets/{id}/data", cognee.UploadData)
	mux.HandleFunc("POST /v1/cognee/datasets/{id}/cognify", cognee.Cognify)
	mux.HandleFunc("POST /v1/cognee/search", cognee.Search)

	// === /v1/graphiti/* ===
	mux.HandleFunc("POST /v1/graphiti/episodes", graphiti.IngestEpisode)
	mux.HandleFunc("POST /v1/graphiti/search", graphiti.Search)
	mux.HandleFunc("GET /v1/graphiti/nodes/{id}", graphiti.GetNode)
	mux.HandleFunc("GET /v1/graphiti/edges/{id}", graphiti.GetEdge)

	// === /v1/memobase/* ===
	mux.HandleFunc("POST /v1/memobase/users/{uid}/blobs", memobase.InsertBlob)
	mux.HandleFunc("POST /v1/memobase/users/{uid}/flush", memobase.Flush)
	mux.HandleFunc("GET /v1/memobase/users/{uid}/context", memobase.GetContext)
	mux.HandleFunc("GET /v1/memobase/users/{uid}/profiles", memobase.GetProfiles)
	mux.HandleFunc("GET /v1/memobase/users/{uid}/events", memobase.GetEvents)

	// === /v1/ov/* (OpenViking) ===
	mux.HandleFunc("GET /v1/ov/files/{path...}", ov.ReadFile)
	mux.HandleFunc("PUT /v1/ov/files/{path...}", ov.WriteFile)
	mux.HandleFunc("DELETE /v1/ov/files/{path...}", ov.DeleteFile)
	mux.HandleFunc("GET /v1/ov/tree/{path...}", ov.Tree)
	mux.HandleFunc("POST /v1/ov/grep", ov.Grep)
	mux.HandleFunc("POST /v1/ov/search", ov.Search)
	mux.HandleFunc("POST /v1/ov/sessions", ov.CreateSession)
	mux.HandleFunc("POST /v1/ov/sessions/{id}/messages", ov.AddMessage)
	mux.HandleFunc("POST /v1/ov/sessions/{id}/commit", ov.CommitSession)
	mux.HandleFunc("POST /v1/ov/resources/ingest", ov.Ingest)

	// === /v1/zep/* ===
	mux.HandleFunc("POST /v1/zep/users", zep.CreateUser)
	mux.HandleFunc("GET /v1/zep/users/{id}", zep.GetUser)
	mux.HandleFunc("PATCH /v1/zep/users/{id}", zep.UpdateUser)
	mux.HandleFunc("POST /v1/zep/sessions/{id}/memory", zep.PutMemory)
	mux.HandleFunc("GET /v1/zep/sessions/{id}/memory", zep.GetMemory)
	mux.HandleFunc("POST /v1/zep/graph/search", zep.GraphSearch)
	mux.HandleFunc("POST /v1/zep/sessions/{id}/search", zep.SessionSearch)
	mux.HandleFunc("POST /v1/zep/graph/facts", zep.AddFact)
	mux.HandleFunc("POST /v1/zep/graph/ontology", zep.SetOntology)

	// === /v1/sm/* (Supermemory) ===
	mux.HandleFunc("POST /v1/sm/documents", sm.CreateDocument)
	mux.HandleFunc("GET /v1/sm/documents/{id}", sm.GetDocument)
	mux.HandleFunc("POST /v1/sm/memories", sm.CreateMemory)
	mux.HandleFunc("POST /v1/sm/search", sm.Search)
	mux.HandleFunc("POST /v1/sm/rag", sm.RAG)
	mux.HandleFunc("GET /v1/sm/profiles/{uid}", sm.GetProfile)
	mux.HandleFunc("POST /v1/sm/connections", sm.CreateConnection)
	mux.HandleFunc("POST /v1/sm/connections/{id}/sync", sm.SyncConnection)
	mux.HandleFunc("POST /v1/sm/projects/spaces", sm.CreateSpace)

	// === /v1/admin/* ===
	mux.HandleFunc("POST /v1/admin/tenants", admin.CreateTenant)
	mux.HandleFunc("POST /v1/admin/tenants/{id}/keys", admin.IssueAPIKey)
	mux.HandleFunc("GET /v1/admin/health", admin.Health)
	mux.HandleFunc("GET /v1/admin/metrics", admin.Metrics)

	// === /v1/console/dashboard/* (FEAT-006 / T01) ===
	mux.HandleFunc("GET /v1/console/dashboard/health", dashboard.Health)
	mux.HandleFunc("GET /v1/console/dashboard/metrics", dashboard.Metrics)
	mux.HandleFunc("GET /v1/console/dashboard/throughput", dashboard.Throughput)
	mux.HandleFunc("GET /v1/console/dashboard/heatmap", dashboard.Heatmap)

	// === /v1/console/memory/* (FEAT-007 / T02) ===
	mux.HandleFunc("POST /v1/console/memory/search", explorer.Search)
	mux.HandleFunc("GET /v1/console/memory/{id}", explorer.GetMemory)
	mux.HandleFunc("GET /v1/console/memory/{id}/neighbors", explorer.GetNeighbors)
	mux.HandleFunc("GET /v1/console/memory/{id}/versions", explorer.GetVersions)

	// === /v1/console/graph/* (FEAT-013 / T17) ===
	mux.HandleFunc("POST /v1/console/graph/subgraph", graph.Subgraph)
	mux.HandleFunc("GET /v1/console/graph/entity/{id}", graph.GetEntity)
	mux.HandleFunc("POST /v1/console/graph/timeline", graph.Timeline)
	mux.HandleFunc("GET /v1/console/graph/ontology", graph.GetOntology)
	mux.HandleFunc("PUT /v1/console/graph/ontology", graph.UpdateOntology)
	mux.HandleFunc("POST /v1/console/graph/query", graph.Query)

	// === /v1/console/profiles/* (FEAT-008 / T03) ===
	mux.HandleFunc("GET /v1/console/profiles", profile.ListProfiles)
	mux.HandleFunc("GET /v1/console/profiles/config", profile.GetConfig)
	mux.HandleFunc("PUT /v1/console/profiles/config", profile.UpdateConfig)
	mux.HandleFunc("GET /v1/console/profiles/{user_id}", profile.GetProfile)
	mux.HandleFunc("GET /v1/console/profiles/{user_id}/events", profile.GetEvents)
	mux.HandleFunc("GET /v1/console/profiles/{user_id}/context", profile.GetContext)
	mux.HandleFunc("GET /v1/console/profiles/{user_id}/buffers", profile.GetBuffers)

	// === /v1/console/adaptive/* (FEAT-009 / T04) ===
	mux.HandleFunc("GET /v1/console/adaptive/memories", adaptive.ListMemories)
	mux.HandleFunc("GET /v1/console/adaptive/memories/{id}/versions", adaptive.GetVersions)
	mux.HandleFunc("GET /v1/console/adaptive/connectors", adaptive.ListConnectors)
	mux.HandleFunc("POST /v1/console/adaptive/connectors", adaptive.CreateConnector)
	mux.HandleFunc("POST /v1/console/adaptive/connectors/{id}/sync", adaptive.SyncConnector)
	mux.HandleFunc("GET /v1/console/adaptive/analytics", adaptive.GetAnalytics)
	mux.HandleFunc("GET /v1/console/adaptive/forget-rules", adaptive.GetForgetRules)
	mux.HandleFunc("PUT /v1/console/adaptive/forget-rules", adaptive.UpdateForgetRules)

	// === /v1/console/debugger/* (FEAT-010 / T05) ===
	mux.HandleFunc("POST /v1/console/debugger/trace", debugger.CreateTrace)
	mux.HandleFunc("GET /v1/console/debugger/traces/{id}", debugger.GetTrace)
	mux.HandleFunc("GET /v1/console/debugger/traces", debugger.ListTraces)

	// === /v1/console/sessions/* (FEAT-014 / T18) ===
	mux.HandleFunc("GET /v1/console/sessions", session.ListSessions)
	mux.HandleFunc("GET /v1/console/sessions/live", session.ListLiveSessions)
	mux.HandleFunc("GET /v1/console/sessions/{id}", session.GetSession)
	mux.HandleFunc("GET /v1/console/sessions/{id}/timeline", session.GetTimeline)
	mux.HandleFunc("GET /v1/console/sessions/{id}/diff", session.GetDiff)
	mux.HandleFunc("GET /v1/console/sessions/{id}/working-memory", session.GetWorkingMemory)
	mux.HandleFunc("GET /v1/console/sessions/{id}/user-summary", session.GetUserSummary)

	// === /v1/console/governance/* (FEAT-011 / T06) ===
	mux.HandleFunc("GET /v1/console/governance/tenants", governance.ListTenants)
	mux.HandleFunc("POST /v1/console/governance/tenants", governance.CreateTenant)
	mux.HandleFunc("PUT /v1/console/governance/tenants/{id}", governance.UpdateTenant)
	mux.HandleFunc("GET /v1/console/governance/policies", governance.ListPolicies)
	mux.HandleFunc("POST /v1/console/governance/policies", governance.CreatePolicy)
	mux.HandleFunc("PUT /v1/console/governance/policies/{id}", governance.UpdatePolicy)
	mux.HandleFunc("GET /v1/console/governance/audit", governance.SearchAudit)
	mux.HandleFunc("POST /v1/console/governance/gdpr/forget", governance.GDPRForget)
	mux.HandleFunc("POST /v1/console/governance/gdpr/forget/preview", governance.GDPRForgetPreview)

	// === /v1/console/pipelines/* (FEAT-015 / T19) ===
	mux.HandleFunc("GET /v1/console/pipelines/status", pipeline.Status)
	mux.HandleFunc("GET /v1/console/pipelines/queues", pipeline.Queues)
	mux.HandleFunc("GET /v1/console/pipelines/workers", pipeline.Workers)
	mux.HandleFunc("GET /v1/console/pipelines/templates", pipeline.Templates)
	mux.HandleFunc("GET /v1/console/pipelines/{engine}", pipeline.GetEngine)
	mux.HandleFunc("GET /v1/console/pipelines/{engine}/jobs", pipeline.ListJobs)
	mux.HandleFunc("GET /v1/console/pipelines/{engine}/jobs/{id}", pipeline.GetJob)

	// === /v1/console/infra/* (FEAT-016 / T20) ===
	mux.HandleFunc("GET /v1/console/infra/topology", infra.Topology)
	mux.HandleFunc("GET /v1/console/infra/services", infra.ListServices)
	mux.HandleFunc("GET /v1/console/infra/services/{name}", infra.GetService)
	mux.HandleFunc("GET /v1/console/infra/databases", infra.Databases)
	mux.HandleFunc("GET /v1/console/infra/resources", infra.Resources)
	mux.HandleFunc("GET /v1/console/infra/deployments", infra.Deployments)

	// === /v1/console/observability/* (FEAT-017 / T21) ===
	mux.HandleFunc("GET /v1/console/observability/metrics", observability.Metrics)
	mux.HandleFunc("GET /v1/console/observability/traces", observability.ListTraces)
	mux.HandleFunc("GET /v1/console/observability/traces/{id}", observability.GetTrace)
	mux.HandleFunc("GET /v1/console/observability/errors", observability.Errors)
	mux.HandleFunc("GET /v1/console/observability/costs", observability.Costs)

	// === /v1/console/ws (FEAT-012 / T07) ===
	mux.HandleFunc("GET /v1/console/ws", ws.HandleWS)

	// === Catch-all: SPA or 404 ===
	if spaFS != nil {
		// Serve embedded UI console for all non-API routes
		spa := NewSPAHandler(spaFS)
		mux.Handle("/", spa)
		logger.Info("UI console embedded and serving on /")
	} else {
		// Standalone gateway mode — JSON 404 for unmatched routes
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    domain.ErrNotFound.Code,
					"message": "route not found: " + r.Method + " " + r.URL.Path,
				},
			})
		})
	}

	return chain(mux)
}
