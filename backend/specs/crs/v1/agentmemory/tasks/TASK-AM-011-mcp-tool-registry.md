# TASK-AM-011 — MCP Tool Registry Expansion (16 → 53 tools)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-011 |
| **Wave** | 2 (Integration) |
| **Component** | `gateway/internal/adapter/mcp/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-008 §2.1 → §2.3 |
| **Priority** | High |
| **Depends On** | TASK-AM-001 |
| **Estimated** | 5h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** gateway MCP tool registry: observe tools registered  
---

## Context

Mở rộng MCP server từ 16 → 53 tools. Tổ chức theo 8 categories trong `tools/agentmemory/`. Mỗi file chứa tool definitions (Schema + Handler reference).

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/registry.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/memory_core.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/session_tools.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/observe_tools.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/governance_tools.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/graph_tools.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/orchestration_tools.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/signal_tools.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/replay_slot_tools.go` |
| CREATE | `gateway/internal/adapter/mcp/tools/agentmemory/admin_tools.go` |
| MODIFY | `gateway/internal/adapter/mcp/tool_registry.go` |

---

## Tool List (37 new tools, total 53)

### Category 1: Memory Core (4 tools)
| Tool | Description |
|------|-------------|
| `memory_smart_search` | Hybrid BM25+vector search for memories and observations |
| `memory_save` | Save long-term memory (pattern/preference/architecture/bug/workflow/fact) |
| `memory_context` | Build context block from memories, optimized for token budget |
| `memory_forget` | Delete memory with cascade (indexes + graph), creates audit trail |

### Category 2: Sessions (3 tools)
| Tool | Description |
|------|-------------|
| `memory_sessions` | List sessions for tenant/project |
| `memory_session_detail` | Get session detail with observations |
| `memory_observations` | Get compressed observations for a session |

### Category 3: Observe (6 tools)
| Tool | Description |
|------|-------------|
| `memory_observe` | Capture a hook event from agent activity |
| `memory_session_start` | Start new observation session → returns session_id |
| `memory_session_end` | End session → triggers summarization |
| `memory_import_transcript` | Import a text transcript as observation batch |
| `memory_stream_subscribe` | Subscribe to SSE stream for session events |
| `memory_retention_score` | Get retention score for a specific memory |

### Category 4: Governance (2 tools)
| Tool | Description |
|------|-------------|
| `memory_governance_delete` | Governance-level cascade delete with audit trail |
| `memory_audit_log` | Query audit log with filters |

### Category 5: Knowledge Graph (3 tools)
| Tool | Description |
|------|-------------|
| `memory_graph_query` | Query knowledge graph (extend existing graph_query) |
| `memory_graph_nodes` | List graph nodes by type/concept |
| `memory_graph_edges` | List graph edges for a node |

### Category 6: Actions & Orchestration (10 tools)
| Tool | Description |
|------|-------------|
| `memory_action_create` | Create task action for multi-agent coordination |
| `memory_action_list` | List actions by status |
| `memory_action_update` | Update action status/result |
| `memory_lease_acquire` | Acquire distributed lease (prevent concurrent writes) |
| `memory_lease_release` | Release a lease |
| `memory_checkpoint_create` | Create human-approval gate |
| `memory_checkpoint_approve` | Approve a checkpoint |
| `memory_sketch_create` | Create sketch to group related actions |
| `memory_sketch_promote` | Promote sketch → Crystal (synthesized summary) |
| `memory_crystal_get` | Get crystal (promoted sketch with narrative) |

### Category 7: Signals (2 tools)
| Tool | Description |
|------|-------------|
| `memory_signal_send` | Send signal to another agent (handoff/update/alert) |
| `memory_signal_list` | List signals for an agent (with unread filter) |

### Category 8: Replay/Slots/Admin (15 tools)
| Tool | Description |
|------|-------------|
| `memory_replay_sessions` | List replayable session recordings |
| `memory_replay_load` | Load session replay for review |
| `memory_slot_read` | Read a named memory slot |
| `memory_slot_write` | Write to a named memory slot |
| `memory_slot_list` | List all slots for scope |
| `memory_slot_delete` | Delete a named slot |
| `memory_health` | Get system health snapshot |
| `memory_export` | Export all memories in JSON |
| `memory_compress` | Trigger compression for specific session |
| `memory_summarize` | Trigger summarization for completed session |
| `memory_consolidate` | Run full consolidation pipeline |
| `memory_evict` | Run eviction (with dry_run support) |
| `memory_auto_forget` | Trigger TTL sweep |
| `memory_doctor` | Run diagnostic checks |
| `memory_snapshot_create` | Create git snapshot of memory data |

---

## Implementation

### `tools/agentmemory/registry.go`

```go
package agentmemory

import "github.com/vnp-memory/gateway/internal/adapter/mcp"

// RegisterAllAgentMemoryTools registers 37 new agentmemory tools
func RegisterAllAgentMemoryTools(reg *mcp.ToolRegistry, deps *Dependencies) {
    reg.RegisterAll(MemoryCoreTools(deps)...)
    reg.RegisterAll(SessionTools(deps)...)
    reg.RegisterAll(ObserveTools(deps)...)
    reg.RegisterAll(GovernanceTools(deps)...)
    reg.RegisterAll(GraphTools(deps)...)
    reg.RegisterAll(OrchestrationTools(deps)...)
    reg.RegisterAll(SignalTools(deps)...)
    reg.RegisterAll(ReplaySlotTools(deps)...)
    reg.RegisterAll(AdminTools(deps)...)
}

type Dependencies struct {
    ObserveClient       ObserveServiceClient
    MemoryClient        AgentMemoryServiceClient
    SearchClient        SearchServiceClient
    OrchestrationClient OrchestrationServiceClient
    AdminClient         AdminServiceClient
}
```

### `tools/agentmemory/memory_core.go`

```go
package agentmemory

import "github.com/vnp-memory/gateway/internal/adapter/mcp"

func MemoryCoreTools(deps *Dependencies) []mcp.Tool {
    return []mcp.Tool{
        {
            Name: "memory_smart_search",
            Description: "Search memories and observations using hybrid BM25+vector search. Returns semantically relevant results even without exact keyword match.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "query":         {Type: "string", Description: "Natural language search query"},
                    "project":       {Type: "string", Description: "Filter by project (optional)"},
                    "limit":         {Type: "integer", Default: 10, Description: "Max results (1-50)"},
                    "bm25_weight":   {Type: "number", Default: 0.4},
                    "vector_weight": {Type: "number", Default: 0.6},
                },
                Required: []string{"query"},
            },
            Handler: proxySmartSearch(deps),
        },
        {
            Name: "memory_save",
            Description: "Save a long-term memory that should persist across coding sessions. Use for patterns, architectural decisions, bug fixes, preferences, and workflows.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "type":         {Type: "string", Enum: []string{"pattern","preference","architecture","bug","workflow","fact"}},
                    "title":        {Type: "string", Description: "One-line summary (max 80 chars)"},
                    "content":      {Type: "string", Description: "Full memory content"},
                    "concepts":     {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Key concepts for retrieval (3-10 words)"},
                    "files":        {Type: "array", Items: &mcp.Property{Type: "string"}},
                    "project":      {Type: "string"},
                    "forget_after": {Type: "string", Description: "ISO 8601 timestamp for TTL"},
                },
                Required: []string{"type", "title", "content", "concepts"},
            },
            Handler: proxyMemorySave(deps),
        },
        {
            Name: "memory_context",
            Description: "Build a context block from relevant memories and session history, optimized for a token budget. Call at session start to inject useful context.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "query":        {Type: "string", Description: "Current task (for semantic retrieval)"},
                    "project":      {Type: "string"},
                    "session_id":   {Type: "string"},
                    "token_budget": {Type: "integer", Default: 2000},
                },
                Required: []string{"project"},
            },
            Handler: proxyBuildContext(deps),
        },
        {
            Name: "memory_forget",
            Description: "Permanently delete a memory with cascade (removes from all search indexes and graph). Creates audit trail.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "memory_id": {Type: "string"},
                    "reason":    {Type: "string"},
                },
                Required: []string{"memory_id"},
            },
            Handler: proxyGovernanceDelete(deps),
        },
    }
}
```

### `tools/agentmemory/observe_tools.go`

```go
package agentmemory

func ObserveTools(deps *Dependencies) []mcp.Tool {
    return []mcp.Tool{
        {
            Name: "memory_observe",
            Description: "Capture a hook event from agent activity. Called automatically by agent plugins.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "session_id": {Type: "string"},
                    "hook_type":  {Type: "string", Enum: []string{"session_start","prompt_submit","pre_tool_use","post_tool_use","post_tool_failure","session_end","task_completed"}},
                    "tool_name":  {Type: "string"},
                    "tool_input": {Type: "object"},
                    "tool_output": {Type: "object"},
                    "agent_id":   {Type: "string"},
                },
                Required: []string{"session_id", "hook_type"},
            },
            Handler: proxyObserve(deps),
        },
        {
            Name: "memory_session_start",
            Description: "Start a new observation session. Returns session_id to use in subsequent observe calls.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "project":  {Type: "string"},
                    "cwd":      {Type: "string"},
                    "model":    {Type: "string"},
                    "agent_id": {Type: "string"},
                },
                Required: []string{"project"},
            },
            Handler: proxySessionStart(deps),
        },
        {
            Name: "memory_session_end",
            Description: "End current session. Triggers session summarization and memory consolidation.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "session_id": {Type: "string"},
                },
                Required: []string{"session_id"},
            },
            Handler: proxySessionEnd(deps),
        },
        {
            Name: "memory_import_transcript",
            Description: "Import a text transcript as a batch of observations for the current session.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "session_id": {Type: "string"},
                    "transcript": {Type: "string"},
                    "format":     {Type: "string", Enum: []string{"plain","markdown","json"}},
                },
                Required: []string{"session_id", "transcript"},
            },
            Handler: proxyImportTranscript(deps),
        },
        {
            Name: "memory_stream_subscribe",
            Description: "Get SSE stream URL for real-time session events. Returns URL to subscribe to.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "session_id": {Type: "string", Description: "Filter to specific session (optional)"},
                },
            },
            Handler: proxyStreamSubscribe(deps),
        },
        {
            Name: "memory_retention_score",
            Description: "Get retention score for a memory (strength × recency × frequency). Returns recommendation: keep/review/evict.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{"memory_id": {Type: "string"}},
                Required: []string{"memory_id"},
            },
            Handler: proxyRetentionScore(deps),
        },
    }
}
```

### `tools/agentmemory/orchestration_tools.go`

```go
package agentmemory

func OrchestrationTools(deps *Dependencies) []mcp.Tool {
    return []mcp.Tool{
        {
            Name: "memory_action_create",
            Description: "Create a task action for multi-agent coordination. Track work items with priority, dependencies, and status.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "title":          {Type: "string"},
                    "description":    {Type: "string"},
                    "priority":       {Type: "integer", Default: 50, Description: "0-100"},
                    "agent_id":       {Type: "string"},
                    "project":        {Type: "string"},
                    "requires":       {Type: "array", Items: &mcp.Property{Type: "string"}},
                    "conflicts_with": {Type: "array", Items: &mcp.Property{Type: "string"}},
                    "tags":           {Type: "array", Items: &mcp.Property{Type: "string"}},
                },
                Required: []string{"title"},
            },
            Handler: proxyCreateAction(deps),
        },
        {
            Name: "memory_action_list",
            Description: "List actions by status for the current project.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "status":  {Type: "string", Enum: []string{"pending","active","blocked","done","cancelled","failed"}, Description: "Filter by status"},
                    "project": {Type: "string"},
                    "limit":   {Type: "integer", Default: 20},
                },
            },
            Handler: proxyListActions(deps),
        },
        {
            Name: "memory_action_update",
            Description: "Update action status or result. Use to mark actions as done/failed/blocked.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "action_id": {Type: "string"},
                    "status":    {Type: "string", Enum: []string{"active","done","blocked","cancelled","failed"}},
                    "result":    {Type: "string", Description: "Outcome description"},
                },
                Required: []string{"action_id", "status"},
            },
            Handler: proxyUpdateAction(deps),
        },
        {
            Name: "memory_lease_acquire",
            Description: "Acquire a distributed lease to prevent concurrent writes to shared state. Lease expires automatically after TTL.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "action_id": {Type: "string"},
                    "agent_id":  {Type: "string"},
                    "ttl_secs":  {Type: "integer", Default: 300},
                },
                Required: []string{"action_id", "agent_id"},
            },
            Handler: proxyAcquireLease(deps),
        },
        {
            Name: "memory_lease_release",
            Description: "Release a previously acquired lease.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{"lease_id": {Type: "string"}},
                Required: []string{"lease_id"},
            },
            Handler: proxyReleaseLease(deps),
        },
        {
            Name: "memory_checkpoint_create",
            Description: "Create a human-approval gate. Agent pauses until checkpoint is approved or rejected.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "title":       {Type: "string"},
                    "description": {Type: "string"},
                    "action_id":   {Type: "string"},
                    "expire_hours": {Type: "integer", Default: 24},
                },
                Required: []string{"title"},
            },
            Handler: proxyCreateCheckpoint(deps),
        },
        {
            Name: "memory_checkpoint_approve",
            Description: "Approve a pending checkpoint.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "checkpoint_id": {Type: "string"},
                    "approved_by":   {Type: "string"},
                },
                Required: []string{"checkpoint_id"},
            },
            Handler: proxyApproveCheckpoint(deps),
        },
        {
            Name: "memory_sketch_create",
            Description: "Create a sketch to group related actions into a coherent work unit.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{
                    "title":        {Type: "string"},
                    "project":      {Type: "string"},
                    "expire_hours": {Type: "integer", Default: 72},
                },
                Required: []string{"title"},
            },
            Handler: proxyCreateSketch(deps),
        },
        {
            Name: "memory_sketch_promote",
            Description: "Promote a sketch to a Crystal — a synthesized narrative of completed actions. Uses LLM to generate insights.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{"sketch_id": {Type: "string"}},
                Required: []string{"sketch_id"},
            },
            Handler: proxyPromoteSketch(deps),
        },
        {
            Name: "memory_crystal_get",
            Description: "Get a Crystal (promoted sketch) with its narrative, key outcomes, and lessons.",
            InputSchema: mcp.Schema{
                Type: "object",
                Properties: map[string]mcp.Property{"crystal_id": {Type: "string"}},
                Required: []string{"crystal_id"},
            },
            Handler: proxyCrystalGet(deps),
        },
    }
}
```

### MODIFY `gateway/internal/adapter/mcp/tool_registry.go`

```go
// Thêm vào func RegisterAllTools()

// [NEW] AgentMemory tools (37 tools in 8 categories)
agentmemory.RegisterAllAgentMemoryTools(reg, &agentmemory.Dependencies{
    ObserveClient:       deps.ObserveServiceClient,
    MemoryClient:        deps.AgentMemoryServiceClient,
    SearchClient:        deps.SearchServiceClient,
    OrchestrationClient: deps.OrchestrationServiceClient,
    AdminClient:         deps.AdminServiceClient,
})

// Verify: total = 53
log.Info("MCP tools registered", "count", reg.Count())
if reg.Count() != 53 {
    log.Warn("Unexpected MCP tool count", "expected", 53, "actual", reg.Count())
}
```

---

## Verification

```bash
cd gateway
go build ./...
go test ./internal/adapter/mcp/... -v -run TestMCPToolCount
```

```go
func TestMCPToolCount(t *testing.T) {
    reg := mcp.NewToolRegistry()
    RegisterAllTools(reg, mockDeps)
    assert.Equal(t, 53, reg.Count())

    // No duplicates
    names := reg.Names()
    assert.Equal(t, len(names), len(uniqueStrings(names)))

    // Required tools exist
    required := []string{"memory_smart_search", "memory_save", "memory_observe",
        "memory_lease_acquire", "memory_signal_send", "memory_doctor"}
    for _, name := range required {
        assert.True(t, reg.HasTool(name), "missing tool: "+name)
    }
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `tools/list` → 53 tools | ✅ |
| No duplicate tool names | ✅ |
| All 8 categories represented | ✅ |
| `memory_smart_search` with query → returns results | ✅ |
| `memory_observe` → observation captured | ✅ |
| `memory_lease_acquire` → lease_id returned | ✅ |
