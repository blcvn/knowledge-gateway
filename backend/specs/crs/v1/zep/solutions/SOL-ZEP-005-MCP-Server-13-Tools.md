# Solution: SOL-ZEP-005 — MCP Server (13 Read-Only Tools)

**CR ID:** CR-ZEP-005  
**Solution ID:** SOL-ZEP-005  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp MCP Server hiện có (port `:8082`) để thêm **9 tools mới** (từ 4 tools hiện có lên 13 tools đầy đủ theo Zep spec). Tất cả tools read-only. Bổ sung dual transport: stdio (Claude Desktop/Cline) + HTTP Streamable 2025-03-26. Đóng gói riêng thành standalone binary `zep-mcp-server` cho Docker deployment.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| MCP Server `:8082` | `gateway/adapter/mcp/` | Có: SSE + HTTP Streamable |
| `search_graph` tool | | Thiếu: scope, reranker, min_fact_rating params |
| `get_user_context` | | Thiếu |
| `get_user_nodes/edges` | | Thiếu |
| `get_episodes` | | Thiếu |
| `get_thread_messages` | | Có: basic |
| `get_node/edge/episode` | | Thiếu |
| `get_node_edges` | | Thiếu |
| `get_episode_mentions` | | Thiếu |

### Gap phân tích

Cần thêm 9 tools mới + upgrade `search_graph` với đầy đủ parameters.

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc (Tách riêng binary)

```
services/zep-mcp/              # Standalone MCP server
├── cmd/server/main.go         # Dual transport: --stdio | --port N
├── internal/
│   ├── server.go              # MCP server setup
│   ├── auth.go                # API key validation
│   └── tools/
│       ├── search_graph.go          # search_graph (upgrade)
│       ├── get_user_context.go      # get_user_context
│       ├── get_user.go              # get_user
│       ├── list_threads.go          # list_threads
│       ├── get_user_nodes.go        # get_user_nodes
│       ├── get_user_edges.go        # get_user_edges
│       ├── get_episodes.go          # get_episodes
│       ├── get_thread_messages.go   # get_thread_messages
│       ├── get_node.go              # get_node
│       ├── get_edge.go              # get_edge
│       ├── get_episode.go           # get_episode
│       ├── get_node_edges.go        # get_node_edges
│       └── get_episode_mentions.go  # get_episode_mentions
└── Dockerfile
```

### 3.2. 13 Tools Implementation

#### Group 1: Core Search (4 tools)

```go
// tools/search_graph.go — Flagship tool (UPGRADE)

var SearchGraphTool = mcp.Tool{
    Name: "search_graph",
    Description: "Search the temporal knowledge graph for facts, entities, and events. " +
        "Returns results with temporal annotations (valid_at, invalid_at) showing when facts were true.",
    InputSchema: mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]any{
            "user_id":          {"type": "string", "description": "Required. User ID to search graph for"},
            "query":            {"type": "string", "description": "Required. Natural language search query"},
            "scope":            {"type": "string", "enum": []string{"edges","nodes","episodes","all"}, "default": "edges"},
            "limit":            {"type": "integer", "minimum": 1, "maximum": 50, "default": 10},
            "reranker":         {"type": "string", "enum": []string{"rrf","mmr","cross_encoder","node_distance","episode_mentions"}, "default": "rrf"},
            "node_labels":      {"type": "array", "items": {"type": "string"}, "description": "Filter by node types: User, Preference, Organization..."},
            "edge_types":       {"type": "array", "items": {"type": "string"}, "description": "Filter by relationship types: WORKS_AT, LIVES_IN..."},
            "min_fact_rating":  {"type": "number", "minimum": 0, "maximum": 1, "description": "Minimum fact quality threshold"},
            "mmr_lambda":       {"type": "number", "minimum": 0, "maximum": 1, "description": "MMR: 0=diversity, 1=relevance (default 0.5)"},
            "center_node_uuid": {"type": "string", "description": "For node_distance: reference node UUID"},
        },
        Required: []string{"user_id", "query"},
    },
}

func handleSearchGraph(ctx context.Context, apiClient VNPMemoryClient, args map[string]any) (string, error) {
    resp, err := apiClient.SearchGraph(ctx, &SearchGraphRequest{
        UserID:         args["user_id"].(string),
        Query:          args["query"].(string),
        Scope:          stringOrDefault(args, "scope", "edges"),
        Limit:          intOrDefault(args, "limit", 10),
        Reranker:       stringOrDefault(args, "reranker", "rrf"),
        NodeLabels:     stringSliceOrNil(args, "node_labels"),
        EdgeTypes:      stringSliceOrNil(args, "edge_types"),
        MinFactRating:  float64OrNil(args, "min_fact_rating"),
        MmrLambda:      float64OrNil(args, "mmr_lambda"),
        CenterNodeUUID: stringOrNil(args, "center_node_uuid"),
    })
    if err != nil { return "", err }
    return formatJSON(resp), nil
}

// tools/get_user_context.go
var GetUserContextTool = mcp.Tool{
    Name:        "get_user_context",
    Description: "Get pre-formatted context about a user, ready for injection into LLM system prompt.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{
            "thread_id":   {"type": "string"},
            "template_id": {"type": "string", "description": "Optional: custom format template"},
        },
        Required: []string{"thread_id"},
    },
}

// tools/get_user.go
var GetUserTool = mcp.Tool{
    Name:        "get_user",
    Description: "Get user metadata and profile information.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{"user_id": {"type": "string"}},
        Required:   []string{"user_id"},
    },
}

// tools/list_threads.go
var ListThreadsTool = mcp.Tool{
    Name:        "list_threads",
    Description: "List all conversation threads for a user.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{"user_id": {"type": "string"}},
        Required:   []string{"user_id"},
    },
}
```

#### Group 2: Graph Query (3 tools)

```go
// tools/get_user_nodes.go
var GetUserNodesTool = mcp.Tool{
    Name: "get_user_nodes",
    Description: "Get all entity nodes in a user's knowledge graph. " +
        "Returns classified entities: User, Preference, Organization, Event, Location, etc.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{
            "user_id": {"type": "string"},
            "limit":   {"type": "integer", "default": 50},
        },
        Required: []string{"user_id"},
    },
}

// tools/get_user_edges.go
var GetUserEdgesTool = mcp.Tool{
    Name: "get_user_edges",
    Description: "Get all temporal edges (facts) in a user's knowledge graph. " +
        "Each edge includes valid_at/invalid_at timestamps for temporal reasoning.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{
            "user_id": {"type": "string"},
            "limit":   {"type": "integer", "default": 50},
        },
        Required: []string{"user_id"},
    },
}

// tools/get_episodes.go
var GetEpisodesTool = mcp.Tool{
    Name:        "get_episodes",
    Description: "Get recent episodes (temporal events) from a user's knowledge graph.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{
            "user_id": {"type": "string"},
            "lastn":   {"type": "integer", "default": 10},
        },
        Required: []string{"user_id"},
    },
}
```

#### Group 3: Detail Retrieval (6 tools)

```go
// tools/get_thread_messages.go
var GetThreadMessagesTool = mcp.Tool{
    Name:        "get_thread_messages",
    Description: "Get messages from a specific conversation thread.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{
            "thread_id": {"type": "string"},
            "limit":     {"type": "integer", "default": 50},
        },
        Required: []string{"thread_id"},
    },
}

// tools/get_node.go
var GetNodeTool = mcp.Tool{
    Name:        "get_node",
    Description: "Get detailed information about a specific entity node in the knowledge graph.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{"node_uuid": {"type": "string"}},
        Required:   []string{"node_uuid"},
    },
}

// tools/get_edge.go
var GetEdgeTool = mcp.Tool{
    Name:        "get_edge",
    Description: "Get detailed information about a specific temporal edge (fact) including valid_at/invalid_at.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{"edge_uuid": {"type": "string"}},
        Required:   []string{"edge_uuid"},
    },
}

// tools/get_episode.go
var GetEpisodeTool = mcp.Tool{
    Name:        "get_episode",
    Description: "Get detailed information about a specific episode.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{"episode_uuid": {"type": "string"}},
        Required:   []string{"episode_uuid"},
    },
}

// tools/get_node_edges.go
var GetNodeEdgesTool = mcp.Tool{
    Name:        "get_node_edges",
    Description: "Get all edges (relationships) connected to a specific entity node.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{"node_uuid": {"type": "string"}},
        Required:   []string{"node_uuid"},
    },
}

// tools/get_episode_mentions.go
var GetEpisodeMentionsTool = mcp.Tool{
    Name:        "get_episode_mentions",
    Description: "Get all nodes and edges that are mentioned/referenced in a specific episode.",
    InputSchema: mcp.ToolInputSchema{
        Properties: map[string]any{"episode_uuid": {"type": "string"}},
        Required:   []string{"episode_uuid"},
    },
}
```

### 3.3. Server Setup (Dual Transport)

```go
// services/zep-mcp/cmd/server/main.go

func main() {
    var stdioMode bool
    var port int
    flag.BoolVar(&stdioMode, "stdio", false, "Run in stdio mode (Claude Desktop, Cline)")
    flag.IntVar(&port, "port", 8080, "HTTP port for Streamable HTTP mode")
    flag.Parse()

    apiKey := os.Getenv("VNP_MEMORY_API_KEY")
    apiURL := os.Getenv("VNP_MEMORY_API_URL") // default: "https://api.vnpmemory.io"

    client := NewVNPMemoryClient(apiURL, apiKey)
    server := NewMCPServer(client)

    if stdioMode {
        // stdio transport — Claude Desktop, Cline
        if err := server.RunStdio(); err != nil {
            log.Fatal(err)
        }
    } else {
        // HTTP Streamable — Claude Code (spec 2025-03-26)
        if err := server.RunHTTP(fmt.Sprintf(":%d", port)); err != nil {
            log.Fatal(err)
        }
    }
}

// services/zep-mcp/internal/server.go
func NewMCPServer(client VNPMemoryClient) *MCPServer {
    s := mcp.NewServer("vnp-memory", "1.0.0")

    // Register all 13 read-only tools
    s.AddTool(SearchGraphTool, handleSearchGraph(client))
    s.AddTool(GetUserContextTool, handleGetUserContext(client))
    s.AddTool(GetUserTool, handleGetUser(client))
    s.AddTool(ListThreadsTool, handleListThreads(client))
    s.AddTool(GetUserNodesTool, handleGetUserNodes(client))
    s.AddTool(GetUserEdgesTool, handleGetUserEdges(client))
    s.AddTool(GetEpisodesTool, handleGetEpisodes(client))
    s.AddTool(GetThreadMessagesTool, handleGetThreadMessages(client))
    s.AddTool(GetNodeTool, handleGetNode(client))
    s.AddTool(GetEdgeTool, handleGetEdge(client))
    s.AddTool(GetEpisodeTool, handleGetEpisode(client))
    s.AddTool(GetNodeEdgesTool, handleGetNodeEdges(client))
    s.AddTool(GetEpisodeMentionsTool, handleGetEpisodeMentions(client))

    // NO write tools — read-only by design
    return &MCPServer{server: s, client: client}
}
```

### 3.4. Docker Deployment

```dockerfile
# services/zep-mcp/Dockerfile

FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o zep-mcp-server ./services/zep-mcp/cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/zep-mcp-server /zep-mcp-server
ENTRYPOINT ["/zep-mcp-server"]
CMD ["--port", "8080"]
```

```yaml
# docker-compose.yml
services:
  vnp-memory-mcp:
    image: vnp-memory/mcp-server:latest
    ports: ["8080:8080"]
    environment:
      - VNP_MEMORY_API_KEY=${VNP_MEMORY_API_KEY}
      - VNP_MEMORY_API_URL=https://api.vnpmemory.io
      - LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "/zep-mcp-server", "--health-check"]
      interval: 30s
      timeout: 5s
      retries: 3
```

### 3.5. Claude Desktop Configuration

```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
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

```bash
# Claude Code (HTTP mode)
claude mcp add --transport http vnp-memory http://localhost:8080
```

---

## 4. Tools Summary Table

| # | Tool | Group | Parameters |
|---|------|-------|------------|
| 1 | `search_graph` | Core Search | user_id, query, scope, reranker, node_labels, edge_types, min_fact_rating, mmr_lambda, center_node_uuid |
| 2 | `get_user_context` | Core Search | thread_id, template_id? |
| 3 | `get_user` | Core Search | user_id |
| 4 | `list_threads` | Core Search | user_id |
| 5 | `get_user_nodes` | Graph Query | user_id, limit? |
| 6 | `get_user_edges` | Graph Query | user_id, limit? |
| 7 | `get_episodes` | Graph Query | user_id, lastn? |
| 8 | `get_thread_messages` | Detail | thread_id, limit? |
| 9 | `get_node` | Detail | node_uuid |
| 10 | `get_edge` | Detail | edge_uuid |
| 11 | `get_episode` | Detail | episode_uuid |
| 12 | `get_node_edges` | Detail | node_uuid |
| 13 | `get_episode_mentions` | Detail | episode_uuid |

---

## 5. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | MCP server skeleton + dual transport (stdio/HTTP) | 1 ngày |
| **P2** | search_graph upgrade (full params) | 1 ngày |
| **P3** | get_user_context + get_user + list_threads | 1 ngày |
| **P4** | get_user_nodes + get_user_edges + get_episodes | 1 ngày |
| **P5** | get_thread_messages + get_node + get_edge + get_episode | 1 ngày |
| **P6** | get_node_edges + get_episode_mentions | 1 ngày |
| **P7** | Dockerfile + health check | 0.5 ngày |
| **P8** | Integration tests (Claude Desktop, Claude Code) | 1 ngày |

**Tổng:** ~7.5 ngày (Wave 5)

---

## 6. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| Claude Desktop thấy đủ 13 tools | s.AddTool × 13, --stdio transport |
| Claude Code search_graph → temporal facts với valid_at/invalid_at | search_graph → SearchService → TemporalEdgeSummary |
| get_user_nodes → classified entities | Node.NodeType in response JSON |
| get_episode_mentions → nodes + edges | Neo4j query: episode → mentions → nodes + edges |
| 0 write tools | Chỉ AddTool không có write handlers |
| get_user_context → formatted string | Memory Service GetUserContext endpoint |
| Docker start < 5s | distroless image, no runtime deps |
