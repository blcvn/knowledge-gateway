# Solution: SOL-MB-006 — Gateway & MCP Server Extension

**CR:** [CR-MB-006](../CR-MB-006-Gateway-MCP-Server.md)  
**Wave:** 4 (Access Layer)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Mở rộng `gateway/` hiện có để thêm toàn bộ memobase REST API (30+ endpoints), 3 MCP tools memobase-specific, và các middleware cần thiết (timing header, custom error format, sk-proj-* auth, CORS, rate limiting).

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Thiếu memobase REST endpoints | Thêm route group `/api/v1/` vào gateway router |
| Thiếu sk-proj-* auth | `BearerAuthMiddleware` với dual-mode (root token + project token) |
| Không có X-Process-Time | `TimingMiddleware` bọc mọi request |
| Không có custom error format | `MemobaseResponseWriter` helper → `{data, errno, errmsg}` |
| Thiếu 3 MCP tools | Thêm `save_memory`, `get_user_profiles`, `search_memories` vào MCP server |
| Không có MCP SSE | Enable SSE transport trên `:8082` |
| Thiếu CORS config | `CORSMiddleware` từ `USE_CORS` + `API_HOSTS` env |
| Không có rate limiting | Redis sliding window per project per endpoint |

---

## 2. Gateway Structure Changes

```
gateway/
├── internal/
│   ├── adapter/
│   │   ├── handler/                       ← Hiện có: VNP Memory handlers
│   │   │   └── memobase/                  ← [NEW] Memobase HTTP handlers
│   │   │       ├── blob_handler.go        # InsertBlob, GetBlob, DeleteBlob
│   │   │       ├── buffer_handler.go      # FlushBuffer, GetBufferCapacity
│   │   │       ├── profile_handler.go     # GetProfiles, AddProfile, UpdateProfile, DeleteProfile
│   │   │       ├── context_handler.go     # GetContext
│   │   │       ├── event_handler.go       # GetEvents, UpdateEvent, DeleteEvent, SearchEvents, SearchGists, FilterByTags
│   │   │       ├── project_handler.go     # GetProfileConfig, UpdateProfileConfig, GetBilling, ListUsers, GetUsage
│   │   │       ├── admin_handler.go       # CreateUser, GetUser, UpdateUser, DeleteUser
│   │   │       ├── health_handler.go      # Aggregate health check
│   │   │       └── response.go            # MemobaseResponse{data, errno, errmsg}
│   │   ├── mcp/                           ← Hiện có: VNP Memory MCP tools
│   │   │   └── tools/
│   │   │       └── memobase/              ← [NEW] 3 Memobase MCP tools
│   │   │           ├── save_memory.go
│   │   │           ├── get_user_profiles.go
│   │   │           └── search_memories.go
│   │   └── client/                        ← [EXTEND] Thêm memobase service clients
│   │       ├── ingestion_client.go        # [NEW]
│   │       ├── context_client.go          # [NEW]
│   │       ├── event_client.go            # [NEW]
│   │       └── admin_client.go            # [NEW]
│   └── infra/
│       ├── middleware/                     ← [EXTEND]
│       │   ├── timing.go                  # [NEW] X-Process-Time
│       │   ├── cors.go                    # [NEW] USE_CORS + API_HOSTS
│       │   ├── bearer_auth.go             # [EXTEND] Add sk-proj-* validation
│       │   ├── ratelimit.go               # [NEW] Redis sliding window
│       │   └── request_id.go              # [NEW] X-Request-ID
│       └── server/
│           └── mcp_sse.go                 # [NEW] SSE transport for MCP
```

---

## 3. Route Registration

```go
// gateway/internal/infra/server/router.go

func (s *Server) registerMemobaseRoutes(mux *http.ServeMux) {
    h := &memobaseHandlers{
        ingestion: s.clients.Ingestion,
        context:   s.clients.Context,
        event:     s.clients.Event,
        admin:     s.clients.Admin,
    }

    // Middleware chain cho memobase routes:
    // Recovery → RequestID → Logger → Timing → CORS → Auth → RateLimit

    // Public endpoints (no auth)
    mux.Handle("GET /api/v1/healthcheck", h.health.Check)

    // Protected endpoints (sk-proj-* or root token)
    protected := chain(
        s.middleware.Recovery,
        s.middleware.RequestID,
        s.middleware.Logger,
        s.middleware.Timing,
        s.middleware.CORS,
        s.middleware.BearerAuth,
        s.middleware.RateLimit,
    )

    // User management → memobase-admin
    mux.Handle("POST /api/v1/users", protected(h.admin.CreateUser))
    mux.Handle("GET /api/v1/users/{user_id}", protected(h.admin.GetUser))
    mux.Handle("PUT /api/v1/users/{user_id}", protected(h.admin.UpdateUser))
    mux.Handle("DELETE /api/v1/users/{user_id}", protected(h.admin.DeleteUser))

    // Blob management → memobase-ingestion
    mux.Handle("POST /api/v1/blobs/insert/{user_id}", protected(h.blob.Insert))
    mux.Handle("GET /api/v1/blobs/{user_id}/{blob_id}", protected(h.blob.Get))
    mux.Handle("DELETE /api/v1/blobs/{user_id}/{blob_id}", protected(h.blob.Delete))

    // Buffer → memobase-ingestion
    mux.Handle("POST /api/v1/users/buffer/{user_id}/{buffer_type}", protected(h.buffer.Flush))
    mux.Handle("GET /api/v1/users/buffer/capacity/{user_id}/{buffer_type}", protected(h.buffer.GetCapacity))

    // Profile → memobase-context
    mux.Handle("GET /api/v1/users/profile/{user_id}", protected(h.profile.Get))
    mux.Handle("POST /api/v1/users/profile/{user_id}", protected(h.profile.Add))
    mux.Handle("PUT /api/v1/users/profile/{user_id}/{profile_id}", protected(h.profile.Update))
    mux.Handle("DELETE /api/v1/users/profile/{user_id}/{profile_id}", protected(h.profile.Delete))

    // Context → memobase-context
    mux.Handle("GET /api/v1/users/context/{user_id}", protected(h.ctx.Get))

    // Event → memobase-event
    mux.Handle("GET /api/v1/users/event/{user_id}", protected(h.event.List))
    mux.Handle("PUT /api/v1/users/event/{user_id}/{event_id}", protected(h.event.Update))
    mux.Handle("DELETE /api/v1/users/event/{user_id}/{event_id}", protected(h.event.Delete))
    mux.Handle("GET /api/v1/users/event/search/{user_id}", protected(h.event.Search))
    mux.Handle("GET /api/v1/users/event_gist/search/{user_id}", protected(h.event.SearchGists))
    mux.Handle("GET /api/v1/users/event_tags/search/{user_id}", protected(h.event.FilterByTags))

    // Project → memobase-admin
    mux.Handle("POST /api/v1/project/profile_config", protected(h.project.UpdateConfig))
    mux.Handle("GET /api/v1/project/profile_config", protected(h.project.GetConfig))
    mux.Handle("GET /api/v1/project/billing", protected(h.project.GetBilling))
    mux.Handle("GET /api/v1/project/users", protected(h.project.ListUsers))
    mux.Handle("GET /api/v1/project/usage", protected(h.project.GetUsage))
    
    // Root admin (root token only)
    mux.Handle("GET /api/v1/admin/status_check", protected(h.admin.StatusCheck))
}
```

---

## 4. Response Format Middleware

```go
// gateway/internal/adapter/handler/memobase/response.go

type MemobaseResponse struct {
    Data   any    `json:"data"`
    Errno  int    `json:"errno"`
    ErrMsg string `json:"errmsg"`
}

func WriteSuccess(w http.ResponseWriter, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(MemobaseResponse{
        Data:   data,
        Errno:  0,
        ErrMsg: "",
    })
}

func WriteError(w http.ResponseWriter, httpStatus, errno int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(MemobaseResponse{
        Data:   nil,
        Errno:  errno,
        ErrMsg: msg,
    })
}

// Error code mapping:
var errorCodes = map[error]struct{ HTTP, Errno int }{
    domain.ErrUnauthorized:       {401, 401},
    domain.ErrProjectSuspended:   {403, 403},
    domain.ErrNotFound:           {404, 404},
    domain.ErrInvalidBlobRole:    {400, 400},
    domain.ErrEmbeddingDisabled:  {501, 501},
    domain.ErrRateLimitExceeded:  {429, 429},
}
```

---

## 5. Auth Middleware (Dual Mode)

```go
// gateway/internal/infra/middleware/bearer_auth.go

func (m *BearerAuthMiddleware) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        if token == "" {
            WriteError(w, 401, 401, "Unauthorized")
            return
        }

        var projectID string

        if token == m.config.RootToken {
            // Root token: full access, no project scoping
            projectID = ""
        } else if strings.HasPrefix(token, "sk-proj-") {
            // Project token: validate via admin service
            result, err := m.adminClient.ValidateProjectToken(r.Context(), token)
            if err != nil {
                if errors.Is(err, domain.ErrProjectSuspended) {
                    WriteError(w, 403, 403, "Project suspended")
                    return
                }
                WriteError(w, 401, 401, "Unauthorized")
                return
            }
            projectID = result.ProjectID
        } else {
            // Fallback: check VNP Memory JWT (existing auth)
            claims, err := m.jwtValidator.Validate(token)
            if err != nil {
                WriteError(w, 401, 401, "Unauthorized")
                return
            }
            projectID = claims.TenantID
        }

        // Inject project_id vào request context
        ctx := context.WithValue(r.Context(), contextKeyProjectID, projectID)
        // Propagate to gRPC metadata
        ctx = metadata.AppendToOutgoingContext(ctx, "x-project-id", projectID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 6. Rate Limiting (Redis Sliding Window)

```go
// gateway/internal/infra/middleware/ratelimit.go

type RateLimitConfig struct {
    // Per-endpoint limits (requests per minute per project)
    Rules map[string]int  // path prefix → limit
}

var defaultRules = map[string]int{
    "POST /api/v1/blobs/insert":          100,
    "POST /api/v1/users/buffer":           20,
    "GET /api/v1/users/context":          300,
    "GET /api/v1/users/event/search":     200,
    "GET /api/v1/users/event_gist":       200,
}

// Redis sliding window implementation:
// Key: "ratelimit:{project_id}:{path_prefix}:{minute_bucket}"
// Type: Redis INCR + EXPIRE

func (m *RateLimitMiddleware) Check(projectID, path string) (allowed bool, retryAfter int) {
    rule := matchRule(path, m.config.Rules)
    if rule == 0 {
        return true, 0  // no rule → allow
    }

    key := fmt.Sprintf("ratelimit:%s:%s:%d", projectID, matchPrefix(path), minuteBucket())
    count, err := m.redis.Incr(ctx, key).Result()
    if count == 1 {
        m.redis.Expire(ctx, key, 60*time.Second)
    }

    if count > int64(rule) {
        // Calculate retry-after
        ttl, _ := m.redis.TTL(ctx, key).Result()
        return false, int(ttl.Seconds())
    }
    return true, 0
}
```

---

## 7. MCP Tools (3 Memobase Tools)

### 7.1 save_memory Tool

```go
// gateway/internal/adapter/mcp/tools/memobase/save_memory.go

func HandleSaveMemory(ctx context.Context, req SaveMemoryRequest) (*mcp.CallToolResult, error) {
    // 1. Insert blob
    blobResp, err := ingestionClient.InsertBlob(ctx, &ingestionv1.InsertBlobRequest{
        UserId:    req.UserID,
        ProjectId: extractProjectIDFromCtx(ctx),
        BlobType:  "chat",
        BlobData:  marshalChatBlob(req.Messages),
    })
    if err != nil {
        return mcp.ErrorResult(err), nil
    }

    // 2. Optional: sync flush
    flushed := false
    if req.Flush {
        _, err = ingestionClient.FlushBuffer(ctx, &ingestionv1.FlushBufferRequest{
            UserId:    req.UserID,
            ProjectId: extractProjectIDFromCtx(ctx),
            BlobType:  "chat",
            Sync:      true,  // wait for LLM processing
        })
        flushed = err == nil
    }

    return mcp.SuccessResult(map[string]any{
        "blob_id": blobResp.BlobId,
        "flushed": flushed,
    }), nil
}
```

### 7.2 get_user_profiles Tool

```go
// gateway/internal/adapter/mcp/tools/memobase/get_user_profiles.go

func HandleGetUserProfiles(ctx context.Context, req GetUserProfilesRequest) (*mcp.CallToolResult, error) {
    resp, err := contextClient.GetProfiles(ctx, &contextv1.GetProfilesRequest{
        UserId:        req.UserID,
        ProjectId:     extractProjectIDFromCtx(ctx),
        PreferTopics:  req.PreferTopics,
        MaxTokenSize:  int32(req.MaxTokenSize),
    })
    if err != nil {
        return mcp.ErrorResult(err), nil
    }

    // Format profiles as structured text for LLM context
    var profiles []map[string]string
    for _, p := range resp.Profiles {
        profiles = append(profiles, map[string]string{
            "topic":     p.Attributes.Topic,
            "sub_topic": p.Attributes.SubTopic,
            "content":   p.Content,
        })
    }
    return mcp.SuccessResult(profiles), nil
}
```

### 7.3 search_memories Tool

```go
// gateway/internal/adapter/mcp/tools/memobase/search_memories.go

func HandleSearchMemories(ctx context.Context, req SearchMemoriesRequest) (*mcp.CallToolResult, error) {
    resp, err := eventClient.SearchEvents(ctx, &eventv1.SearchEventsRequest{
        UserId:           req.UserID,
        ProjectId:        extractProjectIDFromCtx(ctx),
        Query:            req.Query,
        Topk:             int32(req.TopK),
        TimeRangeInDays:  int32(req.TimeRangeInDays),
        SimilarityThreshold: 0.2,
    })
    if err != nil {
        return mcp.ErrorResult(err), nil
    }

    var results []map[string]any
    for _, e := range resp.Results {
        results = append(results, map[string]any{
            "event_tip":  e.Event.EventData.EventTip,
            "similarity": e.Similarity,
            "date":       e.Event.CreatedAt,
        })
    }
    return mcp.SuccessResult(results), nil
}
```

### 7.4 MCP Tool Registration

```go
// gateway/internal/adapter/mcp/server.go

func (s *MCPServer) RegisterMemobaseTools() {
    s.RegisterTool(mcp.Tool{
        Name:        "save_memory",
        Description: "Save a conversation or piece of information to the user's long-term memory...",
        InputSchema: saveMemorySchema,
        Handler:     HandleSaveMemory,
    })
    s.RegisterTool(mcp.Tool{
        Name:        "get_user_profiles",
        Description: "Retrieve the structured user profile information...",
        InputSchema: getUserProfilesSchema,
        Handler:     HandleGetUserProfiles,
    })
    s.RegisterTool(mcp.Tool{
        Name:        "search_memories",
        Description: "Search through the user's event timeline using semantic similarity...",
        InputSchema: searchMemoriesSchema,
        Handler:     HandleSearchMemories,
    })
}
```

---

## 8. MCP SSE Transport

```go
// gateway/internal/infra/server/mcp_sse.go

// SSE endpoint: GET /mcp/sse
// Cho phép Claude Desktop remote URL, Cursor, Windsurf kết nối

func (s *MCPSSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")  // MCP clients cần CORS

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    // MCP protocol over SSE
    session := s.mcpServer.NewSSESession(r.Context())
    defer session.Close()

    for {
        select {
        case msg := <-session.Messages():
            fmt.Fprintf(w, "data: %s\n\n", msg)
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

---

## 9. CORS Middleware

```go
// gateway/internal/infra/middleware/cors.go

type CORSConfig struct {
    Enabled      bool     // USE_CORS env
    AllowedHosts []string // API_HOSTS env (comma-separated)
}

func NewCORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !cfg.Enabled {
                next.ServeHTTP(w, r)
                return
            }

            origin := r.Header.Get("Origin")
            if isAllowedOrigin(origin, cfg.AllowedHosts) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
                w.Header().Set("Access-Control-Expose-Headers", "X-Process-Time, X-Request-ID")
            }

            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 10. Timing & Request ID Middleware

```go
// gateway/internal/infra/middleware/timing.go
func TimingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        // Use ResponseRecorder to set header after handler completes
        rw := &responseWriter{ResponseWriter: w}
        next.ServeHTTP(rw, r)
        elapsed := time.Since(start).Milliseconds()
        w.Header().Set("X-Process-Time", fmt.Sprintf("%dms", elapsed))
    })
}

// gateway/internal/infra/middleware/request_id.go
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        w.Header().Set("X-Request-ID", requestID)
        // Propagate to all gRPC calls
        ctx := metadata.AppendToOutgoingContext(r.Context(), "x-request-id", requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## 11. Health Check Aggregation

```go
// gateway/internal/adapter/handler/memobase/health_handler.go

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    // Parallel health checks for all memobase services
    type svcHealth struct {
        name   string
        status string
        latency int64
    }

    services := []string{"ingestion", "engine", "context", "event", "admin"}
    results := make(chan svcHealth, len(services)+3)  // +3 for infra

    for _, svc := range services {
        go func(name string) {
            start := time.Now()
            client := h.clients[name]
            err := client.HealthCheck(ctx)
            status := "healthy"
            if err != nil { status = "unhealthy" }
            results <- svcHealth{name, status, time.Since(start).Milliseconds()}
        }(svc)
    }

    // Also check infra
    go checkInfra(ctx, "postgres", h.db, results)
    go checkInfra(ctx, "redis", h.redis, results)
    go checkInfra(ctx, "nats", h.nats, results)

    // Collect results
    serviceMap := make(map[string]any)
    overallStatus := "healthy"
    for i := 0; i < len(services)+3; i++ {
        r := <-results
        serviceMap[r.name] = map[string]any{
            "status": r.status,
            "latency_ms": r.latency,
        }
        if r.status != "healthy" { overallStatus = "degraded" }
    }

    WriteSuccess(w, map[string]any{
        "status":   overallStatus,
        "services": serviceMap,
    })
}
```

---

## 12. Configuration

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
  enabled: "${USE_CORS}"         # false by default
  allowed_hosts: "${API_HOSTS}"  # comma-separated

mcp:
  enabled: true
  sse_path: "/mcp/sse"
  sse_port: 8082
  stdio_enabled: true

rate_limit:
  redis_url: "${REDIS_URL}"
  rules:
    "POST /api/v1/blobs/insert": 100  # per minute per project
    "POST /api/v1/users/buffer": 20
    "GET /api/v1/users/context": 300
    "GET /api/v1/users/event/search": 200

services:
  ingestion: { address: "memobase-ingestion:9041", timeout: 30s }
  engine:    { address: "memobase-engine:9042",    timeout: 120s }
  context:   { address: "memobase-context:9043",   timeout: 5s }
  event:     { address: "memobase-event:9044",     timeout: 10s }
  admin:     { address: "memobase-admin:9045",     timeout: 5s }
```

---

## 13. Testing Strategy

### Unit Tests
- `TestBearerAuthMiddleware_RootToken` → passes through
- `TestBearerAuthMiddleware_ProjectToken` → projectID extracted, injected into context
- `TestBearerAuthMiddleware_SuspendedProject` → 403 returned
- `TestBearerAuthMiddleware_NoToken` → 401 returned
- `TestRateLimitMiddleware_ExceedsLimit` → 429 + Retry-After header
- `TestTimingMiddleware_HeaderPresent` → X-Process-Time in response
- `TestCORSMiddleware_AllowedOrigin` → CORS headers present
- `TestMemobaseResponseWriter_ErrorFormat` → {data: null, errno: X, errmsg: Y}

### Integration Tests
- `TestMCPSaveMemory_E2E` — MCP tool → inserts blob
- `TestMCPGetProfiles_E2E` — MCP tool → returns profiles
- `TestMCPSearchMemories_E2E` — MCP tool → returns events
- `TestHealthCheck_AllServicesUp` — all healthy → {"status": "healthy"}

---

## 14. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| Rate limit Redis key collision giữa projects | Thấp | Key include project_id: `ratelimit:{project_id}:{path}:{bucket}` |
| MCP SSE connection leak | Thấp | `defer session.Close()` + context cancellation |
| `X-Process-Time` không chính xác nếu middleware chain dài | Thấp | Measure từ sau Recovery middleware (trừ panic recovery time) |
| sk-proj-* token validation overhead (bcrypt) | Trung bình | Cache validated tokens trong Redis với TTL 60s |
| CORS wildcard exposure qua MCP SSE | Thấp | SSE endpoint dùng `*` do MCP protocol yêu cầu; REST endpoints dùng `API_HOSTS` |
