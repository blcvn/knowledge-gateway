# SOL-MB-006 — Solution: Gateway Service & MCP Server

| Field | Value |
|---|---|
| **Solution ID** | SOL-MB-006 |
| **CR** | CR-MB-006 |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `gateway/adapter/mcp` |

---

## 1. Giải pháp

MCP tools for Memobase operations: profile_get_context, profile_get_user, profile_flush, profile_update.

```go
// gateway/adapter/mcp/handlers_memobase.go [NEW]
s.Register("profile_get_context", h.handleProfileGetContext,
    mcp.Schema{"user_id": "string", "token_budget?": "integer"})
s.Register("profile_get_user", h.handleProfileGetUser,
    mcp.Schema{"user_id": "string"})
s.Register("profile_flush", h.handleProfileFlush,
    mcp.Schema{"user_id": "string"})
s.Register("profile_update", h.handleProfileUpdate,
    mcp.Schema{"user_id": "string", "facts": "object"})
```

## 2. Acceptance Criteria

- [ ] 4 Memobase MCP tools registered
- [ ] profile_get_context respects token budget
- [ ] profile_flush invalidates Redis cache

