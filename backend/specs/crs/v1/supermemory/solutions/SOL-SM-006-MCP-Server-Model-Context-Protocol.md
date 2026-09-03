# SOL-SM-006 — Solution: MCP Server (Model Context Protocol)

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-006 |
| **CR** | CR-SM-006 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `gateway/adapter/mcp` |

---

## 1. Giải pháp

5 Supermemory MCP tools via gateway.

```go
// gateway/adapter/mcp/handlers_supermemory.go [NEW]
s.Register("sm_ingest", h.handleSMIngest,
    mcp.Schema{"content": "string", "space?": "string", "metadata?": "object"})
s.Register("sm_search", h.handleSMSearch,
    mcp.Schema{"query": "string", "space?": "string", "limit?": "integer"})
s.Register("sm_rag", h.handleSMRAG,
    mcp.Schema{"question": "string", "space?": "string"})
s.Register("sm_get_profile", h.handleSMGetProfile,
    mcp.Schema{"user_id": "string"})
s.Register("sm_forget", h.handleSMForget,
    mcp.Schema{"memory_id": "string"})
```

## 2. Acceptance Criteria

- [ ] 5 SM MCP tools registered
- [ ] sm_rag returns cited answer
- [ ] Memory spaces supported in all tools

