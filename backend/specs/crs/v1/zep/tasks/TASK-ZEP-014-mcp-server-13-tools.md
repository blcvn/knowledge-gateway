# TASK-ZEP-014 — services/zep-mcp: MCP Server (13 Read-Only Tools)

**Task ID:** TASK-ZEP-014  
**Wave:** 5 (Integration)  
**Solution:** [SOL-ZEP-005](../solutions/SOL-ZEP-005-MCP-Server-13-Tools.md)  
**Depends on:** TASK-ZEP-009 (GetUserContext), TASK-ZEP-013 (search_graph)  
**Ước tính:** 4h  
**Priority:** High — AI agent access

**Trạng thái:** 🔄 Partial  
**Ghi chú:** zep-go: 0 .go - MCP server scaffold missing; gateway MCP partial  
---

## Mục tiêu

Tạo `services/zep-mcp/` — standalone binary MCP server với 13 read-only tools:
- Dual transport: `--stdio` (Claude Desktop/Cline) và `--port N` (HTTP Streamable 2025-03-26)
- Docker distroless image
- Claude Desktop JSON config template

---

## Công việc cụ thể

### 1. Tạo MCP Server binary

**`services/zep-mcp/cmd/server/main.go`**

```go
// Flags:
//   --stdio       : run in stdio mode (Claude Desktop, Cline)
//   --port N      : run HTTP Streamable mode (default 8080)
//   --log-level   : debug|info|warn|error (default info)

// Env vars required:
//   VNP_MEMORY_API_KEY  : API key (starts with "vnp_")
//   VNP_MEMORY_API_URL  : base URL (default "https://api.vnpmemory.io")
```

### 2. Implement 13 Tools (grouped)

**Group A — Core Search (4 tools):**

| Tool | Required Params | Optional Params |
|------|----------------|-----------------|
| `search_graph` | user_id, query | scope, limit, reranker, node_labels, edge_types, min_fact_rating, mmr_lambda, center_node_uuid |
| `get_user_context` | thread_id | template_id |
| `get_user` | user_id | — |
| `list_threads` | user_id | — |

**Group B — Graph Query (3 tools):**

| Tool | Required Params | Optional Params |
|------|----------------|-----------------|
| `get_user_nodes` | user_id | limit (default 50) |
| `get_user_edges` | user_id | limit (default 50) |
| `get_episodes` | user_id | lastn (default 10) |

**Group C — Detail Retrieval (6 tools):**

| Tool | Required Params |
|------|----------------|
| `get_thread_messages` | thread_id |
| `get_node` | node_uuid |
| `get_edge` | edge_uuid |
| `get_episode` | episode_uuid |
| `get_node_edges` | node_uuid |
| `get_episode_mentions` | episode_uuid |

### 3. Tool Implementation Pattern

```go
// services/zep-mcp/internal/tools/search_graph.go

var SearchGraphTool = mcp.Tool{
    Name:        "search_graph",
    Description: "Search the temporal knowledge graph for facts, entities, and events. " +
                 "Returns results with temporal annotations (valid_at, invalid_at) showing when facts were true. " +
                 "Use scope='edges' for facts, 'nodes' for entities, 'episodes' for events.",
    InputSchema: mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]any{
            "user_id": map[string]any{"type": "string", "description": "Required. User ID to search"},
            "query":   map[string]any{"type": "string", "description": "Required. Natural language query"},
            "scope":   map[string]any{"type": "string", "enum": []string{"edges","nodes","episodes","all"}, "default": "edges"},
            "limit":   map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "default": 10},
            "reranker":         map[string]any{"type": "string", "enum": []string{"rrf","mmr","cross_encoder","node_distance","episode_mentions"}, "default": "rrf"},
            "node_labels":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
            "edge_types":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
            "min_fact_rating":  map[string]any{"type": "number", "minimum": 0, "maximum": 1},
            "mmr_lambda":       map[string]any{"type": "number", "minimum": 0, "maximum": 1},
            "center_node_uuid": map[string]any{"type": "string"},
        },
        Required: []string{"user_id", "query"},
    },
}

func handleSearchGraph(client VNPMemoryClient) mcp.HandlerFunc {
    return func(ctx context.Context, args map[string]any) (string, error) {
        // parse args, call client.SearchGraph(), format JSON response
    }
}
```

### 4. Tạo VNPMemoryClient (HTTP Client)

```go
// services/zep-mcp/internal/client.go
// HTTP client gọi VNP Memory REST API

type VNPMemoryClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

// Methods (all read-only):
func (c *VNPMemoryClient) SearchGraph(ctx context.Context, req SearchGraphRequest) (*SearchGraphResponse, error)
func (c *VNPMemoryClient) GetUserContext(ctx context.Context, threadID string) (*UserContextResponse, error)
func (c *VNPMemoryClient) GetUser(ctx context.Context, userID string) (*UserResponse, error)
func (c *VNPMemoryClient) ListThreads(ctx context.Context, userID string) (*ListThreadsResponse, error)
func (c *VNPMemoryClient) GetUserNodes(ctx context.Context, userID string, limit int) (*NodesResponse, error)
func (c *VNPMemoryClient) GetUserEdges(ctx context.Context, userID string, limit int) (*EdgesResponse, error)
// ...etc for all 13 tools
```

### 5. Tạo Dockerfile

```dockerfile
# services/zep-mcp/Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o zep-mcp-server ./services/zep-mcp/cmd/server

# Distroless: minimal image, no shell, no package manager
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/zep-mcp-server /zep-mcp-server
ENTRYPOINT ["/zep-mcp-server"]
CMD ["--port", "8080"]
```

### 6. Tạo Claude Desktop Config Template

**`services/zep-mcp/docs/claude_desktop_config.json`**
```json
{
  "mcpServers": {
    "vnp-memory": {
      "command": "/path/to/zep-mcp-server",
      "args": ["--stdio"],
      "env": {
        "VNP_MEMORY_API_KEY": "vnp_your_api_key_here",
        "VNP_MEMORY_API_URL": "https://api.vnpmemory.io"
      }
    }
  }
}
```

### 7. Tests

- `TestServer_HasExactly13Tools`: MCP server lists exactly 13 tools
- `TestSearchGraph_RequiredParams`: missing user_id → error response
- `TestSearchGraph_DefaultScope`: no scope → defaults to "edges"
- `TestGetUserContext_Returns200`: mock client → tool returns formatted string
- `TestNoWriteTools`: verify tools list contains NO create/update/delete tools
- `TestStdioMode_Compiles`: `go build` with --stdio flag

---

## Acceptance Criteria

- [ ] `go build ./services/zep-mcp/...` không có lỗi
- [ ] MCP server lists chính xác 13 tools
- [ ] `search_graph` có đầy đủ 9 input parameters
- [ ] Không có tool nào là write operation (create/update/delete)
- [ ] `--stdio` flag → starts without error
- [ ] `--port 8080` flag → HTTP server starts without error
- [ ] Docker build thành công, image < 20MB
- [ ] `go test ./services/zep-mcp/...` pass

---

## Files tạo ra

```
services/zep-mcp/
├── cmd/server/
│   └── main.go
├── internal/
│   ├── server.go
│   ├── client.go
│   ├── auth.go
│   └── tools/
│       ├── search_graph.go
│       ├── get_user_context.go
│       ├── get_user.go
│       ├── list_threads.go
│       ├── get_user_nodes.go
│       ├── get_user_edges.go
│       ├── get_episodes.go
│       ├── get_thread_messages.go
│       ├── get_node.go
│       ├── get_edge.go
│       ├── get_episode.go
│       ├── get_node_edges.go
│       ├── get_episode_mentions.go
│       └── tools_test.go
├── docs/
│   └── claude_desktop_config.json
└── Dockerfile
```

## Sau khi hoàn thành

Chạy: `go build ./services/zep-mcp/... && go test ./services/zep-mcp/...`
Verify: `./zep-mcp-server --help` shows --stdio and --port flags
