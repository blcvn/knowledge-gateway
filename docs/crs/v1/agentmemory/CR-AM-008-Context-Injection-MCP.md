# Change Request: CR-AM-008 — Context Injection & Agent Integration

**CR ID:** CR-AM-008  
**Component:** `services/observe-search` [EXTEND] | `gateway` [EXTEND] | MCP Server  
**Priority:** High  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.1, §6.7, SRS FR-CTX-001..002, §7.2 MCP (53 tools)  
**Spec:** `references/agentmemory/specs/services/gateway-service/spec.md`

---

## 1. Mô tả

Triển khai **Context Injection** cho AI agents và mở rộng MCP Server để expose đầy đủ các tools cho agent integration. Bao gồm:
1. **Auto Context Injection** — inject relevant memories vào agent context khi session start (opt-in).
2. **53 MCP Tools** — đầy đủ tool surface để agent MCP clients gọi vào.
3. **Agent Plugin Installation** — hỗ trợ Claude Code hooks, Codex CLI hooks, OpenCode hooks.
4. **Multi-agent scoping** — `agent_id` + `project` filter cho isolation/sharing.

---

## 2. Vấn đề hiện tại

`gateway/internal/adapter/mcp/tool_registry.go` hiện có 6 tools cơ bản. agentmemory cần **53 MCP tools** nhóm theo 8 categories để đạt full agent integration parity.

---

## 3. Thay đổi đề xuất

### 3.1. Context Injection

```go
// services/observe-search/internal/search/context_builder.go

// POST /search/context
// Called by: session_start hook (when AGENTMEMORY_INJECT_CONTEXT=true)

type ContextRequest struct {
    Query       string `json:"query,omitempty"`
    Project     string `json:"project"`
    SessionID   string `json:"session_id"`
    AgentID     string `json:"agent_id,omitempty"`
    TokenBudget int    `json:"token_budget"` // default 2000
}

type ContextResponse struct {
    Blocks      []ContextBlock `json:"blocks"`
    TotalTokens int            `json:"total_tokens"`
    Formatted   string         `json:"formatted"` // ready-to-inject text
}

// ContextBlock priority:
// P1: Recent high-strength memories (last 30 days, strength > 0.5)
// P2: Last 3 session summaries
// P3: Relevant observations (if query provided, via smart search)

// Token budget: estimated as len(content) / 4
// WARNING: context injection adds ~4000 chars / 2000 tokens per agent turn
// Enable via: AGENTMEMORY_INJECT_CONTEXT=true (default: false)
```

### 3.2. 53 MCP Tools — Full Registry

MCP Server tại port `:8082` (gateway MCP adapter), thêm agentmemory-specific tools:

#### Category 1: Memory Core (4 tools)
```
memory_smart_search    → POST /search/smart
memory_save            → POST /memory/agent/remember  
memory_recall          → POST /search/context
memory_forget          → DELETE /memory/agent/{id}/governance
```

#### Category 2: Sessions (3 tools)
```
memory_sessions        → GET /sessions
memory_session_detail  → GET /sessions/{id}
memory_context         → POST /search/context
```

#### Category 3: Observe (6 tools)
```
memory_observe         → POST /observe
memory_session_start   → POST /observe/session/start
memory_session_end     → POST /observe/session/end
memory_import_transcript → POST /sessions/import
memory_stream_subscribe → GET /stream (SSE)
memory_observations    → GET /sessions/{id}/observations
```

#### Category 4: Governance (2 tools)
```
memory_governance_delete → DELETE /memory/agent/{id}/governance
memory_audit_log         → GET /memory/audit
```

#### Category 5: Knowledge Graph (3 tools)
```
memory_graph_query     → GET /graph
memory_graph_nodes     → GET /graph/nodes
memory_graph_edges     → GET /graph/edges
```

#### Category 6: Actions & Orchestration (10 tools)
```
memory_action_create   → POST /orchestration/actions
memory_action_list     → GET /orchestration/actions
memory_action_update   → PATCH /orchestration/actions/{id}
memory_lease_acquire   → POST /orchestration/leases/acquire
memory_lease_release   → POST /orchestration/leases/release
memory_checkpoint_create → POST /orchestration/checkpoints
memory_checkpoint_approve → POST /orchestration/checkpoints/{id}/approve
memory_sketch_create   → POST /orchestration/sketches
memory_sketch_promote  → POST /orchestration/sketches/{id}/promote
memory_crystal_get     → GET /orchestration/crystals/{id}
```

#### Category 7: Signals (2 tools)
```
memory_signal_send     → POST /orchestration/signals/send
memory_signal_list     → GET /orchestration/signals
```

#### Category 8: Replay & Slots (8 tools)
```
memory_replay_sessions → GET /sessions (for replay listing)
memory_replay_load     → GET /sessions/{id}/replay
memory_slot_read       → GET /memory/slots/{scope}/{label}
memory_slot_write      → POST /memory/slots/{scope}/{label}
memory_slot_list       → GET /memory/slots
memory_slot_delete     → DELETE /memory/slots/{scope}/{label}
memory_health          → GET /health
memory_export          → GET /admin/export
```

#### Additional Tools (15 tools to reach 53):
```
memory_compress            → POST /memory/compress
memory_summarize           → POST /memory/summarize
memory_consolidate         → POST /memory/consolidate (admin)
memory_evict               → POST /memory/agent/evict
memory_auto_forget         → POST /memory/agent/auto-forget
memory_retention_score     → GET /memory/agent/{id}/retention
memory_procedural_list     → GET /memory/procedural
memory_lessons_list        → GET /memory/lessons
memory_insights_list       → GET /memory/insights
memory_doctor              → GET /admin/doctor
memory_snapshot_create     → POST /admin/snapshot
memory_snapshot_list       → GET /admin/snapshots
memory_routine_create      → POST /orchestration/routines
memory_routine_execute     → POST /orchestration/routines/{id}/execute
memory_sentinel_create     → POST /orchestration/sentinels
```

### 3.3. Agent Plugin Support

**Claude Code Plugin** (`.claude-plugin/` directory structure):

```json
// .claude-plugin/settings.json
{
  "hooks": {
    "session_start": {
      "url": "http://localhost:8080/v1/observe",
      "method": "POST",
      "hookType": "session_start"
    },
    "post_tool_use": {
      "url": "http://localhost:8080/v1/observe",
      "method": "POST",
      "hookType": "post_tool_use"
    },
    "stop": {
      "url": "http://localhost:8080/v1/observe/session/end",
      "method": "POST",
      "hookType": "stop"
    }
  }
}
```

**[NEW]** `GET /v1/admin/plugin/claude-code` → trả về plugin config JSON cho user copy-paste.  
**[NEW]** `GET /v1/admin/plugin/codex` → Codex CLI plugin config.  
**[NEW]** `POST /v1/admin/plugin/install` → auto-install hooks (requires fs write permission to agent config dir).

### 3.4. Multi-Agent Scoping

```go
// All observe/memory/search endpoints accept optional:
// Header: X-Agent-ID: cursor-dev-sam
// Query: ?project=my-app&agent_id=cursor-dev-sam

// Scoping modes (AGENTMEMORY_AGENT_SCOPE):
// "shared" (default): tag memories with agent_id, but search includes all
// "isolated": filter recall to agent_id only

// Memory save: always tags with agent_id from context
// Smart search: if scope=isolated, adds agent_id filter
```

### 3.5. Gateway MCP Tool Schema Example

```go
// gateway/internal/adapter/mcp/tools/agentmemory/memory_core.go

var memorySaveToolSchema = mcp.Tool{
    Name: "memory_save",
    Description: "Save a memory (fact, preference, architecture decision, bug fix, workflow, or pattern) that should persist across sessions.",
    InputSchema: mcp.Schema{
        Type: "object",
        Properties: map[string]mcp.Property{
            "type":     {Type: "string", Enum: []string{"pattern","preference","architecture","bug","workflow","fact"}, Description: "Memory type"},
            "title":    {Type: "string", Description: "One-line memory title"},
            "content":  {Type: "string", Description: "Full memory content"},
            "concepts": {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Key concepts for retrieval"},
            "project":  {Type: "string"},
            "forget_after": {Type: "string", Description: "ISO 8601 timestamp for TTL expiry"},
        },
        Required: []string{"type", "title", "content", "concepts"},
    },
    Handler: proxyMemorySave,
}

var memorySmartSearchSchema = mcp.Tool{
    Name: "memory_smart_search",
    Description: "Search memories and observations using hybrid BM25+vector+graph search. Returns semantically relevant results even without keyword match.",
    InputSchema: mcp.Schema{
        Type: "object",
        Properties: map[string]mcp.Property{
            "query":   {Type: "string"},
            "project": {Type: "string"},
            "limit":   {Type: "integer", Default: 10},
        },
        Required: []string{"query"},
    },
    Handler: proxySmartSearch,
}
```

---

## 4. Acceptance Criteria

- [x] `tools/list` trên MCP server trả về đủ 53 agentmemory tools (theo đúng category breakdown).
- [x] MCP tool `memory_smart_search` với query "auth middleware" trả về relevant memories.
- [x] MCP tool `memory_save` với type `"architecture"` tạo AgentMemory entry với `is_latest: true`.
- [x] MCP tool `memory_governance_delete` cascade xóa khỏi search indexes.
- [x] Context injection (khi `AGENTMEMORY_INJECT_CONTEXT=true`): `session_start` hook trigger `POST /search/context` và kết quả được prepend vào session context.
- [x] `GET /v1/admin/plugin/claude-code` trả về valid JSON config có thể dùng ngay với Claude Code.
- [x] Agent với `AGENTMEMORY_AGENT_SCOPE=isolated` chỉ recall memories có cùng `agent_id`.
- [x] Agent với `AGENTMEMORY_AGENT_SCOPE=shared` recall all memories trong project.
