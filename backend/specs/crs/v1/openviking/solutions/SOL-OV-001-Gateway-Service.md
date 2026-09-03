# Solution: SOL-OV-001 — Unified Gateway Service (REST, MCP, WebDAV)

**CR:** [CR-OV-001](../CR-OV-001-Gateway-Service.md)  
**Wave:** 6 (Gateway — xây cuối cùng, sau tất cả backend services)  
**Priority:** Critical  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/openviking-gateway` — entry point duy nhất cho hệ thống OpenViking. Gateway là **Presentation Layer thuần túy**: không chứa business logic, chỉ nhận request, xác thực, route tới backend gRPC services, và format response.

### Ports
| Protocol | Port | Mô tả |
|---|---|---|
| REST HTTP | 8080 | Main REST API (17 route groups) |
| gRPC/gRPC-Web | 8081 | gRPC passthrough |
| MCP Streamable HTTP | 8082 | 9 AI tools cho Claude, Codex, etc. |
| Health | 8083 | /healthz, /readyz, /metrics |

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Thiếu unified entry point | chi router + gRPC clients tới 6 backend services |
| Chưa có MCP server đủ 9 tools | Streamable HTTP MCP server (spec 2025-03-26) |
| Chưa có WebDAV support | WebDAV protocol adapter → gRPC FS calls |
| 3-mode auth phức tạp | Strategy pattern: DEV / API_KEY / TRUSTED |
| API key bcrypt overhead | Redis cache (TTL=5min) trong `pkg/auth/` |
| Backend down → client hangs | Circuit breaker (sony/gobreaker) per upstream |

---

## 2. Codebase Structure

```
services/openviking-gateway/
├── cmd/server/main.go
├── api/openapi/openviking.yaml          # OpenAPI 3.1 specification
├── internal/
│   ├── domain/
│   │   ├── identity.go      # RequestContext, UserIdentifier, Role
│   │   ├── auth_mode.go     # AuthMode enum
│   │   └── errors.go
│   ├── usecase/
│   │   ├── resolve_identity.go  # Identity resolution logic
│   │   ├── check_namespace.go   # Namespace access validation
│   │   └── port/output.go       # AdminClient interface (for API key resolve)
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── router.go              # chi router setup, all 17 route groups
│   │   │   ├── filesystem_handler.go  # ls, tree, mkdir, rm
│   │   │   ├── content_handler.go     # read, write, mv, cp, stat
│   │   │   ├── search_handler.go      # find, search, grep, glob
│   │   │   ├── session_handler.go     # create, add-messages, commit, delete
│   │   │   ├── resource_handler.go    # add, list, delete, refresh, status
│   │   │   ├── relations_handler.go   # get, add, remove
│   │   │   ├── admin_handler.go       # accounts, users, api-keys
│   │   │   ├── observer_handler.go    # retrieval stats, replay search
│   │   │   ├── privacy_handler.go     # privacy config CRUD
│   │   │   ├── tasks_handler.go       # task status tracking
│   │   │   ├── system_handler.go      # status, wait-ready, info
│   │   │   ├── debug_handler.go       # graph inspect, debug flags
│   │   │   ├── bot_handler.go         # VikingBot integration endpoints
│   │   │   ├── pack_handler.go        # context export/pack
│   │   │   ├── maintenance_handler.go # storage cleanup, index rebuild
│   │   │   └── stats_handler.go       # usage statistics per tenant
│   │   ├── mcp/
│   │   │   ├── server.go              # MCP Streamable HTTP server
│   │   │   ├── search_tool.go
│   │   │   ├── read_tool.go
│   │   │   ├── list_tool.go
│   │   │   ├── store_tool.go
│   │   │   ├── add_resource_tool.go
│   │   │   ├── grep_tool.go
│   │   │   ├── glob_tool.go
│   │   │   ├── forget_tool.go
│   │   │   └── health_tool.go
│   │   ├── webdav/
│   │   │   └── handler.go             # WebDAV → gRPC FS proxy
│   │   └── client/
│   │       ├── fs_client.go
│   │       ├── search_client.go
│   │       ├── session_client.go
│   │       ├── resource_client.go
│   │       └── admin_client.go
│   └── infra/
│       ├── config/config.go
│       ├── server/
│       │   ├── http.go        # chi server setup
│       │   └── grpc.go        # gRPC server (passthrough)
│       ├── middleware/        # HTTP middleware chain
│       └── wire/wire.go
```

---

## 3. Router Setup — 17 Route Groups

```go
// internal/adapter/http/router.go

func NewRouter(
    deps *Dependencies,
    authMW,
    tenantMW,
    rateLimitMW func(http.Handler) http.Handler,
) http.Handler {
    r := chi.NewRouter()
    
    // Global middleware
    r.Use(
        middleware.RequestID,
        middleware.RealIP,
        otelMiddleware.OpenTelemetry,           // Distributed tracing
        loggingMiddleware.StructuredLogger,     // Structured access log
        middleware.Recoverer,                    // Panic recovery
    )
    
    // Health & Metrics (no auth required)
    r.Get("/healthz",  deps.System.HealthzHandler)
    r.Get("/readyz",   deps.System.ReadyzHandler)
    r.Get("/metrics",  promhttp.Handler())
    
    // Authenticated API routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Use(authMW, tenantMW)  // Auth resolved BEFORE rate limit (for per-tenant limits)
        r.Use(rateLimitMW)
        
        // 17 route groups
        r.Mount("/filesystem",      newFilesystemRouter(deps.FS))
        r.Mount("/content",         newContentRouter(deps.FS))
        r.Mount("/search",          newSearchRouter(deps.Search))
        r.Mount("/sessions",        newSessionRouter(deps.Session))
        r.Mount("/resources",       newResourceRouter(deps.Resource))
        r.Mount("/relations",       newRelationsRouter(deps.FS))
        r.Mount("/admin",           newAdminRouter(deps.Admin))
        r.Mount("/observer",        newObserverRouter(deps.Search))
        r.Mount("/privacy-configs", newPrivacyRouter(deps.FS))
        r.Mount("/tasks",           newTaskRouter(deps.Resource))
        r.Mount("/system",          newSystemRouter(deps.Admin))
        r.Mount("/debug",           newDebugRouter(deps.Search, deps.Admin))
        r.Mount("/bot",             newBotRouter(deps.Session, deps.Search))
        r.Mount("/pack",            newPackRouter(deps.FS))
        r.Mount("/maintenance",     newMaintenanceRouter(deps.Admin))
        r.Mount("/stats",           newStatsRouter(deps.Admin))
    })
    
    // MCP server (auth via headers, not Bearer)
    r.Mount("/mcp", deps.MCP.Handler())
    
    // WebDAV (auth via standard WebDAV headers)
    r.Mount("/webdav", deps.WebDAV.Handler())
    
    return r
}
```

---

## 4. Authentication Middleware — 3 Modes

```go
// internal/adapter/http/middleware/auth.go

type AuthMiddleware struct {
    mode        domain.AuthMode
    keyResolver auth.KeyResolver  // from pkg/auth/
    localIPs    []net.IPNet       // For DEV mode: 127.0.0.1/8, ::1/128
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var rc *viking.RequestContext
        var err error
        
        switch m.mode {
        case domain.AuthModeDev:
            rc, err = m.resolveDevMode(r)
        case domain.AuthModeAPIKey:
            rc, err = m.resolveAPIKeyMode(r)
        case domain.AuthModeTrusted:
            rc, err = m.resolveTrustedMode(r)
        }
        
        if err != nil {
            httpError(w, http.StatusUnauthorized, err.Error())
            return
        }
        
        // Inject RequestContext into request context
        ctx := viking.WithContext(r.Context(), rc)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// DEV mode: localhost only, ROOT role, no key needed
func (m *AuthMiddleware) resolveDevMode(r *http.Request) (*viking.RequestContext, error) {
    clientIP := net.ParseIP(getRealIP(r))
    for _, local := range m.localIPs {
        if local.Contains(clientIP) {
            return &viking.RequestContext{
                Role: viking.RoleRoot,
                User: viking.UserIdentifier{
                    AccountID: r.Header.Get("X-OpenViking-Account"),
                    UserID:    r.Header.Get("X-OpenViking-User"),
                    AgentID:   r.Header.Get("X-OpenViking-Agent"),
                },
            }, nil
        }
    }
    return nil, fmt.Errorf("dev mode only accepts localhost connections")
}

// API_KEY mode: resolve key via Admin service (with Redis cache)
func (m *AuthMiddleware) resolveAPIKeyMode(r *http.Request) (*viking.RequestContext, error) {
    apiKey := r.Header.Get("X-Api-Key")
    if apiKey == "" {
        // Try Authorization: Bearer
        bearer := r.Header.Get("Authorization")
        if strings.HasPrefix(bearer, "Bearer ") {
            apiKey = strings.TrimPrefix(bearer, "Bearer ")
        }
    }
    if apiKey == "" {
        return nil, fmt.Errorf("missing API key")
    }
    
    // Resolve via pkg/auth/CachedKeyResolver (Redis TTL=5min)
    resolved, err := m.keyResolver.Resolve(r.Context(), apiKey)
    if err != nil {
        return nil, fmt.Errorf("invalid API key")
    }
    
    return &viking.RequestContext{
        Role:     resolved.Role,
        APIKeyID: resolved.KeyID,
        User: viking.UserIdentifier{
            AccountID: resolved.AccountID,
            UserID:    resolved.UserID,
            AgentID:   r.Header.Get("X-OpenViking-Agent"),  // BOT can pass agent
        },
    }, nil
}

// TRUSTED mode: trust headers directly (for internal service mesh with mTLS)
func (m *AuthMiddleware) resolveTrustedMode(r *http.Request) (*viking.RequestContext, error) {
    accountID := r.Header.Get("X-OpenViking-Account")
    if accountID == "" {
        return nil, fmt.Errorf("missing X-OpenViking-Account header in trusted mode")
    }
    role := parseRoleHeader(r.Header.Get("X-OpenViking-Role"))
    return &viking.RequestContext{
        Role: role,
        User: viking.UserIdentifier{
            AccountID: accountID,
            UserID:    r.Header.Get("X-OpenViking-User"),
            AgentID:   r.Header.Get("X-OpenViking-Agent"),
        },
    }, nil
}
```

---

## 5. RBAC Enforcement

```go
// internal/usecase/check_namespace.go

type NamespaceChecker struct{}

// CheckAccess validates if the request context can access the target URI
func (nc *NamespaceChecker) CheckAccess(ctx context.Context, targetURI string) error {
    rc, ok := viking.FromContext(ctx)
    if !ok {
        return &viking.OpenVikingError{Code: viking.ErrUnauthenticated}
    }
    if !viking.IsAccessible(targetURI, rc) {
        return &viking.OpenVikingError{
            Code:    viking.ErrPermissionDenied,
            Message: fmt.Sprintf("role %v cannot access %s", rc.Role, targetURI),
        }
    }
    return nil
}

// CheckRole validates minimum required role for admin endpoints
func CheckMinRole(ctx context.Context, minRole viking.Role) error {
    rc, ok := viking.FromContext(ctx)
    if !ok || rc.Role < minRole {
        return &viking.OpenVikingError{Code: viking.ErrPermissionDenied}
    }
    return nil
}

// Usage in admin_handler.go:
// GET /api/v1/admin/accounts → requires RoleRoot
// GET /api/v1/admin/users    → requires RoleAdmin
// GET /api/v1/admin/api-keys → requires RoleAdmin (only own account)
```

---

## 6. MCP Server — 9 Tools Implementation

### 6.1 MCP Streamable HTTP Transport

```go
// internal/adapter/mcp/server.go

// MCP Streamable HTTP protocol (spec 2025-03-26)
// Single endpoint: POST /mcp
// Content-Type: application/json or text/event-stream (for streaming results)

type MCPServer struct {
    tools    map[string]ToolHandler
    authMW   func(http.Handler) http.Handler
}

func (s *MCPServer) Handler() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("/", s.authMW(http.HandlerFunc(s.handleMCP)).ServeHTTP)
    return mux
}

func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
    var req MCPRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    handler, exists := s.tools[req.Method]
    if !exists {
        writeMCPError(w, -32601, "Method not found: "+req.Method)
        return
    }
    
    result, err := handler.Handle(r.Context(), req.Params)
    if err != nil {
        writeMCPError(w, mapErrorCode(err), err.Error())
        return
    }
    
    writeMCPResult(w, req.ID, result)
}

// MCP tool registration:
func NewMCPServer(deps *Dependencies, authMW func(http.Handler) http.Handler) *MCPServer {
    return &MCPServer{
        authMW: authMW,
        tools: map[string]ToolHandler{
            "search":       NewSearchTool(deps.Search, deps.Session),
            "read":         NewReadTool(deps.FS),
            "list":         NewListTool(deps.FS),
            "store":        NewStoreTool(deps.Session),
            "add_resource": NewAddResourceTool(deps.Resource),
            "grep":         NewGrepTool(deps.FS),
            "glob":         NewGlobTool(deps.FS),
            "forget":       NewForgetTool(deps.FS, deps.Search),
            "health":       NewHealthTool(deps.Admin),
        },
    }
}
```

### 6.2 MCP Tool Definitions

```go
// internal/adapter/mcp/search_tool.go
// Tool: "search"
// Params: {"query": str, "uri"?: str, "type"?: "MEMORY|RESOURCE|SKILL|SESSION", "limit"?: int}
// Action: searchClient.Search() or searchClient.Find()
// Returns: [{uri, abstract, score, context_type}]

// internal/adapter/mcp/read_tool.go
// Tool: "read"
// Params: {"uri": str, "level"?: 0|1|2}
// Action: fsClient.Read(uri, level)
// Returns: {"content": str, "uri": str}

// internal/adapter/mcp/list_tool.go
// Tool: "list"
// Params: {"uri": str, "format"?: "tree|ls", "depth"?: int}
// Action: fsClient.Ls(uri) or fsClient.Tree(uri, depth)
// Returns: [{uri, name, is_directory, abstract}]

// internal/adapter/mcp/store_tool.go
// Tool: "store"
// Params: {"messages": [{role, content}], "session_id"?: str}
// Action: sessionClient.AddMessages() [auto-extracts memories]
// Returns: {"session_id": str, "tokens_pending": int}

// internal/adapter/mcp/add_resource_tool.go
// Tool: "add_resource"
// Params: {"url": str, "name": str, "watch"?: bool}
// Action: resourceClient.AddResource()
// Returns: {"task_id": str, "viking_uri": str}

// internal/adapter/mcp/grep_tool.go
// Tool: "grep"
// Params: {"pattern": str, "uri": str}
// Action: fsClient.Grep(pattern, uri)
// Returns: [{uri, line, content}]

// internal/adapter/mcp/glob_tool.go
// Tool: "glob"
// Params: {"pattern": str, "uri": str}
// Action: fsClient.Glob(pattern, uri)
// Returns: [uri strings]

// internal/adapter/mcp/forget_tool.go
// Tool: "forget"
// Params: {"uri": str}
// Action: fsClient.Rm(uri) + searchClient.RemoveContent(uri)
// Returns: {"deleted": true}

// internal/adapter/mcp/health_tool.go
// Tool: "health"
// Params: {}
// Action: adminClient.GetAggregatedHealth()
// Returns: {"status": "serving|degraded", "services": {}}

// MCP Manifest (returned on GET /mcp or tools/list call):
// {
//   "protocol_version": "2025-03-26",
//   "server_info": {"name": "openviking", "version": "1.0.0"},
//   "capabilities": {"tools": {"listChanged": false}},
//   "tools": [
//     {"name": "search", "description": "...", "inputSchema": {...}},
//     ...9 tools total...
//   ]
// }
```

---

## 7. WebDAV Proxy

```go
// internal/adapter/webdav/handler.go

import "golang.org/x/net/webdav"

// WebDAV handler that translates WebDAV operations to gRPC FS calls
type WebDAVHandler struct {
    fsClient port.FSClient
    authMW   func(http.Handler) http.Handler
}

// Implement webdav.FileSystem interface using FS gRPC client
type vikingWebDAVFS struct {
    fsClient port.FSClient
    account  string
}

func (fs *vikingWebDAVFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
    uri := webdavPathToVikingURI(name, fs.account)
    return fs.fsClient.Mkdir(ctx, uri, false)
}

func (fs *vikingWebDAVFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
    uri := webdavPathToVikingURI(name, fs.account)
    if flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0 {
        return &vikingWebDAVFile{fsClient: fs.fsClient, uri: uri, writable: true}, nil
    }
    // Read mode: fetch content
    content, err := fs.fsClient.Read(ctx, uri, 2)  // L2 = full content
    return &vikingWebDAVFile{content: content, uri: uri}, err
}

func (fs *vikingWebDAVFS) RemoveAll(ctx context.Context, name string) error {
    uri := webdavPathToVikingURI(name, fs.account)
    return fs.fsClient.Rm(ctx, uri, true)
}

func (fs *vikingWebDAVFS) Rename(ctx context.Context, oldName, newName string) error {
    oldURI := webdavPathToVikingURI(oldName, fs.account)
    newURI := webdavPathToVikingURI(newName, fs.account)
    return fs.fsClient.Mv(ctx, oldURI, newURI)
}

func (fs *vikingWebDAVFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
    uri := webdavPathToVikingURI(name, fs.account)
    return fs.fsClient.Stat(ctx, uri)
}

// URI mapping: /webdav/user/alice/memories/ → viking://user/{account}/alice/memories/
func webdavPathToVikingURI(webdavPath, accountID string) string {
    trimmed := strings.TrimPrefix(webdavPath, "/")
    return "viking://" + trimmed
}

func (h *WebDAVHandler) Handler() http.Handler {
    return h.authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rc, _ := viking.FromContext(r.Context())
        
        handler := &webdav.Handler{
            FileSystem: &vikingWebDAVFS{
                fsClient: h.fsClient,
                account:  rc.User.AccountID,
            },
            LockSystem: webdav.NewMemLS(),  // In-memory WebDAV locks
        }
        handler.ServeHTTP(w, r)
    }))
}
```

---

## 8. Circuit Breaker Integration

```go
// internal/adapter/client/fs_client.go

type FSClient struct {
    conn    *grpc.ClientConn
    client  fsv1.FileSystemServiceClient
    breaker *resilience.CircuitBreaker
}

func (c *FSClient) Read(ctx context.Context, uri string, level int) ([]byte, error) {
    result, err := c.breaker.Execute(func() (interface{}, error) {
        resp, err := c.client.Read(ctx, &fsv1.ReadRequest{
            Uri:   uri,
            Level: int32(level),
        })
        if err != nil {
            return nil, err
        }
        return resp.Content, nil
    })
    if err != nil {
        if errors.Is(err, gobreaker.ErrOpenState) {
            return nil, &viking.OpenVikingError{
                Code:    viking.ErrInternal,
                Message: "filesystem service unavailable (circuit open)",
            }
        }
        return nil, mapGRPCError(err)
    }
    return result.([]byte), nil
}

// Each downstream has its own circuit breaker:
// FS:       maxRequests=3, interval=10s, timeout=30s
// Search:   maxRequests=3, interval=10s, timeout=30s
// Session:  maxRequests=3, interval=10s, timeout=60s (commit can be slow)
// Resource: maxRequests=1, interval=30s, timeout=120s (ingestion can be slow)
// Admin:    maxRequests=5, interval=5s,  timeout=10s (critical for auth)
```

---

## 9. Sample Route Handlers

```go
// internal/adapter/http/filesystem_handler.go

type FilesystemHandler struct {
    fsClient port.FSClient
    checker  *usecase.NamespaceChecker
}

// POST /api/v1/filesystem/ls
func (h *FilesystemHandler) ListDirectory(w http.ResponseWriter, r *http.Request) {
    var req struct {
        URI string `json:"uri"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httpError(w, http.StatusBadRequest, "invalid request")
        return
    }
    
    if err := h.checker.CheckAccess(r.Context(), req.URI); err != nil {
        httpError(w, http.StatusForbidden, err.Error())
        return
    }
    
    entries, err := h.fsClient.Ls(r.Context(), req.URI)
    if err != nil {
        handleGRPCError(w, err)
        return
    }
    
    writeJSON(w, http.StatusOK, map[string]any{
        "uri":     req.URI,
        "entries": entries,
        "count":   len(entries),
    })
}

// internal/adapter/http/search_handler.go
// POST /api/v1/search/find
func (h *SearchHandler) Find(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Query       string  `json:"query"`
        ContextType *string `json:"type,omitempty"`
        Limit       int     `json:"limit"`
        Threshold   float64 `json:"threshold"`
        Reranker    bool    `json:"reranker_enabled"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    if req.Limit == 0 { req.Limit = 10 }
    
    rc, _ := viking.FromContext(r.Context())
    
    results, err := h.searchClient.Find(r.Context(), &searchv1.FindRequest{
        Query:          req.Query,
        AccountId:      rc.User.AccountID,
        UserId:         rc.User.UserID,
        ContextType:    strPtr(req.ContextType),
        Limit:          int32(req.Limit),
        Threshold:      req.Threshold,
        RerankerEnabled: req.Reranker,
    })
    if err != nil {
        handleGRPCError(w, err)
        return
    }
    
    writeJSON(w, http.StatusOK, results)
}
```

---

## 10. Claude Code MCP Integration

```json
// ~/.claude/settings.json (User configuration)
{
  "mcpServers": {
    "openviking": {
      "type": "http",
      "url": "http://localhost:8082/mcp",
      "headers": {
        "X-OpenViking-Account": "my-account",
        "X-OpenViking-User": "alice"
      }
    }
  }
}
```

```go
// MCP Tool Schema (JSON Schema for each tool):

// search tool:
{
    "name": "search",
    "description": "Search through memories, resources, and skills using semantic similarity",
    "inputSchema": {
        "type": "object",
        "properties": {
            "query": {"type": "string", "description": "Natural language search query"},
            "uri": {"type": "string", "description": "Restrict search to this URI (optional)"},
            "type": {"type": "string", "enum": ["MEMORY", "RESOURCE", "SKILL", "SESSION"]},
            "limit": {"type": "integer", "default": 10, "maximum": 50}
        },
        "required": ["query"]
    }
}

// read tool:
{
    "name": "read",
    "description": "Read file content from VikingFS at L0/L1/L2 detail level",
    "inputSchema": {
        "type": "object",
        "properties": {
            "uri": {"type": "string", "description": "Viking URI (viking://...)"},
            "level": {"type": "integer", "enum": [0, 1, 2], "default": 2,
                "description": "0=abstract(~100 tokens), 1=overview(~2K tokens), 2=full"}
        },
        "required": ["uri"]
    }
}
```

---

## 11. Error Mapping

```go
// internal/adapter/http/errors.go

func handleGRPCError(w http.ResponseWriter, err error) {
    st, ok := status.FromError(err)
    if !ok {
        // Check circuit breaker error
        if errors.Is(err, gobreaker.ErrOpenState) {
            httpError(w, http.StatusServiceUnavailable, "upstream service unavailable")
            return
        }
        httpError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    switch st.Code() {
    case codes.NotFound:
        httpError(w, http.StatusNotFound, st.Message())
    case codes.AlreadyExists:
        httpError(w, http.StatusConflict, st.Message())
    case codes.PermissionDenied:
        httpError(w, http.StatusForbidden, st.Message())
    case codes.Unauthenticated:
        httpError(w, http.StatusUnauthorized, st.Message())
    case codes.InvalidArgument:
        httpError(w, http.StatusBadRequest, st.Message())
    case codes.ResourceExhausted:
        httpError(w, http.StatusTooManyRequests, st.Message())
    case codes.Unavailable:
        httpError(w, http.StatusServiceUnavailable, st.Message())
    default:
        httpError(w, http.StatusInternalServerError, st.Message())
    }
}

func httpError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "error":   http.StatusText(status),
        "message": msg,
    })
}
```

---

## 12. Configuration

```yaml
gateway:
  http:
    port: 8080
    max_request_size: "10MB"
    timeout: 60s
    cors:
      allowed_origins: ["*"]   # Restrict in production
      allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
  
  grpc:
    port: 8081
  
  mcp:
    port: 8082
  
  health:
    port: 8083
  
  auth:
    mode: "dev"        # dev | api_key | trusted
    root_api_key: ""   # Required in api_key mode
  
  rate_limit:
    enabled: true
    redis_url: "redis://redis:6379/0"
    window: 60s
    requests_per_window: 1000
    burst_multiplier: 2  # Allow 2000 requests in first 10 seconds
  
  circuit_breaker:
    max_requests: 3
    interval: 10s
    timeout: 30s
  
  upstream:
    fs:       "openviking-fs:9011"
    search:   "openviking-search:9012"
    session:  "openviking-session:9013"
    resource: "openviking-resource:9014"
    admin:    "openviking-admin:9030"
    
  telemetry:
    service_name: "openviking-gateway"
    otel_endpoint: "otel-collector:4317"
```

---

## 13. Testing Strategy

### Unit Tests
- `TestAuthMiddleware_DEVMode_AcceptsLocalhost` — 127.0.0.1 → ROOT role, no error
- `TestAuthMiddleware_DEVMode_RejectsExternal` — 1.2.3.4 → 401
- `TestAuthMiddleware_APIKeyMode_ResolvesKey` — valid key → RequestContext with role
- `TestAuthMiddleware_APIKeyMode_RejectsMissingKey` — no key → 401
- `TestAuthMiddleware_TrustedMode_ReadsHeaders` — X-OpenViking-Account → AccountID in ctx
- `TestRBACCheck_AdminRoute_RequiresAdmin` — USER role → 403
- `TestCircuitBreaker_OpenState_Returns503` — breaker open → 503, not timeout
- `TestWebDAVPathMapping` — /webdav/user/alice/ → viking://user/alice/
- `TestMCPSearchTool_ForwardsToSearch` — params validated, search client called
- `TestMCPForgeTool_CallsFSAndSearch` — fsClient.Rm + searchClient.RemoveContent called

### Integration Tests
- `TestGatewayE2E_SearchFlow` — real gateway → real search service → results returned
- `TestMCPE2E_ClaudeCompatible` — MCP manifest returns 9 tools; tool call returns valid JSON-RPC
- `TestWebDAVE2E_MountAndWrite` — WebDAV PUT → file in VikingFS
- `TestRateLimitE2E` — >1000 requests → 429 returned
- `TestAuthE2E_KeyRevoked` — revoke key → next request with same key → 401

---

## 14. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| WebDAV lock conflicts với PathLock | Trung bình | WebDAV MemLS chỉ lock WebDAV operations; PathLock trong FS service lock actual writes |
| MCP spec thay đổi (draft protocol) | Thấp | Pin to spec 2025-03-26; version in manifest header |
| CORS policy quá rộng trong production | Trung bình | Require explicit CORS config; default chặt hơn |
| Circuit breaker timeout = 30s còn chậm | Thấp | Circuit OPEN → fail immediately; timeout chỉ áp dụng trước circuit mở |
| WebDAV large file PUT → timeout | Trung bình | Streaming read/write; timeout=300s cho WebDAV endpoint riêng |
| Rate limit bypass qua header spoofing | Trung bình | Rate limit key = API key ID (not IP) khi api_key mode; IP khi dev mode |
