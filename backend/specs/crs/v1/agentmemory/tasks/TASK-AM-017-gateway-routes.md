# TASK-AM-017 — Gateway Routes (All AgentMemory APIs)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-017 |
| **Wave** | 3 (Orchestration) |
| **Component** | `gateway/internal/adapter/handler/router.go` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.11, SOL-002 §2.12, SOL-003 §2.8, SOL-004 §2.9, SOL-006 §2.9, SOL-007 §2.8 |
| **Priority** | High |
| **Depends On** | TASK-AM-015, TASK-AM-016 |
| **Estimated** | 4h |

---

## Context

Tổng hợp TẤT CẢ route cần thêm vào gateway router. Task này đảm bảo không có route nào bị bỏ sót.

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Complete Route List

### Observe Service (`am-observe`)

```go
// Observations
r.Post("/v1/observe",                              h.ForwardTo("am-observe", "ObserveService/Observe"))
r.Post("/v1/observe/session/start",                h.ForwardTo("am-observe", "ObserveService/StartSession"))
r.Post("/v1/observe/session/end",                  h.ForwardTo("am-observe", "ObserveService/EndSession"))

// Sessions
r.Get("/v1/observe/sessions",                      h.ForwardTo("am-observe", "ObserveService/ListSessions"))
r.Get("/v1/observe/sessions/{id}",                 h.ForwardTo("am-observe", "ObserveService/GetSession"))
r.Delete("/v1/observe/sessions/{id}",              h.ForwardTo("am-observe", "ObserveService/DeleteSession"))
r.Get("/v1/observe/sessions/{id}/observations",    h.ForwardTo("am-observe", "ObserveService/GetObservations"))

// Replay
r.Get("/v1/observe/replay/sessions",               h.ForwardTo("am-observe", "ObserveService/ListReplaySessions"))
r.Get("/v1/observe/replay/{id}/timeline",          h.ForwardTo("am-observe", "ObserveService/LoadTimeline"))

// SSE
r.Get("/v1/stream",                                observeSSEHandler.ServeSSE)  // Direct HTTP handler
```

### AgentMemory Service (`memory-service`)

```go
// Memories
r.Post("/v1/memory/agent/remember",                h.ForwardTo("memory-service", "AgentMemoryService/RememberAgent"))
r.Get("/v1/memory/agent/list",                     h.ForwardTo("memory-service", "AgentMemoryService/ListAgentMemories"))
r.Get("/v1/memory/agent/{id}",                     h.ForwardTo("memory-service", "AgentMemoryService/GetAgentMemory"))
r.Delete("/v1/memory/agent/{id}",                  h.ForwardTo("memory-service", "AgentMemoryService/DeleteAgentMemory"))
r.Get("/v1/memory/agent/{id}/retention",           h.ForwardTo("memory-service", "AgentMemoryService/GetRetentionScore"))

// Lifecycle
r.Post("/v1/memory/agent/evict",                   h.ForwardTo("memory-service", "AgentMemoryService/EvictMemories"))
r.Post("/v1/memory/agent/auto-forget",             h.ForwardTo("memory-service", "AgentMemoryService/AutoForgetSweep"))

// Slots
r.Get("/v1/memory/slots",                          h.ForwardTo("memory-service", "AgentMemoryService/ListSlots"))
r.Get("/v1/memory/slots/{scope}/{label}",          h.ForwardTo("memory-service", "AgentMemoryService/GetSlot"))
r.Post("/v1/memory/slots/{scope}/{label}",         h.ForwardTo("memory-service", "AgentMemoryService/WriteSlot"))
r.Delete("/v1/memory/slots/{scope}/{label}",       h.ForwardTo("memory-service", "AgentMemoryService/DeleteSlot"))

// Consolidation
r.Post("/v1/memory/compress",                      h.ForwardTo("memory-service", "ConsolidationService/CompressObservation"))
r.Post("/v1/memory/summarize",                     h.ForwardTo("memory-service", "ConsolidationService/SummarizeSession"))
r.Post("/v1/memory/consolidate",                   h.ForwardTo("memory-service", "ConsolidationService/RunPipeline"))

// Procedural, Lessons, Insights
r.Get("/v1/memory/procedural",                     h.ForwardTo("memory-service", "ConsolidationService/ListProcedural"))
r.Get("/v1/memory/procedural/{id}",                h.ForwardTo("memory-service", "ConsolidationService/GetProcedural"))
r.Get("/v1/memory/lessons",                        h.ForwardTo("memory-service", "ConsolidationService/ListLessons"))
r.Get("/v1/memory/lessons/{id}",                   h.ForwardTo("memory-service", "ConsolidationService/GetLesson"))
r.Post("/v1/memory/lessons/decay-sweep",           h.ForwardTo("memory-service", "ConsolidationService/LessonDecaySweep"))
r.Get("/v1/memory/insights",                       h.ForwardTo("memory-service", "ConsolidationService/ListInsights"))

// Governance
r.Delete("/v1/memory/agent/{id}/governance",       h.ForwardTo("memory-service", "GovernanceService/Delete"))
r.Get("/v1/memory/audit",                          h.ForwardTo("memory-service", "GovernanceService/ListAudit"))
```

### Observe-Search Service (`am-search`)

```go
r.Post("/v1/observe/search/smart",                 h.ForwardTo("am-search", "ObserveSearchService/SmartSearch"))
r.Post("/v1/observe/search/bm25",                  h.ForwardTo("am-search", "ObserveSearchService/BM25Search"))
r.Post("/v1/observe/search/vector",                h.ForwardTo("am-search", "ObserveSearchService/VectorSearch"))
r.Post("/v1/observe/search/context",               h.ForwardTo("am-search", "ObserveSearchService/BuildContext"))
r.Post("/v1/observe/search/index",                 h.ForwardTo("am-search", "ObserveSearchService/IndexAdd"))
r.Delete("/v1/observe/search/index/{docId}",       h.ForwardTo("am-search", "ObserveSearchService/IndexRemove"))
r.Post("/v1/observe/search/rebuild",               h.ForwardTo("am-search", "ObserveSearchService/RebuildIndex"))
r.Get("/v1/observe/search/stats",                  h.ForwardTo("am-search", "ObserveSearchService/GetIndexStats"))
```

### Orchestration Service (`am-orchestration`)

```go
// Actions
r.Post("/v1/orchestration/actions",                h.ForwardTo("am-orchestration", "OrchestrationService/CreateAction"))
r.Get("/v1/orchestration/actions",                 h.ForwardTo("am-orchestration", "OrchestrationService/ListActions"))
r.Get("/v1/orchestration/actions/{id}",            h.ForwardTo("am-orchestration", "OrchestrationService/GetAction"))
r.Patch("/v1/orchestration/actions/{id}",          h.ForwardTo("am-orchestration", "OrchestrationService/UpdateAction"))
r.Delete("/v1/orchestration/actions/{id}",         h.ForwardTo("am-orchestration", "OrchestrationService/DeleteAction"))

// Leases
r.Post("/v1/orchestration/leases/acquire",         h.ForwardTo("am-orchestration", "OrchestrationService/AcquireLease"))
r.Post("/v1/orchestration/leases/renew",           h.ForwardTo("am-orchestration", "OrchestrationService/RenewLease"))
r.Post("/v1/orchestration/leases/release",         h.ForwardTo("am-orchestration", "OrchestrationService/ReleaseLease"))
r.Get("/v1/orchestration/leases/{actionId}",       h.ForwardTo("am-orchestration", "OrchestrationService/GetLease"))

// Signals
r.Post("/v1/orchestration/signals/send",           h.ForwardTo("am-orchestration", "OrchestrationService/SendSignal"))
r.Get("/v1/orchestration/signals",                 h.ForwardTo("am-orchestration", "OrchestrationService/ListSignals"))
r.Post("/v1/orchestration/signals/{id}/read",      h.ForwardTo("am-orchestration", "OrchestrationService/MarkSignalRead"))
r.Delete("/v1/orchestration/signals/{id}",         h.ForwardTo("am-orchestration", "OrchestrationService/DeleteSignal"))

// Routines
r.Post("/v1/orchestration/routines",               h.ForwardTo("am-orchestration", "OrchestrationService/CreateRoutine"))
r.Get("/v1/orchestration/routines",                h.ForwardTo("am-orchestration", "OrchestrationService/ListRoutines"))
r.Post("/v1/orchestration/routines/{id}/execute",  h.ForwardTo("am-orchestration", "OrchestrationService/ExecuteRoutine"))

// Checkpoints
r.Post("/v1/orchestration/checkpoints",            h.ForwardTo("am-orchestration", "OrchestrationService/CreateCheckpoint"))
r.Get("/v1/orchestration/checkpoints",             h.ForwardTo("am-orchestration", "OrchestrationService/ListCheckpoints"))
r.Post("/v1/orchestration/checkpoints/{id}/approve", h.ForwardTo("am-orchestration", "OrchestrationService/ApproveCheckpoint"))
r.Post("/v1/orchestration/checkpoints/{id}/reject",  h.ForwardTo("am-orchestration", "OrchestrationService/RejectCheckpoint"))

// Sentinels
r.Post("/v1/orchestration/sentinels",              h.ForwardTo("am-orchestration", "OrchestrationService/CreateSentinel"))
r.Get("/v1/orchestration/sentinels",               h.ForwardTo("am-orchestration", "OrchestrationService/ListSentinels"))
r.Delete("/v1/orchestration/sentinels/{id}",       h.ForwardTo("am-orchestration", "OrchestrationService/DeleteSentinel"))

// Sketches & Crystals
r.Post("/v1/orchestration/sketches",               h.ForwardTo("am-orchestration", "OrchestrationService/CreateSketch"))
r.Get("/v1/orchestration/sketches",                h.ForwardTo("am-orchestration", "OrchestrationService/ListSketches"))
r.Post("/v1/orchestration/sketches/{id}/add-action", h.ForwardTo("am-orchestration", "OrchestrationService/AddActionToSketch"))
r.Post("/v1/orchestration/sketches/{id}/promote",  h.ForwardTo("am-orchestration", "OrchestrationService/PromoteSketch"))
r.Get("/v1/orchestration/crystals",                h.ForwardTo("am-orchestration", "OrchestrationService/ListCrystals"))
r.Get("/v1/orchestration/crystals/{id}",           h.ForwardTo("am-orchestration", "OrchestrationService/GetCrystal"))
```

### Admin / Health / Governance (`vnp-platform`)

```go
// Health & Doctor
r.Get("/v1/health",                                h.ForwardTo("vnp-platform", "AdminService/GetHealthSnapshot"))
r.Get("/v1/admin/doctor",                          h.ForwardTo("vnp-platform", "AdminService/Doctor"))

// Snapshots
r.Post("/v1/admin/snapshot",                       h.ForwardTo("vnp-platform", "AdminService/CreateSnapshot"))
r.Get("/v1/admin/snapshots",                       h.ForwardTo("vnp-platform", "AdminService/ListSnapshots"))

// Plugin configs
r.Get("/v1/admin/plugin/claude-code",              h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Get("/v1/admin/plugin/codex",                    h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Get("/v1/admin/plugin/opencode",                 h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Post("/v1/admin/plugin/install",                 h.ForwardTo("vnp-platform", "AdminService/InstallPlugin"))
```

---

## Route Summary

| Service | New Routes | Total HTTP methods |
|---------|-----------|-------------------|
| `am-observe` | 10 | GET×6, POST×4, DELETE×1 |
| `memory-service` | 22 | GET×12, POST×8, DELETE×2 |
| `am-search` | 8 | GET×2, POST×5, DELETE×1 |
| `am-orchestration` | 30 | GET×10, POST×16, PATCH×1, DELETE×3 |
| `vnp-platform` | 6 | GET×4, POST×2 |
| **Total new** | **76** | |

---

## Verification

```bash
cd gateway
go build ./...

# Verify all routes compile
go test ./internal/adapter/handler/... -v -run TestRoutes

# Check no duplicate routes
grep -E 'r\.(Get|Post|Patch|Delete|Put)\(' internal/adapter/handler/router.go | sort | uniq -d
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| All 76 new routes registered | ✅ |
| No duplicate routes (same method+path) | ✅ |
| Gateway builds without errors | ✅ |
| `GET /v1/health` → 200 OK | ✅ |
| `POST /v1/observe` → proxied to am-observe | ✅ |
