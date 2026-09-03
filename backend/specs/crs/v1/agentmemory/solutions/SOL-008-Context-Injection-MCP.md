# Solution: SOL-008 — Context Injection & Agent Integration (MCP 53 Tools)

**CR ID:** CR-AM-008  
**Solution ID:** SOL-008  
**Priority:** High (Wave 2)  
**Architecture:** EXTEND `gateway/` MCP adapter (16 → 53 tools)

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §4.3`:
- MCP Server chạy trên port `:8082`, hỗ trợ **SSE và HTTP Streamable**.
- Hiện có **16 MCP tools** trong `gateway/internal/adapter/mcp/tool_registry.go`.
- Tool handler pattern: mỗi tool là một handler function map đến một gRPC service call.
- `gateway/internal/adapter/mcp/` folder đã tồn tại với tool definitions.

**Chiến lược:**
1. Thêm **37 tools mới** vào MCP registry (16 → 53).
2. Tổ chức theo 8 categories trong subfolder `tools/agentmemory/`.
3. Context injection implementation trong `observe-search` service (đã có trong SOL-003).
4. Agent plugin configs expose qua Admin API.

---

## 2. Giải pháp

### 2.1. MCP Tool Registry Expansion

**Existing tools (16):** memory_store, memory_recall, memory_search, memory_timeline, memory_profile, memory_forget, graph_query, ov_read_file, ov_write_file, ov_search, ov_list_dir, ov_grep, ov_tree, ov_session_commit, ov_ingest, ov_delete.

**New agentmemory tools (37):**

```go
// gateway/internal/adapter/mcp/tools/agentmemory/registry.go

// Category 1: Memory Core (4 tools)
// memory_smart_search, memory_save, memory_recall (extend existing), memory_forget (extend existing)

// Category 2: Sessions (3 tools)
// memory_sessions, memory_session_detail, memory_context

// Category 3: Observe (6 tools)
// memory_observe, memory_session_start, memory_session_end,
// memory_import_transcript, memory_stream_subscribe, memory_observations

// Category 4: Governance (2 tools)
// memory_governance_delete, memory_audit_log

// Category 5: Knowledge Graph (3 tools)
// memory_graph_query (extend existing graph_query), memory_graph_nodes, memory_graph_edges

// Category 6: Actions & Orchestration (10 tools)
// memory_action_create, memory_action_list, memory_action_update,
// memory_lease_acquire, memory_lease_release,
// memory_checkpoint_create, memory_checkpoint_approve,
// memory_sketch_create, memory_sketch_promote, memory_crystal_get

// Category 7: Signals (2 tools)
// memory_signal_send, memory_signal_list

// Category 8: Replay & Slots + Admin (15 tools)
// memory_replay_sessions, memory_replay_load,
// memory_slot_read, memory_slot_write, memory_slot_list, memory_slot_delete,
// memory_health, memory_export,
// memory_compress, memory_summarize, memory_consolidate,
// memory_evict, memory_auto_forget, memory_retention_score,
// memory_procedural_list, memory_lessons_list, memory_insights_list,
// memory_doctor, memory_snapshot_create, memory_snapshot_list,
// memory_routine_create, memory_routine_execute, memory_sentinel_create
```

### 2.2. Tool Schema Examples

```go
// gateway/internal/adapter/mcp/tools/agentmemory/memory_core.go

package agentmemory

import "github.com/vnp-memory/gateway/internal/adapter/mcp"

var MemoryCoreTools = []mcp.Tool{
    {
        Name: "memory_smart_search",
        Description: "Search memories and observations using hybrid BM25+vector search. Returns semantically relevant results even without keyword match. Use for: finding past solutions, discovering patterns, recalling decisions.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query":        {Type: "string", Description: "Natural language search query"},
                "project":      {Type: "string", Description: "Filter by project (optional)"},
                "limit":        {Type: "integer", Default: 10, Description: "Max results (default 10)"},
                "bm25_weight":  {Type: "number", Default: 0.4},
                "vector_weight": {Type: "number", Default: 0.6},
            },
            Required: []string{"query"},
        },
        Handler: proxySmartSearch,
    },
    {
        Name: "memory_save",
        Description: "Save a long-term memory (fact, preference, architecture decision, bug fix, workflow, or pattern) that should persist across coding sessions.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "type":     {Type: "string", Enum: []string{"pattern","preference","architecture","bug","workflow","fact"}, Description: "Memory type"},
                "title":    {Type: "string", Description: "One-line summary (max 80 chars)"},
                "content":  {Type: "string", Description: "Full memory content"},
                "concepts": {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Key concepts for retrieval (3-10 words)"},
                "files":    {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Affected file paths"},
                "project":  {Type: "string"},
                "forget_after": {Type: "string", Description: "ISO 8601 timestamp for TTL (optional)"},
            },
            Required: []string{"type", "title", "content", "concepts"},
        },
        Handler: proxyMemorySave,
    },
    {
        Name: "memory_context",
        Description: "Build a context block from relevant memories and session history, optimized for a given token budget. Call at session start to inject useful context.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query":        {Type: "string", Description: "Current task description (used for semantic retrieval)"},
                "project":      {Type: "string"},
                "session_id":   {Type: "string"},
                "token_budget": {Type: "integer", Default: 2000, Description: "Max tokens to return"},
            },
            Required: []string{"project"},
        },
        Handler: proxyBuildContext,
    },
    {
        Name: "memory_forget",
        Description: "Permanently delete a memory with cascade (removes from search indexes and graph). Creates an audit trail.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "memory_id":    {Type: "string"},
                "reason":       {Type: "string", Description: "Reason for deletion"},
            },
            Required: []string{"memory_id"},
        },
        Handler: proxyGovernanceDelete,
    },
}
```

```go
// gateway/internal/adapter/mcp/tools/agentmemory/observe_tools.go

var ObserveTools = []mcp.Tool{
    {
        Name: "memory_observe",
        Description: "Capture a hook event from agent activity. Called automatically by agent plugins, but can be called manually.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "session_id":  {Type: "string"},
                "hook_type":   {Type: "string", Enum: []string{"session_start","prompt_submit","pre_tool_use","post_tool_use","post_tool_failure","session_end","task_completed"}},
                "tool_name":   {Type: "string"},
                "tool_input":  {Type: "object"},
                "tool_output": {Type: "object"},
                "agent_id":    {Type: "string"},
            },
            Required: []string{"session_id", "hook_type"},
        },
        Handler: proxyObserve,
    },
    {
        Name: "memory_session_start",
        Description: "Start a new observation session. Returns session_id to use in subsequent observe calls.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "project":    {Type: "string"},
                "cwd":        {Type: "string"},
                "model":      {Type: "string"},
                "agent_id":   {Type: "string"},
            },
            Required: []string{"project"},
        },
        Handler: proxySessionStart,
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
        Handler: proxySessionEnd,
    },
    // ... memory_import_transcript, memory_stream_subscribe, memory_observations
}
```

```go
// gateway/internal/adapter/mcp/tools/agentmemory/orchestration_tools.go

var OrchestrationTools = []mcp.Tool{
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
                "requires":       {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Action IDs that must complete first"},
                "conflicts_with": {Type: "array", Items: &mcp.Property{Type: "string"}},
                "tags":           {Type: "array", Items: &mcp.Property{Type: "string"}},
            },
            Required: []string{"title"},
        },
        Handler: proxyCreateAction,
    },
    {
        Name: "memory_lease_acquire",
        Description: "Acquire a distributed lease to prevent concurrent writes. Use before modifying shared state.",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "action_id": {Type: "string"},
                "agent_id":  {Type: "string"},
                "ttl_secs":  {Type: "integer", Default: 300},
            },
            Required: []string{"action_id", "agent_id"},
        },
        Handler: proxyAcquireLease,
    },
    {
        Name: "memory_signal_send",
        Description: "Send a signal to another agent (handoff, update, alert, request).",
        InputSchema: mcp.Schema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "from":     {Type: "string", Description: "Sender agent ID"},
                "to":       {Type: "string", Description: "Recipient agent ID"},
                "type":     {Type: "string", Enum: []string{"handoff","update","cancel","request","response","alert"}},
                "content":  {Type: "string"},
                "reply_to": {Type: "string", Description: "Signal ID being replied to"},
            },
            Required: []string{"from", "to", "type", "content"},
        },
        Handler: proxySendSignal,
    },
    // ... other orchestration tools
}
```

### 2.3. Tool Registry Registration

```go
// gateway/internal/adapter/mcp/tool_registry.go — MODIFY

func RegisterAllTools(reg *mcp.ToolRegistry, deps *Dependencies) {
    // Existing 16 tools
    reg.Register(memory_store, memory_recall, ...)  // existing
    
    // [NEW] AgentMemory tools (37 tools, grouped by category)
    reg.RegisterAll(agentmemory.MemoryCoreTools...)
    reg.RegisterAll(agentmemory.SessionTools...)
    reg.RegisterAll(agentmemory.ObserveTools...)
    reg.RegisterAll(agentmemory.GovernanceTools...)
    reg.RegisterAll(agentmemory.GraphTools...)
    reg.RegisterAll(agentmemory.OrchestrationTools...)
    reg.RegisterAll(agentmemory.SignalTools...)
    reg.RegisterAll(agentmemory.ReplaySlotTools...)
    reg.RegisterAll(agentmemory.AdminTools...)
    
    // Verify: total should be 53
    log.Info("MCP tools registered", "count", reg.Count())
}
```

### 2.4. Tool Handler Proxy Pattern

```go
// gateway/internal/adapter/mcp/tools/agentmemory/proxy.go

// Proxy pattern: MCP tool call → HTTP to downstream service

func proxySmartSearch(ctx context.Context, input map[string]any) (any, error) {
    query, _ := input["query"].(string)
    limit, _ := input["limit"].(int)
    if limit == 0 { limit = 10 }
    project, _ := input["project"].(string)
    
    req := observesearch.SmartSearchRequest{
        Query:   query,
        Limit:   limit,
        Project: project,
        Weights: observesearch.ScoreWeights{BM25: 0.4, Vector: 0.6},
    }
    
    // Forward via gRPC (InProcessRegistry)
    resp, err := deps.ObserveSearchClient.SmartSearch(ctx, &req)
    if err != nil { return nil, err }
    
    return resp, nil
}

func proxyMemorySave(ctx context.Context, input map[string]any) (any, error) {
    req := agentmemory.RememberAgentRequest{
        Type:     agentmemory.MemoryType(input["type"].(string)),
        Title:    input["title"].(string),
        Content:  input["content"].(string),
        Concepts: toStringSlice(input["concepts"]),
        Files:    toStringSlice(input["files"]),
        Project:  getString(input, "project"),
        TenantID: extractTenantFromContext(ctx),
    }
    if fa, ok := input["forget_after"].(string); ok && fa != "" {
        t, _ := time.Parse(time.RFC3339, fa)
        req.ForgetAfter = &t
    }
    
    resp, err := deps.MemoryServiceClient.RememberAgent(ctx, &req)
    if err != nil { return nil, err }
    return map[string]any{
        "memory_id": resp.MemoryID,
        "version":   resp.Version,
        "superseded": resp.Superseded,
    }, nil
}
```

### 2.5. Context Injection — Auto-inject on Session Start

```go
// services/observe-service/internal/observe/pipeline.go — MODIFY step 1

// When hookType = session_start AND AGENTMEMORY_INJECT_CONTEXT=true:
// 1. Call observe-search BuildContext (gRPC, internal)
// 2. Return context blocks in ObserveResponse
// 3. Agent plugin reads context and prepends to first message

func (p *Pipeline) Execute(ctx context.Context, req ObserveRequest) (*ObserveResponse, error) {
    resp := &ObserveResponse{}
    
    // ... existing pipeline steps ...
    
    // Context injection (session_start hook only)
    if req.HookType == HookSessionStart && p.config.InjectContext {
        ctxResp, err := p.searchClient.BuildContext(ctx, BuildContextRequest{
            Project:     req.Project,
            SessionID:   req.SessionID,
            TokenBudget: p.config.TokenBudget,
        })
        if err == nil {
            resp.InjectedContext = ctxResp.Formatted
            resp.ContextTokens  = ctxResp.TotalTokens
        }
        // Non-fatal: log if context injection fails, don't block observation
    }
    
    return resp, nil
}
```

### 2.6. Multi-Agent Scoping

```go
// gateway/internal/domain/entity.go — EXTEND AuthContext

type AuthContext struct {
    TenantID  string
    UserID    string
    Roles     []string
    AgentID   string    // [NEW] from X-Agent-ID header
    Project   string    // [NEW] from X-Project header or query param
    AgentScope string   // [NEW] "shared" | "isolated"
}

// gateway/internal/adapter/handler/middleware.go — MODIFY

func extractAgentContext(r *http.Request) AgentScopeContext {
    agentID := r.Header.Get("X-Agent-ID")
    if agentID == "" { agentID = r.URL.Query().Get("agent_id") }
    
    project := r.Header.Get("X-Project")
    if project == "" { project = r.URL.Query().Get("project") }
    
    scope := r.Header.Get("X-Agent-Scope")
    if scope == "" { scope = os.Getenv("AGENTMEMORY_AGENT_SCOPE") }
    if scope == "" { scope = "shared" }
    
    return AgentScopeContext{AgentID: agentID, Project: project, Scope: scope}
}
```

### 2.7. Agent Plugin Config API

```go
// services/vnp-platform/internal/usecase/admin/plugin.go

const claudeCodePluginConfig = `{
  "mcpServers": {
    "agentmemory": {
      "type": "http",
      "url": "http://localhost:8082",
      "headers": {
        "X-Project": "{{project}}",
        "X-Agent-ID": "claude-code"
      }
    }
  },
  "hooks": {
    "session_start": {
      "url": "http://localhost:8080/v1/observe",
      "method": "POST",
      "body": {"hookType": "session_start", "project": "{{project}}"}
    },
    "post_tool_use": {
      "url": "http://localhost:8080/v1/observe",
      "method": "POST"
    },
    "stop": {
      "url": "http://localhost:8080/v1/observe/session/end",
      "method": "POST"
    }
  }
}`

func (uc *PluginUseCase) GetConfig(ctx context.Context, agentType string) (string, error) {
    cfg := pluginConfigs[agentType]
    project := extractProjectFromCtx(ctx)
    cfg = strings.ReplaceAll(cfg, "{{project}}", project)
    return cfg, nil
}

func (uc *PluginUseCase) Install(ctx context.Context, req InstallPluginRequest) error {
    // Write plugin config to agent-specific location
    // Claude Code: ~/.claude/settings.json
    // Codex: ~/.codex/settings.json
    // OpenCode: ~/.opencode/settings.json
    configPath := agentConfigPaths[req.AgentType]
    // ... merge with existing config, write file
}
```

### 2.8. MCP Tool Count Verification

```go
// gateway/internal/adapter/mcp/tool_registry_test.go

func TestMCPToolCount(t *testing.T) {
    reg := mcp.NewToolRegistry()
    RegisterAllTools(reg, mockDeps)
    
    assert.Equal(t, 53, reg.Count(), "MCP server must expose exactly 53 agentmemory tools")
    
    // Verify no duplicate names
    names := reg.Names()
    assert.Equal(t, len(names), len(unique(names)), "No duplicate tool names")
    
    // Verify all required categories present
    assert.True(t, reg.HasTool("memory_smart_search"))
    assert.True(t, reg.HasTool("memory_save"))
    assert.True(t, reg.HasTool("memory_observe"))
    assert.True(t, reg.HasTool("memory_lease_acquire"))
    assert.True(t, reg.HasTool("memory_signal_send"))
    assert.True(t, reg.HasTool("memory_doctor"))
}
```

### 2.9. Files

#### [NEW]

| File | Mô tả |
|------|-------|
| `gateway/internal/adapter/mcp/tools/agentmemory/memory_core.go` | memory_smart_search, memory_save, memory_context, memory_forget |
| `gateway/internal/adapter/mcp/tools/agentmemory/session_tools.go` | memory_sessions, memory_session_detail, memory_context |
| `gateway/internal/adapter/mcp/tools/agentmemory/observe_tools.go` | 6 observe tools |
| `gateway/internal/adapter/mcp/tools/agentmemory/governance_tools.go` | 2 governance tools |
| `gateway/internal/adapter/mcp/tools/agentmemory/graph_tools.go` | 3 graph tools |
| `gateway/internal/adapter/mcp/tools/agentmemory/orchestration_tools.go` | 10 orchestration tools |
| `gateway/internal/adapter/mcp/tools/agentmemory/signal_tools.go` | 2 signal tools |
| `gateway/internal/adapter/mcp/tools/agentmemory/replay_slot_tools.go` | 8 replay/slot tools |
| `gateway/internal/adapter/mcp/tools/agentmemory/admin_tools.go` | 15 admin tools |
| `gateway/internal/adapter/mcp/tools/agentmemory/proxy.go` | Proxy handler implementations |
| `gateway/internal/adapter/mcp/tools/agentmemory/registry.go` | RegisterAllAgentMemoryTools() |
| `services/vnp-platform/internal/usecase/admin/plugin.go` | Plugin config generator + installer |

#### [MODIFY]

| File | Thay đổi |
|------|---------|
| `gateway/internal/adapter/mcp/tool_registry.go` | Register agentmemory tools |
| `gateway/internal/domain/entity.go` | Add AgentID, Project, AgentScope to AuthContext |
| `gateway/internal/infra/middleware/auth.go` | Extract X-Agent-ID, X-Project, X-Agent-Scope headers |
| `services/observe-service/internal/observe/pipeline.go` | Context injection on session_start |
| `apps/memory/configs/config.yaml` | inject_context: false, agent_scope: shared |

---

## 3. Acceptance Criteria Mapping

| AC từ CR-AM-008 | Covered by |
|-----------------|------------|
| tools/list → 53 agentmemory tools | TestMCPToolCount() |
| memory_smart_search "auth middleware" → memories | proxySmartSearch() |
| memory_save type="architecture" → is_latest:true | proxyMemorySave() → RememberAgent() |
| memory_governance_delete cascade | proxyGovernanceDelete() → GovernanceDelete() |
| INJECT_CONTEXT=true → session_start builds context | pipeline.go session_start branch |
| GET /admin/plugin/claude-code → valid JSON | plugin.go GetConfig("claude-code") |
| SCOPE=isolated → only same agent_id | search filter agentID in isolated mode |
| SCOPE=shared → all project memories | search without agentID filter |
