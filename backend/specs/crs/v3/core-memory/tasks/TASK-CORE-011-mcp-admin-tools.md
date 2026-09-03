# TASK-CORE-011 — MCP Admin Tools

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-011 |
| **Wave** | 3 |
| **Solution** | [SOL-CORE-006](../solutions/SOL-CORE-006-MCP-Server.md) §2.1 |
| **Component** | `gateway/adapter/mcp/handlers_admin.go` |
| **Priority** | 🟠 Medium |
| **Depends On** | TASK-CORE-005, TASK-CORE-009 |
| **Estimated** | 3h |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** MCP server.go: ToolRegistry exists; admin tools (tenant/user CRUD) not implemented
---

## Mục tiêu

Register 6 Admin Operation MCP tools: health, metrics, forget, api keys.

---

## Công việc cụ thể

### `gateway/adapter/mcp/handlers_admin.go` [NEW]

```go
// admin_get_health — aggregated health check
func (h *Handlers) handleAdminHealth(ctx context.Context, params map[string]any) (any, error) {
    // Call GET :8083/healthz internally
    resp, err := http.Get("http://localhost:8083/healthz")
    if err != nil { return nil, err }
    defer resp.Body.Close()
    var health map[string]any
    json.NewDecoder(resp.Body).Decode(&health)
    return health, nil
}

// admin_get_metrics — return key Prometheus metrics as structured data
func (h *Handlers) handleAdminMetrics(ctx context.Context, params map[string]any) (any, error) { ... }

// admin_forget_user — GDPR forget (admin only)
func (h *Handlers) handleAdminForget(ctx context.Context, params map[string]any) (any, error) {
    if !hasAdminRole(ctx) { return nil, errors.New("admin role required") }
    // Call forget usecase
    conn := h.registry.Get("vnp-admin")
    client := adminpb.NewAdminServiceClient(conn)
    resp, err := client.ForgetUser(ctx, &adminpb.ForgetRequest{
        UserId: params["user_id"].(string),
        Reason: params["reason"].(string),
        TenantId: tenant.FromContext(ctx),
    })
    return resp, err
}

// admin_list_api_keys
func (h *Handlers) handleListAPIKeys(ctx context.Context, params map[string]any) (any, error) { ... }

// admin_revoke_key
func (h *Handlers) handleRevokeKey(ctx context.Context, params map[string]any) (any, error) { ... }

// admin_get_agent_context — debug endpoint
func (h *Handlers) handleGetAgentContext(ctx context.Context, params map[string]any) (any, error) { ... }
```

### Register in `server.go`

```go
s.Register("admin_get_health",    h.handleAdminHealth,  mcp.Schema{})
s.Register("admin_get_metrics",   h.handleAdminMetrics, mcp.Schema{"service?": "string"})
s.Register("admin_forget_user",   h.handleAdminForget,  mcp.Schema{"user_id": "string", "reason": "string"})
s.Register("admin_list_api_keys", h.handleListAPIKeys,  mcp.Schema{})
s.Register("admin_revoke_key",    h.handleRevokeKey,    mcp.Schema{"key_id": "string"})
s.Register("admin_get_agent_context", h.handleGetAgentContext, mcp.Schema{"user_id": "string"})
```

---

## Acceptance Criteria

- [ ] 6 admin tools registered
- [ ] Total MCP tools after TASK-009+010+011: ≥ 37 tools
- [ ] `admin_forget_user` → 403 without admin role
- [ ] `admin_get_health` returns aggregated status

## Files

```
gateway/adapter/mcp/handlers_admin.go  [NEW]
gateway/adapter/mcp/server.go          [MODIFY — register 6 tools, total 37+]
```
