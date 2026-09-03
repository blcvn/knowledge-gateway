# TASK-OV-017 — `services/openviking-gateway` MCP Server & WebDAV Proxy

**Wave:** 6 (Gateway)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-016 (REST gateway base)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-OV-001 §6, §7, §8](../solutions/SOL-OV-001-Gateway-Service.md)  
**Ports:** MCP=8082, WebDAV=8083

---

## Mục tiêu

Hoàn thiện `services/openviking-gateway/` với:
1. **MCP Server** (Model Context Protocol 2025-03-26): 9 tools cho Claude, Codex, Gemini
2. **WebDAV Proxy**: mount VikingFS như network drive (macOS Finder, VS Code)
3. **gRPC passthrough** port 8081

---

## 1. MCP Streamable HTTP Server

**Port:** 8082  
**Spec:** MCP 2025-03-26 (`Streamable HTTP` transport)

### Protocol Handler

**File: `internal/adapter/mcp/server.go`**

```go
// MCP Streamable HTTP protocol:
// Client → POST /mcp → Server sends NDJSON stream (newline-delimited JSON)
// Session via SSE or plain NDJSON stream
// Auth: same as REST (X-Api-Key header or Authorization: Bearer)

type MCPServer struct {
    tools   []MCPTool
    authMW  *middleware.AuthMiddleware
}

type MCPTool struct {
    Name        string
    Description string
    InputSchema map[string]any  // JSON Schema
    Handler     ToolHandler
}

type ToolHandler func(ctx context.Context, args map[string]any) (*MCPToolResult, error)

type MCPMessage struct {
    JSONRPC string `json:"jsonrpc"` // "2.0"
    Method  string `json:"method,omitempty"`
    ID      any    `json:"id,omitempty"`
    Params  any    `json:"params,omitempty"`
    Result  any    `json:"result,omitempty"`
    Error   *MCPError `json:"error,omitempty"`
}

type MCPError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}

type MCPToolResult struct {
    Content []MCPContent `json:"content"`
    IsError bool         `json:"isError,omitempty"`
}

type MCPContent struct {
    Type string `json:"type"`  // "text" | "resource"
    Text string `json:"text,omitempty"`
    URI  string `json:"uri,omitempty"`
}

// POST /mcp — main endpoint
func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Authenticate (same auth middleware)
    rc, err := s.authMW.Resolve(r)
    if err != nil { httpError(w, 401, err.Error()); return }
    ctx := viking.WithContext(r.Context(), rc)
    
    var msg MCPMessage
    json.NewDecoder(r.Body).Decode(&msg)
    
    switch msg.Method {
    case "initialize":
        s.handleInitialize(w, &msg, ctx)
    case "tools/list":
        s.handleToolsList(w, &msg, ctx)
    case "tools/call":
        s.handleToolCall(w, &msg, ctx)
    default:
        writeMCPError(w, msg.ID, -32601, "Method not found")
    }
}

// Response for "initialize"
func (s *MCPServer) handleInitialize(w http.ResponseWriter, msg *MCPMessage, ctx context.Context) {
    writeMCPResult(w, msg.ID, map[string]any{
        "protocolVersion": "2025-03-26",
        "capabilities": map[string]any{
            "tools": map[string]any{"listChanged": false},
        },
        "serverInfo": map[string]any{
            "name":    "openviking",
            "version": "1.0.0",
        },
    })
}

// Response for "tools/list"
func (s *MCPServer) handleToolsList(w http.ResponseWriter, msg *MCPMessage, ctx context.Context) {
    tools := make([]map[string]any, len(s.tools))
    for i, t := range s.tools {
        tools[i] = map[string]any{
            "name":        t.Name,
            "description": t.Description,
            "inputSchema": t.InputSchema,
        }
    }
    writeMCPResult(w, msg.ID, map[string]any{"tools": tools})
}

// Response for "tools/call"
func (s *MCPServer) handleToolCall(w http.ResponseWriter, msg *MCPMessage, ctx context.Context) {
    params, _ := msg.Params.(map[string]any)
    toolName, _ := params["name"].(string)
    args, _ := params["arguments"].(map[string]any)
    
    for _, tool := range s.tools {
        if tool.Name == toolName {
            result, err := tool.Handler(ctx, args)
            if err != nil {
                writeMCPResult(w, msg.ID, &MCPToolResult{
                    IsError: true,
                    Content: []MCPContent{{Type: "text", Text: err.Error()}},
                })
                return
            }
            writeMCPResult(w, msg.ID, result)
            return
        }
    }
    writeMCPError(w, msg.ID, -32602, fmt.Sprintf("tool not found: %s", toolName))
}
```

### 9 MCP Tools

**File: `internal/adapter/mcp/tools.go`**

```go
func RegisterAllTools(
    fsClient port.FSClient,
    searchClient port.SearchClient,
    sessionClient port.SessionClient,
) []MCPTool {
    return []MCPTool{
        
        // 1. viking_read_context
        {
            Name: "viking_read_context",
            Description: "Read file content from VikingFS. Supports tiered context levels (abstract/overview/detail).",
            InputSchema: schema("uri", "string", "Level: 0=abstract, 1=overview, 2=detail (default: 1)"),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                uri, _ := args["uri"].(string)
                level := intArg(args, "level", 1)
                content, err := fsClient.Read(ctx, uri, level)
                if err != nil { return nil, err }
                return textResult(string(content)), nil
            },
        },
        
        // 2. viking_write_context
        {
            Name: "viking_write_context",
            Description: "Write content to a file in VikingFS (encrypted if enabled).",
            InputSchema: schema("uri,content", "string,string", ""),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                rc, _ := viking.FromContext(ctx)
                uri, _ := args["uri"].(string)
                content, _ := args["content"].(string)
                err := fsClient.Write(ctx, uri, []byte(content), rc.User.AccountID)
                if err != nil { return nil, err }
                return textResult("OK"), nil
            },
        },
        
        // 3. viking_list_directory
        {
            Name: "viking_list_directory",
            Description: "List directory contents in VikingFS.",
            InputSchema: schema("uri", "string", ""),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                uri, _ := args["uri"].(string)
                entries, err := fsClient.Ls(ctx, uri)
                if err != nil { return nil, err }
                data, _ := json.Marshal(entries)
                return textResult(string(data)), nil
            },
        },
        
        // 4. viking_search_memory
        {
            Name: "viking_search_memory",
            Description: "Semantic search across agent memory (user+agent namespaces, tiered retrieval).",
            InputSchema: schema("query,account_id,limit", "string,string,integer", ""),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                query, _ := args["query"].(string)
                accountID, _ := args["account_id"].(string)
                limit := intArg(args, "limit", 10)
                results, err := searchClient.Find(ctx, query, accountID, -1, nil, limit)
                if err != nil { return nil, err }
                data, _ := json.Marshal(results)
                return textResult(string(data)), nil
            },
        },
        
        // 5. viking_search_resources
        {
            Name: "viking_search_resources",
            Description: "Search across ingested resources (code, docs). Returns relevant file snippets.",
            InputSchema: schema("query,account_id,limit", "string,string,integer", ""),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                query, _ := args["query"].(string)
                accountID, _ := args["account_id"].(string)
                limit := intArg(args, "limit", 10)
                ctxType := 1  // RESOURCE
                results, err := searchClient.Find(ctx, query, accountID, ctxType, nil, limit)
                if err != nil { return nil, err }
                data, _ := json.Marshal(results)
                return textResult(string(data)), nil
            },
        },
        
        // 6. viking_add_message
        {
            Name: "viking_add_message",
            Description: "Add message(s) to an agent session. Triggers auto-commit if token threshold exceeded.",
            InputSchema: schema("session_id,account_id,messages", "string,string,array", ""),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                sessionID, _ := args["session_id"].(string)
                accountID, _ := args["account_id"].(string)
                msgs := parseMessages(args["messages"])
                err := sessionClient.AddMessages(ctx, sessionID, accountID, msgs)
                if err != nil { return nil, err }
                return textResult("OK"), nil
            },
        },
        
        // 7. viking_commit_session
        {
            Name: "viking_commit_session",
            Description: "Commit a session: archives messages, updates Working Memory, extracts memories.",
            InputSchema: schema("session_id,account_id", "string,string", "Optional: force=true"),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                sessionID, _ := args["session_id"].(string)
                accountID, _ := args["account_id"].(string)
                force, _ := args["force"].(bool)
                resp, err := sessionClient.Commit(ctx, sessionID, accountID, force)
                if err != nil { return nil, err }
                data, _ := json.Marshal(resp)
                return textResult(string(data)), nil
            },
        },
        
        // 8. viking_grep
        {
            Name: "viking_grep",
            Description: "Search file content with regex pattern in VikingFS.",
            InputSchema: schema("uri,pattern,account_id", "string,string,string", ""),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                uri, _ := args["uri"].(string)
                pattern, _ := args["pattern"].(string)
                accountID, _ := args["account_id"].(string)
                matches, err := fsClient.Grep(ctx, uri, pattern, accountID)
                if err != nil { return nil, err }
                data, _ := json.Marshal(matches)
                return textResult(string(data)), nil
            },
        },
        
        // 9. viking_manage_resource
        {
            Name: "viking_manage_resource",
            Description: "Ingest a resource (git repo, URL, local file/dir). Returns task_id for progress tracking.",
            InputSchema: schema("action,account_id,url_or_path,name", "string,string,string,string",
                "action: add|refresh|delete|status"),
            Handler: func(ctx context.Context, args map[string]any) (*MCPToolResult, error) {
                action, _ := args["action"].(string)
                accountID, _ := args["account_id"].(string)
                name, _ := args["name"].(string)
                // Dispatch based on action
                // ... (call resourceClient)
                return textResult("task_id: abc123"), nil
            },
        },
    }
}

// Helpers
func textResult(text string) *MCPToolResult {
    return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: text}}}
}

func intArg(args map[string]any, key string, def int) int {
    if v, ok := args[key]; ok {
        if n, ok := v.(float64); ok { return int(n) }
    }
    return def
}
```

---

## 2. WebDAV Proxy

**Port:** 8083

**File: `internal/adapter/webdav/server.go`**

```go
// WebDAV maps HTTP WebDAV operations to VikingFS gRPC calls
// Allows mounting as network drive (macOS: mount via Finder, VS Code via Mounts)

// golang.org/x/net/webdav — standard WebDAV library
// Custom FileSystem implementation that delegates to FSClient

type VikingWebDAVFS struct {
    fsClient    port.FSClient
    accountID   string
    baseURI     string  // "viking://user/{accountID}/" for USER role
}

func (fs *VikingWebDAVFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
    uri := fs.localToURI(name)
    return fs.fsClient.Mkdir(ctx, uri, false)
}

func (fs *VikingWebDAVFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
    uri := fs.localToURI(name)
    if flag&os.O_RDONLY != 0 || flag == 0 {
        content, err := fs.fsClient.Read(ctx, uri, 2)
        if err != nil { return nil, err }
        return newInMemoryFile(name, content), nil
    }
    // Write mode: return a WriteBuffer that flushes to FSClient on Close()
    return newWriteBufferFile(name, uri, fs.fsClient, fs.accountID), nil
}

func (fs *VikingWebDAVFS) RemoveAll(ctx context.Context, name string) error {
    return fs.fsClient.Rm(ctx, fs.localToURI(name), true)
}

func (fs *VikingWebDAVFS) Rename(ctx context.Context, oldName, newName string) error {
    return fs.fsClient.Mv(ctx, fs.localToURI(oldName), fs.localToURI(newName))
}

func (fs *VikingWebDAVFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
    entry, err := fs.fsClient.Stat(ctx, fs.localToURI(name))
    if err != nil { return nil, err }
    return &webdavFileInfo{entry: entry}, nil
}

// URI mapping: WebDAV /notes/foo.md → viking://user/{accountID}/{userID}/notes/foo.md
func (fs *VikingWebDAVFS) localToURI(name string) string {
    cleaned := path.Clean("/" + name)
    return fs.baseURI + strings.TrimPrefix(cleaned, "/")
}

// Setup WebDAV server
func NewWebDAVServer(authMW *middleware.AuthMiddleware, fsClient port.FSClient) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rc, err := authMW.Resolve(r)
        if err != nil { http.Error(w, "Unauthorized", 401); return }
        
        vfs := &VikingWebDAVFS{
            fsClient:  fsClient,
            accountID: rc.User.AccountID,
            baseURI:   fmt.Sprintf("viking://user/%s/%s/", rc.User.AccountID, rc.User.UserID),
        }
        
        handler := &webdav.Handler{
            FileSystem: vfs,
            LockSystem: webdav.NewMemLS(),  // In-memory lock system
            Logger: func(r *http.Request, err error) {
                slog.Debug("webdav", "method", r.Method, "path", r.URL.Path, "error", err)
            },
        }
        
        handler.ServeHTTP(w, r)
    })
}
```

---

## 3. gRPC Passthrough (Port 8081)

**File: `internal/adapter/grpc_passthrough/server.go`**

```go
// gRPC-Web + gRPC proxy on port 8081
// Forwards to backend services based on service name prefix

type GRPCPassthrough struct {
    conns map[string]*grpc.ClientConn  // service → conn
}

// Options:
// Option 1: grpc-proxy library (transparent proxy)
// Option 2: grpcwebproxy (Google's gRPC-web proxy)
// Recommendation: Use grpcwebproxy for simplicity

// In main.go:
// Start grpcwebproxy in subprocess OR use grpc-go transport
```

---

## 4. Main.go — Complete Gateway

**File: `cmd/server/main.go`**

```go
func main() {
    cfg := config.Load()
    
    // Wire gRPC clients (with circuit breakers)
    fsClient, _ := client.NewFSClient(cfg.Upstream.FS)
    searchClient, _ := client.NewSearchClient(cfg.Upstream.Search)
    sessionClient, _ := client.NewSessionClient(cfg.Upstream.Session)
    resourceClient, _ := client.NewResourceClient(cfg.Upstream.Resource)
    adminClient, _ := client.NewAdminClient(cfg.Upstream.Admin)
    
    // Auth middleware
    keyResolver := auth.NewCachedKeyResolver(adminClient, redisClient, 5*time.Minute)
    authMW := middleware.NewAuthMiddleware(cfg.Auth.Mode, keyResolver)
    
    // Rate limiter
    rateLimiter, _ := ratelimit.NewRateLimiter(ratelimit.Config{
        RedisURL: cfg.RateLimit.RedisURL,
        Window: cfg.RateLimit.Window,
        Max: cfg.RateLimit.RequestsPerWindow,
    })
    
    // REST Server (8080)
    restRouter := router.NewRouter(&router.Dependencies{
        FSClient: fsClient, SearchClient: searchClient,
        SessionClient: sessionClient, ResourceClient: resourceClient,
        AdminClient: adminClient, AuthMW: authMW, RateLimitMW: rateLimiter,
    })
    
    // MCP Server (8082)
    mcpTools := mcp.RegisterAllTools(fsClient, searchClient, sessionClient)
    mcpServer := mcp.NewMCPServer(mcpTools, authMW)
    
    // WebDAV Server (8083)
    webdavHandler := webdav.NewWebDAVServer(authMW, fsClient)
    
    // Start all 3 servers concurrently
    g, _ := errgroup.WithContext(context.Background())
    
    g.Go(func() error {
        slog.Info("REST API listening", "port", cfg.HTTP.Port)
        return http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTP.Port), restRouter)
    })
    
    g.Go(func() error {
        slog.Info("MCP server listening", "port", 8082)
        return http.ListenAndServe(":8082", mcpServer)
    })
    
    g.Go(func() error {
        slog.Info("WebDAV server listening", "port", 8083)
        return http.ListenAndServe(":8083", webdavHandler)
    })
    
    if err := g.Wait(); err != nil {
        slog.Error("server error", "error", err)
        os.Exit(1)
    }
}
```

---

## Unit Tests

```
// MCP Protocol
TestMCPServer_Initialize                 → POST /mcp {"method":"initialize"} → protocolVersion
TestMCPServer_ToolsList                  → POST /mcp {"method":"tools/list"} → 9 tools
TestMCPServer_ToolsList_Names            → all 9 tool names present
TestMCPToolCall_ReadContext              → viking_read_context → fsClient.Read called
TestMCPToolCall_WriteContext             → viking_write_context → fsClient.Write called
TestMCPToolCall_ListDirectory            → viking_list_directory → fsClient.Ls called
TestMCPToolCall_SearchMemory             → viking_search_memory → searchClient.Find called
TestMCPToolCall_SearchResources          → viking_search_resources → ctxType=RESOURCE
TestMCPToolCall_AddMessage               → viking_add_message → sessionClient.AddMessages called
TestMCPToolCall_CommitSession            → viking_commit_session → sessionClient.Commit called
TestMCPToolCall_Grep                     → viking_grep → fsClient.Grep called
TestMCPToolCall_UnknownTool_Error        → unknown tool → error in result
TestMCPServer_Auth_Required              → no API key → 401
TestMCPServer_InvalidJSON_Error          → invalid JSON → jsonrpc error response

// WebDAV
TestWebDAVFS_LocalToURI_User             → /notes/foo → viking://user/acct/alice/notes/foo
TestWebDAVFS_Mkdir                       → MKCOL /docs → fsClient.Mkdir called
TestWebDAVFS_Stat                        → PROPFIND / → 207 Multi-Status
TestWebDAVFS_Put                         → PUT /test.md + body → fsClient.Write called
TestWebDAVFS_Delete                      → DELETE /test.md → fsClient.Rm called
TestWebDAVFS_Rename                      → MOVE /old → /new → fsClient.Mv called
TestWebDAVServer_Auth_Required           → no auth → 401

// Integration
TestEndToEnd_MCPReadAfterWrite           → REST write → MCP read → same content
TestEndToEnd_WebDAVMountFlow             → MKCOL → PUT → PROPFIND → DELETE
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go get golang.org/x/net/webdav
go build ./services/openviking-gateway/...
go test ./services/openviking-gateway/... -v -count=1

# Test MCP
curl -s -X POST http://localhost:8082/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}'

# Test WebDAV (macOS mount)
# Finder → Go → Connect to Server → http://localhost:8083

# curl WebDAV
curl -s -X PROPFIND http://localhost:8083/ -H "Depth: 0"
```

---

## Ghi chú triển khai

- MCP response format: `Content-Type: application/json` (không phải NDJSON streaming cho simple calls)
- MCP streaming: implement SSE streaming nếu client gửi `Accept: text/event-stream`
- WebDAV LockSystem: `webdav.NewMemLS()` là in-memory — đủ cho single-instance; distributed lock cần Redis-backed
- `schema()` helper: tạo JSON Schema object từ comma-separated field/type strings
- 9 tools phải khớp đúng tên với spec trong `SOL-OV-001`
