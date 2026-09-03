# TASK-CORE-009 — MCP Agent Tools (observe + orchestration)

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-009 |
| **Wave** | 3 |
| **Solution** | [SOL-CORE-006](../solutions/SOL-CORE-006-MCP-Server.md) §2.1 |
| **Component** | `gateway/adapter/mcp/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-CORE-002 |
| **Estimated** | 5h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** gateway/adapter/mcp/tools/agentmemory: 37 tools (memory core/session/observe) registered
---

## Mục tiêu

Register 7 Agent Operation MCP tools: 5 observe-service + 2 orchestration-service.

---

## Công việc cụ thể

### `gateway/adapter/mcp/handlers_agent.go` [NEW]

```go
package mcp

// observe_session_start — POST /v1/observe/sessions
func (h *Handlers) handleObserveSessionStart(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("observe-service")
    client := observepb.NewObserveServiceClient(conn)
    resp, err := client.StartSession(ctx, &observepb.StartSessionRequest{
        TenantId:  tenant.FromContext(ctx),
        AgentId:   stringParam(params, "agent_id"),
        Project:   stringParam(params, "project"),
        Model:     stringParam(params, "model"),
    })
    if err != nil { return nil, err }
    return map[string]string{"session_id": resp.SessionId}, nil
}

// observe_session_end
func (h *Handlers) handleObserveSessionEnd(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("observe-service")
    client := observepb.NewObserveServiceClient(conn)
    _, err := client.EndSession(ctx, &observepb.EndSessionRequest{
        SessionId: params["session_id"].(string),
    })
    return map[string]bool{"success": err == nil}, err
}

// observe_hook — submit hook event
func (h *Handlers) handleObserveHook(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("observe-service")
    client := observepb.NewObserveServiceClient(conn)
    payload, _ := json.Marshal(params["payload"])
    resp, err := client.Observe(ctx, &observepb.ObserveRequest{
        SessionId: params["session_id"].(string),
        HookType:  params["hook_type"].(string),
        Payload:   string(payload),
    })
    if err != nil { return nil, err }
    return map[string]string{"observation_id": resp.ObservationId}, nil
}

// observe_list_sessions
func (h *Handlers) handleObserveListSessions(ctx context.Context, params map[string]any) (any, error) { ... }

// observe_replay_session
func (h *Handlers) handleObserveReplaySession(ctx context.Context, params map[string]any) (any, error) { ... }

// orchestration_acquire_lease
func (h *Handlers) handleAcquireLease(ctx context.Context, params map[string]any) (any, error) {
    conn := h.registry.Get("orchestration-service")
    client := orchpb.NewOrchestrationServiceClient(conn)
    ttl := int64(30)
    if v, ok := params["ttl_seconds"]; ok { ttl = int64(v.(float64)) }
    resp, err := client.AcquireLease(ctx, &orchpb.AcquireLeaseRequest{
        ResourceId: params["resource_id"].(string),
        TtlSeconds: ttl,
        TenantId:   tenant.FromContext(ctx),
    })
    if err != nil { return nil, err }
    return map[string]string{"lease_id": resp.LeaseId}, nil
}

// orchestration_send_signal
func (h *Handlers) handleSendSignal(ctx context.Context, params map[string]any) (any, error) { ... }
```

### `gateway/adapter/mcp/server.go` [MODIFY] — register tools

```go
// Add to RegisterTools():
s.Register("observe_session_start", h.handleObserveSessionStart,
    mcp.Schema{"agent_id": "string", "project?": "string", "model?": "string"})
s.Register("observe_session_end", h.handleObserveSessionEnd,
    mcp.Schema{"session_id": "string"})
s.Register("observe_hook", h.handleObserveHook,
    mcp.Schema{"session_id": "string", "hook_type": "string", "payload?": "object"})
s.Register("observe_list_sessions", h.handleObserveListSessions,
    mcp.Schema{"limit?": "integer"})
s.Register("observe_replay_session", h.handleObserveReplaySession,
    mcp.Schema{"session_id": "string", "from_index?": "integer"})
s.Register("orchestration_acquire_lease", h.handleAcquireLease,
    mcp.Schema{"resource_id": "string", "ttl_seconds?": "integer"})
s.Register("orchestration_send_signal", h.handleSendSignal,
    mcp.Schema{"agent_id": "string", "signal_type": "string", "payload?": "object"})
```

---

## Acceptance Criteria

- [ ] 7 tools registered in MCP server
- [ ] `observe_hook` dispatches to observe-service correctly
- [ ] `orchestration_acquire_lease` returns lease_id on success
- [ ] `orchestration_acquire_lease` returns error on conflict
- [ ] Integration test for each tool

## Files

```
gateway/adapter/mcp/handlers_agent.go  [NEW]
gateway/adapter/mcp/server.go          [MODIFY — register 7 tools]
```
