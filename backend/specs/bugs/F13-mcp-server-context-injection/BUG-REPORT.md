# Bug Report — F13: MCP Server & Context Injection

> Feature: MCP protocol server cho AI Agent tools
> Luồng: `apps/memory → gateway MCP port → mcp.Server → services`

---

## BUG-F13-001: MCP Server Handler Thiếu Implementation

**Severity:** HIGH  
**File:** `gateway/adapter/mcp/`

**Mô tả:**  
MCP server được khởi tạo (`mcp.NewServer(registry, logger)`) và expose trên MCP port, nhưng cần verify `adapter/mcp/` directory có đầy đủ tool handlers.

**Kiểm tra:** Cần xem `gateway/adapter/mcp/` để xác định tools nào đã được implement.

---

## BUG-F13-002: MCP Server Không Được Wired Với Auth Context

**Severity:** HIGH  
**File:** `gateway/cmd/main.go:260, 285-292`

**Mô tả:**  
MCP server chạy trên port riêng (`cfg.Server.MCPPort`) với HTTP server riêng. Không có Auth middleware được apply cho MCP endpoints.

```go
mcpHTTPSrv := server.NewHTTPServer(mcpSrv.Handler(), cfg.Server.MCPPort, logger)
```

**Impact:**  
- MCP endpoints không require authentication.
- Any agent có thể call MCP tools không cần API key hoặc JWT.
- Tenant isolation bị phá vỡ trong MCP context.
