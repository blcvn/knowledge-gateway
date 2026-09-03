# ADR-009 — MCP Server: SSE + HTTP Streamable (Dual Transport)

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-04 |
| **Deciders** | Integration Team |
| **Feature** | F13 (MCP Server & Context Injection) |

---

## Context

Model Context Protocol (MCP) hỗ trợ 2 transport modes:
1. **SSE (Server-Sent Events):** GET /mcp/sse → stream JSON-RPC events
2. **HTTP Streamable:** POST /mcp/message → request-response

Claude Code và nhiều AI frameworks cần MCP. Claude Code hỗ trợ SSE; mới hơn có thể dùng HTTP Streamable.

---

## Decision

**Hỗ trợ cả 2 transport modes đồng thời.**

```go
// SSE transport (legacy, Claude Code support)
mux.HandleFunc("GET /mcp/sse", mcpHandler.HandleSSE)

// HTTP Streamable (new MCP spec)
mux.HandleFunc("POST /mcp/message", mcpHandler.HandleHTTP)

// 37+ tools exposed via both transports:
// memory_store, memory_recall, memory_forget
// ov_grep, ov_read_file, ov_write_file, ov_search
// observe_session_start, observe_hook
// ... 29 more
```

**Agent scope cho context filtering:**

```json
{
  "name": "memory_recall",
  "arguments": {
    "query": "auth middleware",
    "scope": "project",        // project / session / global
    "agent_id": "claude-code"
  }
}
```

---

## Consequences

**Positive:**
- Claude Code, Cursor, GitHub Copilot tất cả đều work ngay
- Không cần client upgrade khi MCP spec thay đổi
- 37+ tools = full memory operations từ IDE

**Negative:**
- 2 transport implementations cần maintain
- SSE có limitation: server push only (không bidirectional)
- HTTP Streamable mới hơn = ít client support

---

## Alternatives Considered

### A1 — Chỉ SSE
- **Rejected:** Claude Code dùng SSE nhưng newer clients cần HTTP Streamable; limit future compatibility

### A2 — Chỉ HTTP Streamable
- **Rejected:** Claude Code hiện tại dùng SSE; break existing integrations
