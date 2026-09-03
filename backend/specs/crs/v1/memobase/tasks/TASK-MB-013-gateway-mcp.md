# TASK-MB-013 — Gateway MCP Server: 3 Memobase Tools & SSE Transport

**Wave:** 4 (Access Layer — song song với TASK-MB-012)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-MB-012 (gateway REST clients), TASK-MB-004/009/010 (services operational)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-MB-006 §7, §8](../solutions/SOL-MB-006-Gateway-MCP-Server.md)

**Trạng thái:** ⏳ Pending  
**Ghi chú:** MCP server for memobase not implemented  
---

## Mục tiêu

Mở rộng MCP server hiện có trong gateway để thêm 3 memobase-specific tools (`save_memory`, `get_user_profiles`, `search_memories`) và enable SSE (Server-Sent Events) transport trên port 8082 cho AI clients như Claude Desktop, Cursor, Windsurf.

---

## Cấu trúc thư mục

```
gateway/
└── internal/
    ├── adapter/
    │   ├── mcp/
    │   │   └── tools/
    │   │       └── memobase/                    ← [NEW directory]
    │   │           ├── save_memory.go           # Tool: save conversation to memory
    │   │           ├── get_user_profiles.go     # Tool: retrieve user profiles
    │   │           ├── search_memories.go       # Tool: semantic event search
    │   │           ├── types.go                 # Input/output structs
    │   │           └── register.go              # RegisterMemobaseTools()
    │   └── client/
    │       └── (ingestion, context, event clients — từ TASK-MB-012)
    └── infra/
        └── server/
            └── mcp_sse.go                       ← [NEW] SSE transport
```

---

## 1. MCP Tool Definitions

### File: `internal/adapter/mcp/tools/memobase/types.go`

```go
package memobasemcp

// save_memory input schema
type SaveMemoryRequest struct {
    UserID   string          `json:"user_id"`             // required
    Messages []ChatMessage   `json:"messages"`            // required: chat conversation
    Flush    bool            `json:"flush,omitempty"`     // optional: sync flush to LLM
}

type ChatMessage struct {
    Role    string `json:"role"`    // "user" | "assistant" | "system"
    Content string `json:"content"`
}

// get_user_profiles input schema
type GetUserProfilesRequest struct {
    UserID       string   `json:"user_id"`                    // required
    PreferTopics []string `json:"prefer_topics,omitempty"`    // topics to prioritize
    MaxTokenSize int      `json:"max_token_size,omitempty"`   // default: 500
}

// search_memories input schema
type SearchMemoriesRequest struct {
    UserID         string `json:"user_id"`                      // required
    Query          string `json:"query"`                        // required: semantic query
    TopK           int    `json:"topk,omitempty"`               // default: 5
    TimeRangeInDays int   `json:"time_range_in_days,omitempty"` // default: 21
}

// JSON schemas for MCP tool registration
var SaveMemoryInputSchema = map[string]any{
    "type": "object",
    "required": []string{"user_id", "messages"},
    "properties": map[string]any{
        "user_id": map[string]string{"type": "string", "description": "The user's unique ID"},
        "messages": map[string]any{
            "type": "array",
            "description": "The conversation messages to save",
            "items": map[string]any{
                "type": "object",
                "required": []string{"role", "content"},
                "properties": map[string]any{
                    "role":    map[string]string{"type": "string", "enum": "user,assistant,system"},
                    "content": map[string]string{"type": "string"},
                },
            },
        },
        "flush": map[string]any{"type": "boolean", "description": "If true, synchronously process with LLM (slower but immediate)"},
    },
}

var GetUserProfilesInputSchema = map[string]any{
    "type": "object",
    "required": []string{"user_id"},
    "properties": map[string]any{
        "user_id":       map[string]string{"type": "string"},
        "prefer_topics": map[string]any{"type": "array", "items": map[string]string{"type": "string"}},
        "max_token_size": map[string]any{"type": "integer", "default": 500},
    },
}

var SearchMemoriesInputSchema = map[string]any{
    "type": "object",
    "required": []string{"user_id", "query"},
    "properties": map[string]any{
        "user_id":          map[string]string{"type": "string"},
        "query":            map[string]string{"type": "string", "description": "Semantic search query"},
        "topk":             map[string]any{"type": "integer", "default": 5},
        "time_range_in_days": map[string]any{"type": "integer", "default": 21},
    },
}
```

---

## 2. Tool Handlers

### File: `internal/adapter/mcp/tools/memobase/save_memory.go`

```go
package memobasemcp

type SaveMemoryHandler struct {
    ingestionClient IngestionGRPCClient
}

func (h *SaveMemoryHandler) Handle(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
    var req SaveMemoryRequest
    if err := json.Unmarshal(args, &req); err != nil {
        return mcp.ErrorResult("invalid arguments: " + err.Error()), nil
    }
    if req.UserID == "" { return mcp.ErrorResult("user_id is required"), nil }
    if len(req.Messages) == 0 { return mcp.ErrorResult("messages cannot be empty"), nil }

    // Validate message roles
    for _, msg := range req.Messages {
        switch msg.Role {
        case "user", "assistant", "system": // valid
        default: return mcp.ErrorResult(fmt.Sprintf("invalid role: %s", msg.Role)), nil
        }
    }

    projectID := extractProjectIDFromCtx(ctx)

    // 1. Build ChatBlobData JSON
    chatBlob, _ := json.Marshal(map[string]any{"messages": req.Messages})

    // 2. Insert blob via ingestion service
    insertResp, err := h.ingestionClient.InsertBlob(ctx, &ingestionv1.InsertBlobRequest{
        UserId:    req.UserID,
        ProjectId: projectID,
        BlobType:  "chat",
        BlobData:  chatBlob,
    })
    if err != nil {
        return mcp.ErrorResult("failed to save memory: " + err.Error()), nil
    }

    // 3. Optional sync flush (triggers LLM processing immediately)
    flushed := false
    if req.Flush {
        _, flushErr := h.ingestionClient.FlushBuffer(ctx, &ingestionv1.FlushBufferRequest{
            UserId: req.UserID, ProjectId: projectID,
            BlobType: "chat", Sync: true,
        })
        flushed = flushErr == nil
        if flushErr != nil {
            slog.Warn("mcp save_memory: flush failed (blob saved)", "error", flushErr)
        }
    }

    return mcp.SuccessResult(map[string]any{
        "blob_id":         insertResp.BlobId,
        "flush_triggered": insertResp.FlushTriggered || flushed,
        "message":         "Memory saved successfully. Will be processed by LLM pipeline.",
    }), nil
}

func (h *SaveMemoryHandler) Description() string {
    return `Save a conversation or piece of information to the user's long-term memory.
The content will be processed by an LLM pipeline to extract profile facts and events.
Use flush=true to wait for LLM processing (slower, ~30-90s).`
}
```

### File: `internal/adapter/mcp/tools/memobase/get_user_profiles.go`

```go
type GetUserProfilesHandler struct {
    contextClient ContextGRPCClient
}

func (h *GetUserProfilesHandler) Handle(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
    var req GetUserProfilesRequest
    if err := json.Unmarshal(args, &req); err != nil {
        return mcp.ErrorResult("invalid arguments: " + err.Error()), nil
    }
    if req.UserID == "" { return mcp.ErrorResult("user_id is required"), nil }

    maxTokens := req.MaxTokenSize
    if maxTokens <= 0 { maxTokens = 500 }

    resp, err := h.contextClient.GetProfiles(ctx, &contextv1.GetProfilesRequest{
        UserId:       req.UserID,
        ProjectId:    extractProjectIDFromCtx(ctx),
        PreferTopics: req.PreferTopics,
        MaxTokenSize: int32(maxTokens),
    })
    if err != nil {
        return mcp.ErrorResult("failed to get profiles: " + err.Error()), nil
    }

    // Format profiles for LLM readability
    var profiles []map[string]string
    for _, p := range resp.Profiles {
        profiles = append(profiles, map[string]string{
            "topic":     p.Attributes.Topic,
            "sub_topic": p.Attributes.SubTopic,
            "content":   p.Content,
        })
    }

    return mcp.SuccessResult(map[string]any{
        "user_id":  req.UserID,
        "profiles": profiles,
        "count":    len(profiles),
    }), nil
}

func (h *GetUserProfilesHandler) Description() string {
    return `Retrieve the structured user profile information extracted from past conversations.
Profiles are organized by topic (e.g., basic_info, work, lifestyle, interests).
Use prefer_topics to prioritize specific categories.`
}
```

### File: `internal/adapter/mcp/tools/memobase/search_memories.go`

```go
type SearchMemoriesHandler struct {
    eventClient EventGRPCClient
}

func (h *SearchMemoriesHandler) Handle(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
    var req SearchMemoriesRequest
    if err := json.Unmarshal(args, &req); err != nil {
        return mcp.ErrorResult("invalid arguments: " + err.Error()), nil
    }
    if req.UserID == "" { return mcp.ErrorResult("user_id is required"), nil }
    if req.Query == "" { return mcp.ErrorResult("query is required"), nil }

    topK := req.TopK
    if topK <= 0 { topK = 5 }
    timeRange := req.TimeRangeInDays
    if timeRange <= 0 { timeRange = 21 }

    resp, err := h.eventClient.SearchEvents(ctx, &eventv1.SearchEventsRequest{
        UserId:          req.UserID,
        ProjectId:       extractProjectIDFromCtx(ctx),
        Query:           req.Query,
        Topk:            int32(topK),
        TimeRangeInDays: int32(timeRange),
        SimilarityThreshold: 0.2,
    })
    if err != nil {
        return mcp.ErrorResult("failed to search memories: " + err.Error()), nil
    }

    var results []map[string]any
    for _, r := range resp.Results {
        results = append(results, map[string]any{
            "event_summary": r.Event.EventData.EventTip,
            "similarity":    fmt.Sprintf("%.2f", r.Similarity),
            "date":          r.Event.CreatedAt.Format("2006-01-02"),
            "tags":          extractTagNames(r.Event.EventData.EventTags),
        })
    }

    return mcp.SuccessResult(map[string]any{
        "query":   req.Query,
        "results": results,
        "count":   len(results),
    }), nil
}

func (h *SearchMemoriesHandler) Description() string {
    return `Search through the user's event timeline using semantic similarity.
Returns past events and conversations most relevant to the query.
Use time_range_in_days to limit the search window (default: 21 days).`
}
```

---

## 3. Tool Registration

### File: `internal/adapter/mcp/tools/memobase/register.go`

```go
package memobasemcp

type MemobaseToolDeps struct {
    IngestionClient IngestionGRPCClient
    ContextClient   ContextGRPCClient
    EventClient     EventGRPCClient
}

func RegisterMemobaseTools(server MCPServer, deps MemobaseToolDeps) {
    server.RegisterTool(mcp.ToolDefinition{
        Name:        "save_memory",
        Description: (&SaveMemoryHandler{}).Description(),
        InputSchema: SaveMemoryInputSchema,
        Handler:     (&SaveMemoryHandler{ingestionClient: deps.IngestionClient}).Handle,
    })

    server.RegisterTool(mcp.ToolDefinition{
        Name:        "get_user_profiles",
        Description: (&GetUserProfilesHandler{}).Description(),
        InputSchema: GetUserProfilesInputSchema,
        Handler:     (&GetUserProfilesHandler{contextClient: deps.ContextClient}).Handle,
    })

    server.RegisterTool(mcp.ToolDefinition{
        Name:        "search_memories",
        Description: (&SearchMemoriesHandler{}).Description(),
        InputSchema: SearchMemoriesInputSchema,
        Handler:     (&SearchMemoriesHandler{eventClient: deps.EventClient}).Handle,
    })
}
```

---

## 4. SSE Transport

### File: `internal/infra/server/mcp_sse.go`

```go
package server

// MCP Server-Sent Events transport
// Endpoint: GET /mcp/sse
// Port: 8082 (separate from main HTTP port 8080)
// Allows remote connections from Claude Desktop, Cursor, Windsurf

type MCPSSEServer struct {
    mcpServer MCPServer
}

func (s *MCPSSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // SSE response headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")  // MCP clients need CORS
    w.Header().Set("X-Accel-Buffering", "no")           // Disable nginx buffering

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    // Create SSE session for this connection
    session := s.mcpServer.NewSSESession(r.Context())
    defer session.Close()

    slog.Info("mcp: SSE client connected", "remote", r.RemoteAddr)

    for {
        select {
        case msg, ok := <-session.Messages():
            if !ok { return }
            fmt.Fprintf(w, "data: %s\n\n", msg)
            flusher.Flush()
        case <-r.Context().Done():
            slog.Info("mcp: SSE client disconnected", "remote", r.RemoteAddr)
            return
        }
    }
}

// Start SSE server on separate port (default :8082)
func StartMCPSSEServer(ctx context.Context, mcpServer MCPServer, addr string) error {
    sseServer := &MCPSSEServer{mcpServer: mcpServer}
    mux := http.NewServeMux()
    mux.Handle("/mcp/sse", sseServer)
    mux.Handle("/mcp/message", mcpServer.HTTPMessageHandler())  // POST for client → server msgs

    httpServer := &http.Server{Addr: addr, Handler: mux}

    go func() {
        <-ctx.Done()
        httpServer.Shutdown(context.Background())
    }()

    slog.Info("mcp: SSE server starting", "addr", addr)
    return httpServer.ListenAndServe()
}
```

---

## 5. Config (update gateway config)

```yaml
mcp:
  enabled: true
  sse:
    enabled: true
    port: 8082         # MEMOBASE_MCP_SSE_PORT
    path: "/mcp/sse"
  stdio:
    enabled: true      # For local stdio connections (Claude Desktop native mode)
  tools:
    save_memory:
      enabled: true
    get_user_profiles:
      enabled: true
    search_memories:
      enabled: true
```

---

## 6. Gateway Bootstrap Integration

```go
// apps/memory/internal/bootstrap/gateway.go (MODIFY)
func bootstrapGateway(ctx context.Context, cfg *config.Config, registry *bus.InProcessRegistry) error {
    // ... existing setup ...

    // Dial memobase services (in-process via bufconn)
    ingestionConn, _ := registry.Dial("memobase-ingestion")
    contextConn, _   := registry.Dial("memobase-context")
    eventConn, _     := registry.Dial("memobase-event")
    adminConn, _     := registry.Dial("memobase-admin")

    // Register 3 memobase MCP tools
    memobasemcp.RegisterMemobaseTools(mcpServer, memobasemcp.MemobaseToolDeps{
        IngestionClient: ingestionv1.NewIngestionServiceClient(ingestionConn),
        ContextClient:   contextv1.NewContextServiceClient(contextConn),
        EventClient:     eventv1.NewEventServiceClient(eventConn),
    })

    // Start SSE server on separate goroutine
    if cfg.MCP.SSE.Enabled {
        go func() {
            addr := fmt.Sprintf(":%d", cfg.MCP.SSE.Port)
            if err := server.StartMCPSSEServer(ctx, mcpServer, addr); err != nil {
                slog.Error("mcp: SSE server failed", "error", err)
            }
        }()
    }

    return nil
}
```

---

## Unit Tests

```
TestSaveMemoryHandler_ValidChat               → InsertBlob called → success result
TestSaveMemoryHandler_EmptyUserID             → error "user_id is required"
TestSaveMemoryHandler_EmptyMessages           → error "messages cannot be empty"
TestSaveMemoryHandler_InvalidRole             → "model" role → error returned
TestSaveMemoryHandler_WithFlush               → flush=true → FlushBuffer called
TestSaveMemoryHandler_FlushFails_BlobStillSaved → flush error → blob_id still in result
TestSaveMemoryHandler_InsertFails             → gRPC error → error result
TestGetUserProfilesHandler_ValidRequest       → GetProfiles called → profiles returned
TestGetUserProfilesHandler_EmptyUserID        → error "user_id is required"
TestGetUserProfilesHandler_EmptyProfiles      → empty list → count=0
TestGetUserProfilesHandler_DefaultMaxTokens   → max_token_size=0 → 500 used
TestGetUserProfilesHandler_PreferTopics       → prefer_topics forwarded to gRPC
TestGetUserProfilesHandler_FormatsStructured  → profiles as {topic, sub_topic, content}
TestSearchMemoriesHandler_ValidQuery          → SearchEvents called → results returned
TestSearchMemoriesHandler_EmptyUserID         → error "user_id is required"
TestSearchMemoriesHandler_EmptyQuery          → error "query is required"
TestSearchMemoriesHandler_DefaultTopK         → topk=0 → 5 used
TestSearchMemoriesHandler_FormatsResults      → similarity formatted as "0.85"
TestSearchMemoriesHandler_NoResults           → empty → count=0
TestRegisterMemobaseTools_RegistrationCount   → 3 tools registered
TestMCPSSEServer_Connection                   → SSE headers set correctly
TestMCPSSEServer_Disconnect                   → ctx cancel → session closed
TestMCPSSEServer_MessageStreaming             → message → "data: {msg}\n\n" written
TestExtractProjectIDFromCtx_Set              → context has project ID → returned
TestExtractProjectIDFromCtx_NotSet           → empty context → ""
TestSaveMemoryInputSchema_Valid              → matches JSON Schema draft spec
TestGetUserProfilesInputSchema_Required      → "user_id" in required array
TestSearchMemoriesInputSchema_Required       → "user_id", "query" in required
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./gateway/internal/adapter/mcp/...
go build ./gateway/internal/infra/server/...
go test ./gateway/internal/adapter/mcp/... -v -count=1

# Test SSE endpoint
curl -N -H "Accept: text/event-stream" http://localhost:8082/mcp/sse

# MCP tool via HTTP (for testing)
curl -X POST http://localhost:8082/mcp/message \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-proj-test-..." \
  -d '{"method":"tools/list","id":1}'
```

---

## Ghi chú triển khai

- **SSE Port 8082**: riêng biệt với REST port 8080 — allow CORS `*` cho MCP clients
- **MCP CORS**: SSE endpoint dùng `Access-Control-Allow-Origin: *` vì MCP clients không gửi Origin header cố định
- **Stdio mode**: vẫn hỗ trợ cho local Claude Desktop native connections (không qua network)
- **`extractProjectIDFromCtx`**: dùng context key giống gateway REST middleware — đảm bảo project scoping nhất quán
- **Flush với flush=false** (default): non-blocking — blob saved, LLM picks up async
- **Tool result format**: `mcp.SuccessResult(data)` → `{"content": [{"type": "text", "text": "{json}"}]}`
- MCP library: dùng library hiện có trong gateway hoặc `github.com/mark3labs/mcp-go`
