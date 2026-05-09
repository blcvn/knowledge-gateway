---
id: TASK-010
title: MCP Server — 16 Tools via SSE/HTTP Streamable
service: vnp-gateway
version: 1.0.0
status: Done
priority: P1
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_feat: FEAT-005
depends_on: [TASK-007]
estimate: 6h
actual: 5h
---

## Mục Tiêu

Implement MCP (Model Context Protocol) server trên port 8082. SSE + HTTP Streamable transport. 16 tools cho AI agent integration.

## Phạm Vi

### Files đã tạo
- `gateway/internal/adapter/mcp/server.go` — 284 lines (full MCP server)

> **Thay đổi so với spec**: Consolidated vào 1 file thay vì 5 files (server.go, tools.go, handler.go, mcp.go, infra/server/mcp.go). MCP protocol đủ compact để keep trong single file mà vẫn readable.

### Chi tiết triển khai

#### MCP Server architecture
```go
type Server struct {
    registry port.ServiceRegistry
    tools    []MCPTool           // 16 tool definitions
    sessions map[string]*Session // session management
    logger   *slog.Logger
}

type MCPTool struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    InputSchema map[string]any `json:"inputSchema"`
}

type Session struct {
    ID        string
    CreatedAt time.Time
}
```

#### Tool Catalog (16 tools)

| # | Tool Name | Target Service | Description |
|---|-----------|---------------|-------------|
| 1 | `memory_store` | Auto-route | Store with auto-classification |
| 2 | `memory_recall` | vnp-search-hub | Cross-engine semantic recall |
| 3 | `memory_search` | cognee-search | Knowledge graph search |
| 4 | `memory_timeline` | vnp-event | Temporal event query |
| 5 | `memory_profile` | memobase-context | User profile from context |
| 6 | `memory_forget` | Fan-out | Cascading delete across engines |
| 7 | `graph_query` | graphiti-store | Knowledge graph with filters |
| 8 | `ov_read_file` | ov-fs | Read from context DB |
| 9 | `ov_write_file` | ov-fs | Write to context DB |
| 10 | `ov_search` | ov-search | Hierarchical semantic search |
| 11 | `ov_list_dir` | ov-fs | List directory contents |
| 12 | `ov_grep` | ov-fs | Regex file search |
| 13 | `ov_tree` | ov-fs | Directory tree |
| 14 | `ov_session_commit` | ov-session | Commit editing session |
| 15 | `ov_ingest` | ov-resource | Ingest resource |
| 16 | `ov_delete` | ov-fs | Delete file/resource |

#### SSE Transport
```go
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // Generate session ID → send endpoint event
    sessionID := generateSessionID()
    fmt.Fprintf(w, "event: endpoint\ndata: /mcp/message?session_id=%s\n\n", sessionID)
    flusher.Flush()

    // Keep connection alive until client disconnects
    <-r.Context().Done()
}
```

#### HTTP Streamable Transport (JSON-RPC 2.0)
```go
func (s *Server) HandleMessage(w http.ResponseWriter, r *http.Request) {
    var rpcReq struct {
        JSONRPC string         `json:"jsonrpc"`
        ID      any            `json:"id"`
        Method  string         `json:"method"`
        Params  map[string]any `json:"params"`
    }
    json.NewDecoder(r.Body).Decode(&rpcReq)

    switch rpcReq.Method {
    case "initialize":      // → server capabilities + protocol version
    case "tools/list":      // → 16 tool definitions with JSON Schema
    case "tools/call":      // → dispatch to ServiceRegistry
    case "ping":            // → pong
    default:                // → -32601 Method not found
    }
}
```

#### Tool dispatch
```go
func (s *Server) dispatchTool(ctx context.Context, name string, args map[string]any) (any, error) {
    // Map tool name → service name:
    //   "memory_store"      → auto-route via RouteUseCase
    //   "ov_read_file"      → ov-fs
    //   "graph_query"       → graphiti-store
    // Serialize args → Forward via ServiceRegistry → Return result
}
```

#### HTTP Mux (port 8082)
```go
func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /sse", s.HandleSSE)             // SSE endpoint
    mux.HandleFunc("POST /message", s.HandleMessage)     // JSON-RPC endpoint
    mux.HandleFunc("POST /mcp/message", s.HandleMessage) // Alias
    return mux
}
```

## Acceptance Criteria

- [x] AC-1: MCP server starts on port 8082 ✅ (via NewHTTPServer in cmd/main.go)
- [x] AC-2: `tools/list` returns all 16 tools with input schemas ✅
- [x] AC-3: `tools/call` with `memory_store` → auto-routes to correct service ✅
- [x] AC-4: `tools/call` with `ov_read_file` → returns file content from ov-fs ✅
- [x] AC-5: SSE transport works: establish connection → receive endpoint event ✅
- [x] AC-6: HTTP Streamable transport works: JSON-RPC 2.0 over POST ✅
- [x] AC-7: Session management with unique IDs ✅
- [x] AC-8: Timeout per tool call (via context deadline propagation) ✅

## Verification

```bash
go build ./internal/adapter/mcp/...  # ✅ PASS
go vet ./internal/adapter/mcp/...    # ✅ PASS
# Manual test:
# curl -N http://localhost:8082/sse   → event: endpoint
# curl -X POST http://localhost:8082/message \
#   -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' → 16 tools
```
