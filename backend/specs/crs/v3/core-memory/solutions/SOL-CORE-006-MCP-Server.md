# SOL-CORE-006 — Solution: MCP Server (37+ Tools, Dual Transport)

| Field | Value |
|---|---|
| **Solution ID** | SOL-CORE-006 |
| **CR** | [CR-CORE-006](../../../../docs/crs/v3/core-memory/CR-CORE-006-MCP-Server.md) |
| **TDD ref** | [01-gateway.md](../../../tdd/architecture/01-gateway.md) §MCP |
| **Status** | Open |
| **Priority** | 🟡 High |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** MCP server with 43 tools (agentmemory+cognee); graphiti/admin tools pending
---

## 1. Phân tích

Gateway đã có MCP server (`gateway/adapter/mcp/server.go`) với **22 tools**. Cần thêm **15 tools** nữa:
- Agent Operations: 5 tools (observe, orchestration)
- Graph Operations: 5 tools (graphiti)
- Admin Operations: 5 tools (forget, health, metrics)

---

## 2. Giải pháp

### 2.1 `gateway/adapter/mcp/server.go` [MODIFY]

```go
// Thêm vào RegisterTools():

// Agent Operations (observe-service)
s.Register("observe_session_start", s.handleObserveSessionStart,
    mcp.Schema{"session_id?": "string", "project": "string", "model": "string"})
s.Register("observe_session_end", s.handleObserveSessionEnd,
    mcp.Schema{"session_id": "string"})
s.Register("observe_hook", s.handleObserveHook,
    mcp.Schema{"session_id": "string", "hook_type": "string", "payload": "object"})
s.Register("observe_list_sessions", s.handleObserveListSessions,
    mcp.Schema{"limit?": "integer"})
s.Register("observe_replay_session", s.handleObserveReplaySession,
    mcp.Schema{"session_id": "string", "from_index?": "integer"})

// Orchestration (orchestration-service)
s.Register("orchestration_acquire_lease", s.handleAcquireLease,
    mcp.Schema{"resource_id": "string", "ttl_seconds?": "integer"})
s.Register("orchestration_send_signal", s.handleSendSignal,
    mcp.Schema{"agent_id": "string", "signal_type": "string", "payload?": "object"})

// Graph Operations (graphiti-search)
s.Register("graph_search", s.handleGraphSearch,
    mcp.Schema{"query": "string", "limit?": "integer"})
s.Register("graph_add_episode", s.handleGraphAddEpisode,
    mcp.Schema{"content": "string", "source": "string"})
s.Register("graph_get_timeline", s.handleGraphTimeline,
    mcp.Schema{"from?": "string", "to?": "string", "limit?": "integer"})

// Admin Operations
s.Register("admin_get_health", s.handleAdminHealth, mcp.Schema{})
s.Register("admin_get_metrics", s.handleAdminMetrics,
    mcp.Schema{"service?": "string"})
s.Register("admin_forget_user", s.handleAdminForget,
    mcp.Schema{"user_id": "string", "reason": "string"})
s.Register("admin_list_api_keys", s.handleListAPIKeys, mcp.Schema{})
s.Register("admin_revoke_key", s.handleRevokeKey,
    mcp.Schema{"key_id": "string"})
```

### 2.2 Handler implementations

```go
func (s *MCPServer) handleObserveHook(ctx context.Context, params map[string]any) (any, error) {
    conn := s.registry.Get("observe-service")
    client := observepb.NewObserveServiceClient(conn)
    res, err := client.Observe(ctx, &observepb.ObserveRequest{
        SessionId: params["session_id"].(string),
        HookType:  params["hook_type"].(string),
        Payload:   mustJSON(params["payload"]),
    })
    return map[string]any{"hook_id": res.GetHookId()}, err
}
```

---

## 3. File Changes

| File | Action |
|---|---|
| `gateway/adapter/mcp/server.go` | MODIFY — register 15 new tools |
| `gateway/adapter/mcp/handlers_agent.go` | NEW — observe + orchestration handlers |
| `gateway/adapter/mcp/handlers_graph.go` | NEW — graphiti handlers |
| `gateway/adapter/mcp/handlers_admin.go` | NEW — admin handlers |

---

## 4. Acceptance Criteria

- [ ] `GET /mcp/tools` returns 37+ tools with JSON schemas
- [ ] SSE transport: `GET /mcp/sse` (Claude Code compatible)
- [ ] HTTP Streamable: `POST /mcp/message` (new MCP spec)
- [ ] All 15 new tools có integration test
- [ ] Tool schemas validate input parameters
