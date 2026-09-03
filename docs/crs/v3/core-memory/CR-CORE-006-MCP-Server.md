# Change Request: CR-CORE-006 — MCP Server (37+ Tools, Dual Transport)

**CR ID:** CR-CORE-006
**Component:** `backend/gateway` — MCP Server layer
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Core Memory & Integration
**Solution:** [S2 — Unified Memory API](../../../bussiness/solutions/S2-unified-api.md)
**Features:** [F13](../../../features/13-mcp-server-context-injection/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P6-03 | Framework Integrator | Phải viết manual context injection cho mỗi LLM framework |
| PP-P5-02 | IDE Plugin User | IDE plugin phải call REST API thủ công — verbose |
| PP-P5-04 | IDE Plugin User | AI phải re-read files mỗi session vì không có project memory |

**Before:** Developer viết custom integration code cho mỗi framework.
**After:** 37+ MCP tools available — Claude Code, LangChain, AutoGen kết nối ngay.

---

## 2. MCP Tools Index

```
Memory Operations (6):
  memory_store, memory_recall, memory_forget, memory_timeline
  memory_observe, memory_consolidate

OpenViking / Filesystem (8):
  ov_read_file, ov_write_file, ov_grep, ov_search,
  ov_list_dir, ov_create_file, ov_delete_file, ov_get_summary

Profile Operations (5):
  profile_get_context, profile_get_user, profile_flush,
  profile_list_categories, profile_update

Agent Operations (7):
  observe_session_start, observe_session_end, observe_hook,
  observe_list_sessions, observe_replay_session,
  orchestration_acquire_lease, orchestration_send_signal

Graph Operations (5):
  graph_search, graph_add_episode, graph_get_entity,
  graph_get_timeline, graph_set_ontology

Admin Operations (6):
  admin_forget_user, admin_list_tenants, admin_get_health,
  admin_get_metrics, admin_list_api_keys, admin_revoke_key
```

---

## 3. API Endpoints

```
GET  /mcp/sse            → SSE transport (Claude Code, legacy)
POST /mcp/message        → HTTP Streamable transport (new clients)
GET  /mcp/tools          → List all 37+ tools with schemas
```

---

## 4. Thay đổi đề xuất

### 4.1 `backend/gateway/internal/adapter/mcp/server.go` [MODIFY]

```go
// Register all tools
func (s *MCPServer) RegisterTools() {
    s.Register("memory_store", s.handleMemoryStore)
    s.Register("memory_recall", s.handleMemoryRecall)
    s.Register("ov_grep", s.handleOVGrep)
    s.Register("ov_read_file", s.handleOVReadFile)
    s.Register("profile_get_context", s.handleProfileContext)
    s.Register("observe_hook", s.handleObserveHook)
    // ... 31 more tools
}

// Dual transport
mux.HandleFunc("GET /mcp/sse", s.HandleSSE)       // SSE for Claude Code
mux.HandleFunc("POST /mcp/message", s.HandleHTTP)  // HTTP Streamable
```

### 4.2 Context Injection với token budget

```go
// memory_recall với scope + token budget
func (s *MCPServer) handleMemoryRecall(args map[string]any) (string, error) {
    scope := args["scope"].(string) // project | session | global
    tokenBudget := args["token_budget"].(int) // default 2048

    results := s.memoryService.Recall(ctx, RecallRequest{
        Query:       args["query"].(string),
        Scope:       scope,
        TokenBudget: tokenBudget,
    })
    
    return formatForLLM(results, tokenBudget), nil
}
```

---

## 5. Acceptance Criteria

- [ ] 37+ tools registered và accessible qua cả SSE và HTTP Streamable
- [ ] Claude Code kết nối được qua SSE transport
- [ ] `token_budget` parameter enforced (không trả quá budget)
- [ ] 3 agent scopes: `project`, `session`, `global`
- [ ] Tool schemas đúng JSON Schema format (for LLM tool use)
- [ ] `GET /mcp/tools` trả full list với descriptions
