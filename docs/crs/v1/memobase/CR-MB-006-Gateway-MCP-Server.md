# Change Request: CR-MB-006 — Gateway Service & MCP Server (Memobase Protocol)

**CR ID:** CR-MB-006  
**Component:** `gateway` [EXTEND] | MCP Server [EXTEND]  
**Priority:** High  
**Status:** In Progress
**Reference:** memobase PRD §5.7 (F-7), SRS §5, specs/services/01-gateway.md  
**Maps to Python:** FastAPI server + `mcp_server/`

---

## 1. Mô tả

Mở rộng **memobase-gateway** để:
1. **Full Memobase REST API** — routing tất cả memobase endpoints tới các microservices.
2. **MCP Server** — 3 memobase tools cho Claude Desktop, Cursor, Windsurf, n8n.
3. **Multi-Tenant Auth** — JWT và project token (`sk-proj-*`) authentication.
4. **Rate Limiting** — Redis sliding window per-project per-endpoint.
5. **CORS + Response Timing** — `X-Process-Time` header, configurable CORS.
6. **Custom Error Format** — `{data, errno, errmsg}` consistent response format.

---

## 2. Vấn đề hiện tại

`gateway` hiện tại trong VNP Memory:
- ✅ Có cơ bản REST routing và JWT auth.
- ❌ Thiếu **memobase REST endpoints** đầy đủ (blob, buffer, profile, event, context, project).
- ❌ MCP server thiếu **memobase-specific tools** (save_memory, get_user_profiles, search_memories).
- ❌ Không có **`sk-proj-*` project token** authentication.
- ❌ Không có **`X-Process-Time` header** trên mọi response.
- ❌ Không có **custom error format** `{data, errno, errmsg}` (memobase standard).
- ❌ Không có **`USE_CORS` + `API_HOSTS`** config cho CORS whitelist.
- ❌ Không có **project context propagation** (project_id in gRPC metadata).
- ❌ Không có **MCP SSE transport** support (chỉ có stdio).

---

## 3. Thay đổi đề xuất

### 3.1. [EXTEND] Gateway REST Router

```go
// gateway/internal/adapter/http/memobase/

// Route Group: /api/v1/

// Chore
GET    /api/v1/healthcheck                              → health check (no auth)
GET    /api/v1/admin/status_check                       → admin.GetSystemStatus (root auth)

// User Management → memobase-admin
POST   /api/v1/users                                    → admin.CreateUser
GET    /api/v1/users/{user_id}                          → admin.GetUser
PUT    /api/v1/users/{user_id}                          → admin.UpdateUser
DELETE /api/v1/users/{user_id}                          → admin.DeleteUser

// Blob → memobase-ingestion
POST   /api/v1/blobs/insert/{user_id}                   → ingestion.InsertBlob
GET    /api/v1/blobs/{user_id}/{blob_id}                → ingestion.GetBlob
DELETE /api/v1/blobs/{user_id}/{blob_id}                → ingestion.DeleteBlob

// Buffer → memobase-ingestion
POST   /api/v1/users/buffer/{user_id}/{buffer_type}     → ingestion.FlushBuffer
GET    /api/v1/users/buffer/capacity/{user_id}/{buffer_type} → ingestion.GetBufferCapacity

// Profile → memobase-context
GET    /api/v1/users/profile/{user_id}                  → context.GetProfiles
POST   /api/v1/users/profile/{user_id}                  → context.AddProfile
PUT    /api/v1/users/profile/{user_id}/{profile_id}     → context.UpdateProfile
DELETE /api/v1/users/profile/{user_id}/{profile_id}     → context.DeleteProfile

// Context → memobase-context
GET    /api/v1/users/context/{user_id}                  → context.GetContext

// Event → memobase-event
GET    /api/v1/users/event/{user_id}                    → event.GetEvents
PUT    /api/v1/users/event/{user_id}/{event_id}         → event.UpdateEvent
DELETE /api/v1/users/event/{user_id}/{event_id}         → event.DeleteEvent
GET    /api/v1/users/event/search/{user_id}             → event.SearchEvents
GET    /api/v1/users/event_gist/search/{user_id}        → event.SearchEventGists
GET    /api/v1/users/event_tags/search/{user_id}        → event.FilterEventsByTags

// Project → memobase-admin
POST   /api/v1/project/profile_config                   → admin.UpdateProfileConfig
GET    /api/v1/project/profile_config                   → admin.GetProfileConfig
GET    /api/v1/project/billing                          → admin.GetBilling
GET    /api/v1/project/users                            → admin.ListProjectUsers
GET    /api/v1/project/usage                            → admin.GetUsage

// Roleplay (optional/future)
POST   /api/v1/users/roleplay/proactive/{user_id}       → context.InferProactiveTopics
```

### 3.2. MCP Server — 3 Memobase Tools

```go
// gateway/internal/adapter/mcp/tools/memobase/

// Tool 1: save_memory — insert chat blob and flush
var saveMemoryTool = mcp.Tool{
    Name: "save_memory",
    Description: "Save a conversation or piece of information to the user's long-term memory. The system will extract structured profile information and timeline events from the content.",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "messages": {
                Type: "array",
                Description: "List of chat messages in OpenAI format",
                Items: &mcp.Property{
                    Type: "object",
                    Properties: map[string]mcp.Property{
                        "role":    {Type: "string", Enum: []string{"user", "assistant", "system"}},
                        "content": {Type: "string"},
                    },
                },
            },
            "user_id": {
                Type:        "string",
                Description: "The user identifier for this memory",
            },
            "flush": {
                Type:        "boolean",
                Description: "Whether to immediately process the memory (sync flush). Default: false",
                Default:     false,
            },
        },
        Required: []string{"messages", "user_id"},
    },
    Handler: handleSaveMemory,
    // Implementation:
    // 1. POST /api/v1/blobs/insert/{user_id} (ChatBlob)
    // 2. If flush: POST /api/v1/users/buffer/{user_id}/chat (sync)
    // 3. Return: {blob_id, flushed: bool}
}

// Tool 2: get_user_profiles — retrieve structured user profiles
var getUserProfilesTool = mcp.Tool{
    Name: "get_user_profiles",
    Description: "Retrieve the structured user profile information extracted from previous conversations. Returns topic-organized facts about the user.",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "user_id": {
                Type:        "string",
                Description: "The user identifier",
            },
            "prefer_topics": {
                Type:        "array",
                Items:       &mcp.Property{Type: "string"},
                Description: "List of topics to prioritize in the response",
            },
            "max_token_size": {
                Type:        "integer",
                Description: "Maximum token budget for profiles",
                Default:     500,
            },
        },
        Required: []string{"user_id"},
    },
    Handler: handleGetUserProfiles,
    // Implementation: GET /api/v1/users/profile/{user_id}?prefer_topics=...
}

// Tool 3: search_memories — semantic search on event timeline
var searchMemoriesTool = mcp.Tool{
    Name: "search_memories",
    Description: "Search through the user's event timeline using semantic similarity. Finds relevant past events and interactions based on the query.",
    InputSchema: mcp.Schema{
        Properties: map[string]mcp.Property{
            "user_id": {
                Type:        "string",
                Description: "The user identifier",
            },
            "query": {
                Type:        "string",
                Description: "The search query to find relevant memories",
            },
            "topk": {
                Type:        "integer",
                Description: "Maximum number of results to return",
                Default:     10,
            },
            "time_range_in_days": {
                Type:        "integer",
                Description: "Only search events within this many days",
                Default:     21,
            },
        },
        Required: []string{"user_id", "query"},
    },
    Handler: handleSearchMemories,
    // Implementation: GET /api/v1/users/event/search/{user_id}?query=...&topk=...
}
```

### 3.3. MCP Transport: SSE + Stdio

```go
// gateway/internal/adapter/mcp/server.go

type MCPServerConfig struct {
    // SSE transport (for web-based AI tools)
    SSEEnabled bool
    SSEPath    string      // default: "/mcp/sse"
    SSEPort    int         // default: 8082

    // Stdio transport (for CLI tools like Claude Desktop)
    StdioEnabled bool
}

// SSE endpoint:
// GET /mcp/sse → SSE stream for MCP protocol
// Compatible: Claude Desktop (remote URL), Cursor, Windsurf

// Stdio: run as subprocess
// Compatible: Claude Desktop (local), n8n
```

### 3.4. Auth Middleware

```go
// pkg/middleware/auth/bearer.go

// Supported auth formats:
// 1. Root token: Authorization: Bearer {ACCESS_TOKEN}
//    → grants full access, no project scoping
//
// 2. Project token: Authorization: Bearer sk-proj-{encoded}
//    → validate via admin.ValidateProjectToken gRPC
//    → on success: extract project_id, check status != "suspended"
//    → propagate project_id via gRPC metadata: "x-project-id"
//    → on suspended: 403 Forbidden

// Endpoints bypassing auth:
// GET /api/v1/healthcheck  (public)

// Error responses:
// Missing token: 401 {"data": null, "errno": 401, "errmsg": "Unauthorized"}
// Suspended:     403 {"data": null, "errno": 403, "errmsg": "Project suspended"}
```

### 3.5. Response Format (Memobase Standard)

```go
// All memobase responses use unified format:
type MemobaseResponse struct {
    Data   interface{} `json:"data"`         // response payload or null
    Errno  int         `json:"errno"`         // 0 = success
    ErrMsg string      `json:"errmsg"`        // empty on success
}

// Error codes:
// 0   SUCCESS         200
// 400 BAD_REQUEST     400
// 401 UNAUTHORIZED    401
// 403 FORBIDDEN       403
// 404 NOT_FOUND       404
// 500 INTERNAL_ERROR  500
// 501 NOT_IMPLEMENTED 501 (e.g., embedding disabled)
// 520 PARSE_ERROR     520

// Success example:
// { "data": { "id": "uuid" }, "errno": 0, "errmsg": "" }

// Error example:
// { "data": null, "errno": 400, "errmsg": "invalid blob format" }
```

### 3.6. Response Timing Header

```go
// middleware/timing.go
// X-Process-Time: response time in milliseconds (all responses)

func TimingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        elapsed := time.Since(start).Milliseconds()
        w.Header().Set("X-Process-Time", fmt.Sprintf("%dms", elapsed))
    })
}
```

### 3.7. CORS Configuration

```go
// pkg/middleware/cors.go
// Controlled via USE_CORS env var + API_HOSTS whitelist

type CORSConfig struct {
    Enabled      bool
    AllowedHosts []string  // API_HOSTS env var (comma-separated)
    Methods      []string  // GET, POST, PUT, DELETE, OPTIONS
    Headers      []string  // Content-Type, Authorization, X-Request-ID
}
```

### 3.8. Rate Limiting (Redis Sliding Window)

```go
// pkg/middleware/ratelimit/

// Limits per project (configurable):
// POST /api/v1/blobs/insert/*:        100 req/min
// POST /api/v1/users/buffer/*:         20 req/min
// GET  /api/v1/users/context/*:       300 req/min
// GET  /api/v1/users/event/search/*:  200 req/min

// Response when rate limited:
// HTTP 429
// Retry-After: <seconds until window resets>
// { "data": null, "errno": 429, "errmsg": "Rate limit exceeded" }
```

### 3.9. Request ID Tracking

```go
// middleware/request_id.go
// X-Request-ID: propagate or generate

// If client provides X-Request-ID header: use it
// Otherwise: generate UUID v4
// Propagate to all gRPC calls as metadata
// Include in structured logs
```

### 3.10. Health Check Endpoint

```go
// GET /api/v1/healthcheck → aggregate health (no auth required)
// Response:
// {
//   "data": {
//     "status": "healthy" | "degraded" | "unhealthy",
//     "services": {
//       "ingestion": { "status": "healthy", "latency_ms": 2 },
//       "engine":    { "status": "healthy", "latency_ms": 1 },
//       "context":   { "status": "healthy", "latency_ms": 1 },
//       "event":     { "status": "healthy", "latency_ms": 2 },
//       "admin":     { "status": "healthy", "latency_ms": 1 },
//       "postgres":  { "status": "healthy", "latency_ms": 5 },
//       "redis":     { "status": "healthy", "latency_ms": 1 },
//       "nats":      { "status": "healthy", "latency_ms": 1 }
//     }
//   },
//   "errno": 0
// }
```

---

## 4. Configuration

```yaml
gateway:
  http:
    port: 8080
  grpc:
    port: 8081
  health:
    port: 8083

auth:
  root_token: "${ACCESS_TOKEN}"
  project_token_prefix: "sk-proj-"

cors:
  enabled: "${USE_CORS}"      # false by default
  allowed_hosts: "${API_HOSTS}"  # comma-separated

mcp:
  enabled: true
  sse_path: "/mcp/sse"
  sse_port: 8082
  stdio_enabled: true
  tools:
    - save_memory
    - get_user_profiles
    - search_memories

services:
  ingestion: { address: "memobase-ingestion:9041", timeout: 30s }
  engine:    { address: "memobase-engine:9042", timeout: 120s }  # LLM calls
  context:   { address: "memobase-context:9043", timeout: 5s }
  event:     { address: "memobase-event:9044", timeout: 10s }
  admin:     { address: "memobase-admin:9045", timeout: 5s }
```

---

## 5. Acceptance Criteria

- [ ] `GET /api/v1/healthcheck` (no auth) → 200 OK với `{data: {status: "healthy"}}`.
- [ ] `POST /api/v1/blobs/insert/{user_id}` valid token → 200 `{data: {id: blob_id}, errno: 0}`.
- [ ] `POST /api/v1/blobs/insert/{user_id}` no token → 401 `{data: null, errno: 401}`.
- [ ] `sk-proj-*` token với suspended project → 403 `{data: null, errno: 403, errmsg: "Project suspended"}`.
- [ ] Mọi response đều có `X-Process-Time` header.
- [ ] MCP tool `save_memory` từ Claude Desktop → blob inserted, phản hồi trong ≤ 2s.
- [ ] MCP tool `get_user_profiles` → trả về structured profiles theo topic/sub_topic.
- [ ] MCP tool `search_memories` query "project deadline" → trả về relevant events.
- [ ] MCP SSE endpoint `GET /mcp/sse` → valid SSE stream, compatible với Claude Desktop remote URL.
- [ ] CORS: `USE_CORS=true, API_HOSTS=example.com` → `POST /api/v1/blobs/insert` từ example.com → CORS headers present.
- [ ] Rate limit: `POST /api/v1/blobs/insert` 101 requests/min → 102nd → 429 với `Retry-After`.
- [ ] `X-Request-ID` header propagated to all gRPC calls và visible in structured logs.
