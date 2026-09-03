# Change Request: CR-OV-001 — Unified Gateway Service (API, MCP, WebDAV)

**CR ID:** CR-OV-001  
**Component:** `services/openviking-gateway` [NEW SERVICE]  
**Priority:** Critical  
**Status:** Implemented
**Reference:** OpenViking PRD §4.8 §5.2, SRS §3.1, specs/services/01-gateway.md  
**Maps from Python:** `server/app.py`, `server/mcp_endpoint.py`, `server/routers/webdav.py`, `server/auth.py`

---

## 1. Mô tả

Xây dựng **openviking-gateway** — entry point thống nhất cho toàn bộ hệ thống bằng Golang Clean Architecture. Service này là lớp trình bày (Presentation Layer) duy nhất, **không chứa business logic**:

1. **REST API Router**: Expose 17 route groups (filesystem, content, search, sessions, resources, relations, admin, observer, privacy, tasks, system, debug, bot, pack, maintenance, stats, MCP) và proxy tới các gRPC services.
2. **MCP Server (9 tools)**: Implement đủ 9 tools qua Streamable HTTP tại `/mcp` (search, read, list, store, add_resource, grep, glob, forget, health).
3. **WebDAV Proxy**: Hỗ trợ chuẩn WebDAV tại `/webdav` — mount VikingFS như ổ đĩa mạng (PROPFIND, MKCOL, PUT, DELETE → gRPC calls tới openviking-fs).
4. **3-Mode Auth**: DEV (no-auth, localhost), API_KEY (root + per-user keys, RBAC), TRUSTED (gateway-backed headers).
5. **RBAC**: ROOT → ADMIN → USER resolution với Identity headers (`X-OpenViking-Account`, `X-OpenViking-User`, `X-OpenViking-Agent`).
6. **Rate Limiting**: Redis sliding window, per-tenant per-endpoint.
7. **Circuit Breaker + Retry**: sony/gobreaker per downstream, exponential backoff.

---

## 2. Vấn đề hiện tại

- Python monolith gắn liền Gateway với business logic → khó scale riêng lẻ.
- Chưa có WebDAV support cho phép mount VikingFS như ổ đĩa (đang cần cho FUSE integration).
- MCP server hiện chưa đủ 9 tools với đầy đủ parameters.
- Thiếu BOT role trong RBAC — VikingBot cần role riêng để access có giới hạn.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/openviking-gateway/`

```
Ports:
  REST HTTP:    8080
  gRPC/gRPC-Web: 8081
  MCP SSE:      8082
  Health:       8083
```

### 3.2. Clean Architecture Layout

```
services/openviking-gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── identity.go        # RequestContext, UserIdentifier, Role
│   │   ├── auth_mode.go       # DEV | API_KEY | TRUSTED
│   │   └── errors.go
│   ├── usecase/
│   │   ├── resolve_identity.go # Identity resolution from auth headers
│   │   ├── check_namespace.go  # Namespace access validation
│   │   └── port/
│   │       └── output.go       # AdminClient (for API key resolution)
│   ├── adapter/
│   │   ├── http/               # 17 route group handlers (REST)
│   │   │   ├── filesystem_handler.go
│   │   │   ├── content_handler.go
│   │   │   ├── search_handler.go
│   │   │   ├── session_handler.go
│   │   │   ├── resource_handler.go
│   │   │   ├── relations_handler.go
│   │   │   ├── admin_handler.go
│   │   │   ├── observer_handler.go
│   │   │   ├── privacy_handler.go
│   │   │   ├── tasks_handler.go
│   │   │   ├── system_handler.go
│   │   │   ├── debug_handler.go
│   │   │   ├── bot_handler.go
│   │   │   ├── pack_handler.go
│   │   │   ├── maintenance_handler.go
│   │   │   └── stats_handler.go
│   │   ├── mcp/                # 9 MCP tools (Streamable HTTP)
│   │   │   ├── server.go
│   │   │   ├── search_tool.go
│   │   │   ├── read_tool.go
│   │   │   ├── list_tool.go
│   │   │   ├── store_tool.go
│   │   │   ├── add_resource_tool.go
│   │   │   ├── grep_tool.go
│   │   │   ├── glob_tool.go
│   │   │   ├── forget_tool.go
│   │   │   └── health_tool.go
│   │   ├── webdav/             # WebDAV proxy
│   │   │   └── handler.go      # PROPFIND/MKCOL/PUT/DELETE → gRPC FS
│   │   └── client/             # gRPC clients to downstream services
│   │       ├── fs_client.go
│   │       ├── search_client.go
│   │       ├── session_client.go
│   │       ├── resource_client.go
│   │       ├── crypto_client.go
│   │       └── admin_client.go
│   └── infra/
│       ├── config/config.go
│       ├── server/http.go
│       ├── middleware/         # HTTP middleware chain
│       └── wire/wire.go
```

### 3.3. REST Router — 17 Route Groups

```go
r.Route("/api/v1", func(r chi.Router) {
    r.Use(authMiddleware, tenantMiddleware, rateLimitMiddleware)
    r.Mount("/filesystem",     fsHandler)       // ls, tree, mkdir, rm
    r.Mount("/content",        contentHandler)  // read, write, mv, cp
    r.Mount("/search",         searchHandler)   // find, grep, glob
    r.Mount("/sessions",       sessionHandler)  // CRUD, commit
    r.Mount("/resources",      resourceHandler) // add, list, delete, refresh
    r.Mount("/relations",      relationsHandler)// get, add, remove
    r.Mount("/admin",          adminHandler)    // account, user, key mgmt
    r.Mount("/observer",       observerHandler) // retrieval stats, replay
    r.Mount("/privacy-configs",privacyHandler)  // user privacy config CRUD
    r.Mount("/tasks",          taskHandler)     // task status
    r.Mount("/system",         systemHandler)   // status, wait, debug
    r.Mount("/debug",          debugHandler)    // graph inspect, replay
    r.Mount("/bot",            botHandler)      // VikingBot integration
    r.Mount("/pack",           packHandler)     // context export
    r.Mount("/maintenance",    maintenanceHandler)
    r.Mount("/stats",          statsHandler)    // usage statistics
})
r.Mount("/mcp",     mcpHandler)    // 9 MCP tools (Streamable HTTP)
r.Mount("/webdav",  webdavHandler) // WebDAV protocol
r.Get("/metrics",   metricsHandler)
r.Get("/healthz",   healthHandler)
r.Get("/readyz",    readyHandler)
```

### 3.4. Authentication Middleware — 3 Modes

```go
// pkg/middleware/auth/auth.go
type AuthMode string
const (
    AuthModeDev     AuthMode = "dev"     // No auth, ROOT role, localhost only
    AuthModeAPIKey  AuthMode = "api_key" // Root key + per-user keys
    AuthModeTrusted AuthMode = "trusted" // Trust X-OpenViking-* headers
)

// Identity Resolution flow:
// 1. Check auth mode from config
// 2. DEV: auto ROOT, no validation
// 3. API_KEY:
//    a. Read X-Api-Key / Authorization: Bearer
//    b. gRPC → Admin.ResolveAPIKey → get AccountID, UserID, Role
//    c. Check namespace access (account/user scope)
// 4. TRUSTED: read X-OpenViking-Account/User/Agent headers directly
// 5. Set RequestContext in ctx with: AccountID, UserID, AgentID, Role
```

### 3.5. RBAC Roles

| Role | Scope | Permissions |
|------|-------|-------------|
| `ROOT` | Global | All operations, admin APIs, cross-tenant |
| `ADMIN` | Account-scoped | User management, key management within account |
| `USER` | User-scoped | Data CRUD within own `viking://user/{id}/` namespace |
| `BOT` | Agent-scoped | Access specific `viking://agent/{id}/` namespace |

### 3.6. MCP Server — 9 Tools (Streamable HTTP)

```go
// internal/adapter/mcp/server.go
// Transport: POST /mcp (Streamable HTTP per MCP spec 2025-03-26)
// Identity: ASGI-equivalent middleware extracts X-OpenViking-* headers → ctx

tools := []mcp.Tool{
    {Name: "search",       Handler: SearchTool},    // Semantic search memories/resources/skills
    {Name: "read",         Handler: ReadTool},      // Read full content from URI(s)
    {Name: "list",         Handler: ListTool},      // Directory listing
    {Name: "store",        Handler: StoreTool},     // Store messages → memory extraction
    {Name: "add_resource", Handler: AddResourceTool}, // Add HTTP/git resource
    {Name: "grep",         Handler: GrepTool},      // Regex pattern match in files
    {Name: "glob",         Handler: GlobTool},      // Filename glob matching
    {Name: "forget",       Handler: ForgetTool},    // Delete URI permanently
    {Name: "health",       Handler: HealthTool},    // Server health check
}
```

### 3.7. WebDAV Proxy

```go
// internal/adapter/webdav/handler.go
// Translate WebDAV methods → gRPC calls to openviking-fs

PROPFIND /webdav/user/alice/memories/ → FS.Ls(uri) + FS.Stat(uri)
MKCOL    /webdav/user/alice/notes/    → FS.Mkdir(uri)
PUT      /webdav/user/alice/notes/x.md → FS.Write(uri, content)
DELETE   /webdav/user/alice/notes/x.md → FS.Rm(uri)
GET      /webdav/user/alice/notes/x.md → FS.Read(uri)
```

### 3.8. Configuration

```yaml
gateway:
  http:
    port: 8080
    max_request_size: 10MB
    timeout: 60s
  grpc:
    port: 8081
  mcp:
    port: 8082
  auth:
    mode: "dev"         # dev | api_key | trusted
    root_api_key: ""    # required in api_key mode
  rate_limit:
    enabled: true
    redis_url: "redis://redis:6379"
    window: 60s
    requests_per_window: 1000
  upstream:
    fs:       "openviking-fs:9011"
    search:   "openviking-search:9012"
    session:  "openviking-session:9013"
    resource: "openviking-resource:9014"
    crypto:   "openviking-crypto:9015"
    admin:    "openviking-admin:9030"
```

---

## 4. NATS Events

| Event | Direction | Mô tả |
|-------|-----------|-------|
| Không publish events trực tiếp | — | Gateway chỉ là proxy, events do downstream services emit |

---

## 5. Acceptance Criteria

- [ ] `GET /api/v1/system/status` → HTTP 200 OK với server info.
- [ ] Kết nối Claude Desktop MCP tới `http://localhost:8082/mcp` → thấy đủ 9 tools trong tool list.
- [ ] Mode `api_key`: request không có key → 401; request với key hợp lệ → 200.
- [ ] Mode `trusted`: request với `X-OpenViking-Account: acct1` header → RequestContext có AccountID = "acct1".
- [ ] Mode `dev`: chỉ accept connections từ localhost (127.0.0.1/::1), reject từ external IP.
- [ ] `PROPFIND /webdav/user/alice/` → trả về WebDAV XML listing đúng format.
- [ ] Rate limit: > 1000 request/phút từ cùng tenant → 429 Too Many Requests.
- [ ] Circuit breaker: openviking-fs down → gateway trả về 503 Service Unavailable (không wait timeout 60s).
- [ ] MCP tool `search` với `{ "query": "test", "uri": "viking://user/alice/memories/" }` → forward đúng tới Search service.
- [ ] RBAC: USER role cố gắng access `/api/v1/admin/accounts` → 403 Forbidden.
