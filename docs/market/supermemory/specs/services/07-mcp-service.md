# 07 — MCP Service

> **gRPC**: 9006 | **Health**: 9086 | **SSE**: via Gateway :8080/mcp

---

## 1. Purpose

Model Context Protocol (MCP) server: expose Supermemory capabilities (memory, recall, graph, projects) cho MCP-compatible AI clients (Claude, Cursor, Windsurf, VS Code, etc.) qua JSON-RPC over SSE.

---

## 2. Clean Architecture

```
services/mcp-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # MCPSession, MCPTool, MCPResource, MCPPrompt
│   │   ├── value_object.go     # ToolName, ResourceURI, SessionState
│   │   └── errors.go           # ErrSessionNotFound, ErrToolNotFound
│   ├── usecase/
│   │   ├── handle_tool.go      # Tool dispatch: memory, recall, listProjects, whoAmI
│   │   ├── handle_resource.go  # Resource: supermemory://profile, projects
│   │   ├── handle_prompt.go    # Prompt: context injection
│   │   ├── manage_session.go   # Session lifecycle (create, cache, invalidate)
│   │   ├── port/
│   │   │   ├── input.go        # HandleToolUC, HandleResourceUC, HandlePromptUC
│   │   │   └── output.go       # MemoryClient, SearchClient, ProfileClient,
│   │   │                       # ProjectClient, SessionStore, EventPublisher
│   │   └── dto/
│   │       └── mcp.go          # ToolInput, ToolOutput, ResourceOutput
│   ├── adapter/
│   │   ├── jsonrpc/
│   │   │   ├── server.go       # JSON-RPC 2.0 handler
│   │   │   ├── sse.go          # Server-Sent Events transport
│   │   │   └── protocol.go    # MCP protocol message types
│   │   ├── grpc/handler.go     # MCPServiceServer (for Gateway proxy)
│   │   ├── grpc_client/
│   │   │   ├── document.go     # Document Service client (createMemory)
│   │   │   ├── memory.go       # Memory Service client (forgetMemory)
│   │   │   ├── search.go       # Search Service client (recall)
│   │   │   ├── profile.go      # Profile Service client (getProfile)
│   │   │   └── project.go      # Project Service client (listProjects)
│   │   ├── session/
│   │   │   └── redis.go        # Session state (containerTags cache, clientInfo)
│   │   ├── event/
│   │   │   └── publisher.go    # NATS: analytics events
│   │   └── auth/
│   │       └── validator.go    # API Key (sm_) / OAuth token validation
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
└── Dockerfile
```

---

## 3. MCP Protocol Registration

| Type | Name | Description |
|------|------|-------------|
| **Tool** | `memory` | Save or forget information (max 200K chars) |
| **Tool** | `recall` | Search memories + get user profile |
| **Tool** | `listProjects` | List available projects/container tags |
| **Tool** | `whoAmI` | Get authenticated user info |
| **Resource** | `supermemory://profile` | User profile data |
| **Resource** | `supermemory://projects` | Projects list |
| **Prompt** | `context` | System context injection with user profile |

---

## 4. Session Management

```go
type MCPSession struct {
    SessionID           string
    UserID              string
    OrgID               string
    APIKey              string
    Email               *string
    Name                *string
    RootContainerTag    *string         // From x-sm-project header
    CachedContainerTags []string        // TTL 5min
    TagsLastFetched     *time.Time
    ClientInfo          *MCPClientInfo  // Client name/version
    CreatedAt           time.Time
}

type MCPClientInfo struct {
    Name    string  // "claude-desktop", "cursor", "vscode"
    Version string
}

// Session keyed by user_id in Redis (persistent across reconnects)
// Container tag cache refreshed every 5min or on force-refresh
```

---

## 5. Tool: `memory` (Save/Forget)

```go
func (h *ToolHandler) HandleMemory(ctx context.Context, session *MCPSession, args MemoryArgs) *ToolResult {
    containerTag := resolveContainerTag(args.ContainerTag, session.RootContainerTag)

    switch args.Action {
    case "save":
        result, err := h.documentClient.CreateDocument(ctx, &CreateDocumentRequest{
            Content:       args.Content,       // max 200K chars
            ContainerTags: []string{containerTag},
            Metadata:      map[string]string{"sm_source": "mcp"},
        })
        // Force-refresh container tags if new tag
        if !contains(session.CachedContainerTags, containerTag) {
            h.refreshContainerTags(ctx, session)
        }
        // Track event (PostHog/NATS)
        h.publisher.Publish("mcp.memory.saved", ...)
        return &ToolResult{Text: fmt.Sprintf("Saved to %s (ID: %s)", containerTag, result.ID)}

    case "forget":
        result, err := h.memoryClient.ForgetMemory(ctx, &ForgetMemoryRequest{
            Content:      args.Content,
            ContainerTag: containerTag,
        })
        h.publisher.Publish("mcp.memory.forgotten", ...)
        return &ToolResult{Text: result.Message}
    }
}
```

---

## 6. Tool: `recall` (Search + Profile)

```go
func (h *ToolHandler) HandleRecall(ctx context.Context, session *MCPSession, args RecallArgs) *ToolResult {
    containerTag := resolveContainerTag(args.ContainerTag, session.RootContainerTag)

    // Parallel: search + profile
    var wg sync.WaitGroup
    var searchResult *SearchResult
    var profile *ProfileResponse

    wg.Add(2)
    go func() { defer wg.Done(); searchResult, _ = h.searchClient.MemorySearch(ctx, ...) }()
    go func() { defer wg.Done(); profile, _ = h.profileClient.GetProfile(ctx, ...) }()
    wg.Wait()

    // Assemble response with deduplication
    deduped := deduplicateMemories(profile.Static, profile.Dynamic, searchResult.Results)
    return &ToolResult{Text: formatRecallResponse(deduped)}
}
```

---

## 7. gRPC Interface (Gateway proxy)

```protobuf
service MCPService {
  rpc HandleJSONRPC(JSONRPCRequest) returns (JSONRPCResponse);
  rpc StreamSSE(SSEStreamRequest) returns (stream SSEEvent);
  rpc GetSessionInfo(GetSessionInfoRequest) returns (SessionInfoResponse);
}
```
