// Package handler provides the main HTTP router for vnp-gateway.
package handler

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
	"github.com/vnp-community/vnp-memory/gateway/internal/infra/middleware"
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
	auth  *AuthHandler,
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
	// AgentMemory handlers (TASK-AM-002/005)
	agentmemH *AgentMemoryHandler,
	// Org + SDK handlers (SOL-002 / TASK-003/004)
	org *OrgHandler,
	sdk *SDKHandler,
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

	// === /v1/observe/* (AgentMemory — am-observe service) ===
	mux.HandleFunc("POST /v1/observe/sessions", agentmemH.StartSession)
	mux.HandleFunc("POST /v1/observe/sessions/{id}/observe", agentmemH.Observe)
	mux.HandleFunc("POST /v1/observe/sessions/{id}/end", agentmemH.EndSession)
	mux.HandleFunc("GET /v1/observe/sessions/{id}", agentmemH.GetSession)
	mux.HandleFunc("GET /v1/observe/sessions", agentmemH.ListSessions)
	mux.HandleFunc("DELETE /v1/observe/sessions/{id}", agentmemH.DeleteSession)
	mux.HandleFunc("GET /v1/observe/sessions/{id}/observations", agentmemH.GetObservations)
	mux.HandleFunc("GET /v1/observe/stream", agentmemH.StreamEvents)

	// === /v1/memory/agent/* (AgentMemory — am-memory service) ===
	mux.HandleFunc("POST /v1/memory/agent/remember", agentmemH.RememberAgent)
	mux.HandleFunc("GET /v1/memory/agent/list", agentmemH.ListAgentMemories)
	mux.HandleFunc("GET /v1/memory/agent/{id}", agentmemH.GetAgentMemory)
	mux.HandleFunc("DELETE /v1/memory/agent/{id}", agentmemH.DeleteAgentMemory)
	mux.HandleFunc("GET /v1/memory/agent/{id}/retention", agentmemH.GetRetentionScore)
	mux.HandleFunc("POST /v1/memory/agent/evict", agentmemH.EvictMemories)
	mux.HandleFunc("POST /v1/memory/agent/auto-forget", agentmemH.AutoForgetSweep)

	// === /v1/memory/slots/* (AgentMemory — memory slots) ===
	mux.HandleFunc("GET /v1/memory/slots", agentmemH.ListSlots)
	mux.HandleFunc("GET /v1/memory/slots/{scope}/{label}", agentmemH.GetSlot)
	mux.HandleFunc("POST /v1/memory/slots/{scope}/{label}", agentmemH.WriteSlot)
	mux.HandleFunc("DELETE /v1/memory/slots/{scope}/{label}", agentmemH.DeleteSlot)

	// === /v1/memory/agent/* — governance routes ===
	mux.HandleFunc("DELETE /v1/memory/agent/{id}/governance", agentmemH.GovernanceDelete)
	mux.HandleFunc("GET /v1/memory/audit", agentmemH.ListAudit)

	// === /v1/memory/* — consolidation routes ===
	mux.HandleFunc("POST /v1/memory/compress", agentmemH.CompressObservation)
	mux.HandleFunc("POST /v1/memory/summarize", agentmemH.SummarizeSession)
	mux.HandleFunc("POST /v1/memory/consolidate", agentmemH.RunConsolidationPipeline)
	mux.HandleFunc("GET /v1/memory/procedural", agentmemH.ListProcedural)
	mux.HandleFunc("GET /v1/memory/procedural/{id}", agentmemH.GetProcedural)
	mux.HandleFunc("GET /v1/memory/lessons", agentmemH.ListLessons)
	mux.HandleFunc("GET /v1/memory/lessons/{id}", agentmemH.GetLesson)
	mux.HandleFunc("POST /v1/memory/lessons/decay-sweep", agentmemH.LessonDecaySweep)
	mux.HandleFunc("GET /v1/memory/insights", agentmemH.ListInsights)

	// === /v1/observe/* — hook endpoints (alternate paths) ===
	mux.HandleFunc("POST /v1/observe", agentmemH.ObserveHook)
	mux.HandleFunc("POST /v1/observe/session/start", agentmemH.StartObserveSession)
	mux.HandleFunc("POST /v1/observe/session/end", agentmemH.EndObserveSession)

	// === /v1/observe/replay/* — session replay ===
	mux.HandleFunc("GET /v1/observe/replay/sessions", agentmemH.ListReplaySessions)
	mux.HandleFunc("GET /v1/observe/replay/{id}/timeline", agentmemH.LoadReplayTimeline)

	// === /v1/stream — SSE for real-time session events ===
	mux.HandleFunc("GET /v1/stream", agentmemH.StreamSSE)

	// === /v1/observe/search/* — observe-search service ===
	mux.HandleFunc("POST /v1/observe/search/smart", agentmemH.SmartSearch)
	mux.HandleFunc("POST /v1/observe/search/bm25", agentmemH.BM25Search)
	mux.HandleFunc("POST /v1/observe/search/vector", agentmemH.VectorSearch)
	mux.HandleFunc("POST /v1/observe/search/context", agentmemH.BuildSearchContext)
	mux.HandleFunc("POST /v1/observe/search/index", agentmemH.SearchIndexAdd)
	mux.HandleFunc("DELETE /v1/observe/search/index/{docId}", agentmemH.SearchIndexRemove)
	mux.HandleFunc("POST /v1/observe/search/rebuild", agentmemH.RebuildSearchIndex)
	mux.HandleFunc("GET /v1/observe/search/stats", agentmemH.GetSearchIndexStats)

	// === /v1/orchestration/* — am-orchestration service ===
	// Actions
	mux.HandleFunc("POST /v1/orchestration/actions", agentmemH.CreateAction)
	mux.HandleFunc("GET /v1/orchestration/actions", agentmemH.ListActions)
	mux.HandleFunc("GET /v1/orchestration/actions/{id}", agentmemH.GetAction)
	mux.HandleFunc("PATCH /v1/orchestration/actions/{id}", agentmemH.UpdateAction)
	mux.HandleFunc("DELETE /v1/orchestration/actions/{id}", agentmemH.DeleteAction)
	// Leases
	mux.HandleFunc("POST /v1/orchestration/leases/acquire", agentmemH.AcquireLease)
	mux.HandleFunc("POST /v1/orchestration/leases/renew", agentmemH.RenewLease)
	mux.HandleFunc("POST /v1/orchestration/leases/release", agentmemH.ReleaseLease)
	mux.HandleFunc("GET /v1/orchestration/leases/{actionId}", agentmemH.GetLease)
	// Signals
	mux.HandleFunc("POST /v1/orchestration/signals/send", agentmemH.SendSignal)
	mux.HandleFunc("GET /v1/orchestration/signals", agentmemH.ListSignals)
	mux.HandleFunc("POST /v1/orchestration/signals/{id}/read", agentmemH.MarkSignalRead)
	mux.HandleFunc("DELETE /v1/orchestration/signals/{id}", agentmemH.DeleteSignal)
	// Routines
	mux.HandleFunc("POST /v1/orchestration/routines", agentmemH.CreateRoutine)
	mux.HandleFunc("GET /v1/orchestration/routines", agentmemH.ListRoutines)
	mux.HandleFunc("POST /v1/orchestration/routines/{id}/execute", agentmemH.ExecuteRoutine)
	// Checkpoints
	mux.HandleFunc("POST /v1/orchestration/checkpoints", agentmemH.CreateCheckpoint)
	mux.HandleFunc("GET /v1/orchestration/checkpoints", agentmemH.ListCheckpoints)
	mux.HandleFunc("POST /v1/orchestration/checkpoints/{id}/approve", agentmemH.ApproveCheckpoint)
	mux.HandleFunc("POST /v1/orchestration/checkpoints/{id}/reject", agentmemH.RejectCheckpoint)
	// Sentinels
	mux.HandleFunc("POST /v1/orchestration/sentinels", agentmemH.CreateSentinel)
	mux.HandleFunc("GET /v1/orchestration/sentinels", agentmemH.ListSentinels)
	mux.HandleFunc("DELETE /v1/orchestration/sentinels/{id}", agentmemH.DeleteSentinel)
	// Sketches & Crystals
	mux.HandleFunc("POST /v1/orchestration/sketches", agentmemH.CreateSketch)
	mux.HandleFunc("GET /v1/orchestration/sketches", agentmemH.ListSketches)
	mux.HandleFunc("POST /v1/orchestration/sketches/{id}/add-action", agentmemH.AddActionToSketch)
	mux.HandleFunc("POST /v1/orchestration/sketches/{id}/promote", agentmemH.PromoteSketch)
	mux.HandleFunc("GET /v1/orchestration/crystals", agentmemH.ListCrystals)
	mux.HandleFunc("GET /v1/orchestration/crystals/{id}", agentmemH.GetCrystal)

	// === /v1/health + /v1/admin/* — health & admin tools ===
	mux.HandleFunc("GET /v1/health", agentmemH.GetHealthSnapshot)
	mux.HandleFunc("GET /v1/admin/doctor", agentmemH.DoctorCheck)
	mux.HandleFunc("POST /v1/admin/snapshot", agentmemH.CreateSnapshot)
	mux.HandleFunc("GET /v1/admin/snapshots", agentmemH.ListSnapshots)
	mux.HandleFunc("GET /v1/admin/plugin/claude-code", agentmemH.GetPluginConfig)
	mux.HandleFunc("GET /v1/admin/plugin/codex", agentmemH.GetPluginConfig)
	mux.HandleFunc("GET /v1/admin/plugin/opencode", agentmemH.GetPluginConfig)
	mux.HandleFunc("POST /v1/admin/plugin/install", agentmemH.InstallPlugin)

	// === /v1/auth/* — Authentication (SOL-001 / TASK-001,002) ===
	// NOTE: login and refresh are PUBLIC — they bypass JWT middleware (see middleware/auth.go)
	mux.HandleFunc("POST /v1/auth/login",   auth.Login)
	mux.HandleFunc("POST /v1/auth/refresh", auth.Refresh)
	mux.HandleFunc("POST /v1/auth/logout",  auth.Logout)
	mux.HandleFunc("GET /v1/auth/me",       auth.Me)

	// === /v1/console/org/* — Org settings (SOL-002 / TASK-003) ===
	mux.HandleFunc("GET /v1/console/org/settings",  org.GetSettings)
	mux.HandleFunc("PUT /v1/console/org/settings",  org.UpdateSettings)
	mux.HandleFunc("GET /v1/console/org/members",   org.ListMembers)
	mux.HandleFunc("GET /v1/console/org/roles",     org.ListRoles)

	// === /v1/console/sdk/* — SDK management (SOL-002 / TASK-004) ===
	mux.HandleFunc("GET /v1/console/sdk/keys",              sdk.ListKeys)
	mux.HandleFunc("POST /v1/console/sdk/keys",             sdk.CreateKey)
	mux.HandleFunc("DELETE /v1/console/sdk/keys/{id}",      sdk.DeleteKey)
	mux.HandleFunc("GET /v1/console/sdk/rate-limits",       sdk.GetRateLimits)
	mux.HandleFunc("GET /v1/console/sdk/webhooks",          sdk.ListWebhooks)
	mux.HandleFunc("POST /v1/console/sdk/webhooks",         sdk.CreateWebhook)
	mux.HandleFunc("DELETE /v1/console/sdk/webhooks/{id}",  sdk.DeleteWebhook)

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
