# TASK-MB-012 — Gateway REST Routes: Memobase HTTP Handlers & Middleware

**Wave:** 4 (Access Layer)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-MB-011 (pkg/observability + middleware), TASK-MB-002/003/004/009/010 (tất cả services sẵn sàng)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-MB-006 §3-§12](../solutions/SOL-MB-006-Gateway-MCP-Server.md)

**Trạng thái:** 🔄 Partial  
**Ghi chú:** gateway routes present but memobase context proxy incomplete  
---

## Mục tiêu

Mở rộng `gateway/` hiện có để thêm toàn bộ memobase REST API (25+ endpoints), middleware stack (Auth dual-mode, CORS, timing, rate limit, request ID), và health check aggregation. **Không tạo gateway mới** — mở rộng gateway hiện tại của VNP Memory.

---

## Các file cần tạo/sửa đổi trong `gateway/`

```
gateway/
├── internal/
│   ├── adapter/
│   │   ├── handler/memobase/              ← [NEW directory]
│   │   │   ├── blob_handler.go
│   │   │   ├── buffer_handler.go
│   │   │   ├── profile_handler.go
│   │   │   ├── context_handler.go
│   │   │   ├── event_handler.go
│   │   │   ├── project_handler.go
│   │   │   ├── admin_handler.go
│   │   │   ├── health_handler.go
│   │   │   └── response.go                # MemobaseResponse format
│   │   └── client/                        ← [NEW clients]
│   │       ├── ingestion_client.go
│   │       ├── context_client.go
│   │       ├── event_client.go
│   │       └── admin_client.go
│   └── infra/
│       ├── middleware/                     ← [NEW middleware]
│       │   ├── bearer_auth.go             # Dual-mode: root token + sk-proj-*
│       │   ├── cors.go
│       │   ├── timing.go                  # X-Process-Time header
│       │   ├── ratelimit.go               # Redis sliding window
│       │   └── request_id.go              # X-Request-ID
│       └── server/
│           └── router.go                  ← [MODIFY] Add registerMemobaseRoutes()
```

---

## 1. Response Format Helper

**File: `internal/adapter/handler/memobase/response.go`**

```go
package memobase

type MemobaseResponse struct {
    Data   any    `json:"data"`
    Errno  int    `json:"errno"`
    ErrMsg string `json:"errmsg"`
}

func WriteSuccess(w http.ResponseWriter, data any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(MemobaseResponse{Data: data, Errno: 0, ErrMsg: ""})
}

func WriteError(w http.ResponseWriter, httpStatus, errno int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(MemobaseResponse{Data: nil, Errno: errno, ErrMsg: msg})
}

func WriteGRPCError(w http.ResponseWriter, err error) {
    code := status.Code(err)
    switch code {
    case codes.NotFound:         WriteError(w, 404, 404, err.Error())
    case codes.Unauthenticated:  WriteError(w, 401, 401, "Unauthorized")
    case codes.PermissionDenied: WriteError(w, 403, 403, "Forbidden")
    case codes.InvalidArgument:  WriteError(w, 400, 400, err.Error())
    case codes.Unimplemented:    WriteError(w, 501, 501, err.Error())
    case codes.ResourceExhausted: WriteError(w, 429, 429, "Rate limit exceeded")
    default:                     WriteError(w, 500, 500, "Internal server error")
    }
}
```

---

## 2. HTTP Handlers

### `internal/adapter/handler/memobase/blob_handler.go`

```go
type BlobHandler struct {
    ingestionClient IngestionGRPCClient
}

// POST /api/v1/blobs/insert/{user_id}
// Body: {"blob_type": "chat", "blob_data": {...}, "additional_fields": {}}
func (h *BlobHandler) Insert(w http.ResponseWriter, r *http.Request) {
    userID := r.PathValue("user_id")
    projectID := projectIDFromCtx(r.Context())

    var req struct {
        BlobType         string         `json:"blob_type"`
        BlobData         json.RawMessage `json:"blob_data"`
        AdditionalFields map[string]string `json:"additional_fields"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        WriteError(w, 400, 400, "invalid request body")
        return
    }

    resp, err := h.ingestionClient.InsertBlob(r.Context(), &ingestionv1.InsertBlobRequest{
        UserId: userID, ProjectId: projectID,
        BlobType: req.BlobType, BlobData: req.BlobData,
        AdditionalFields: req.AdditionalFields,
    })
    if err != nil { WriteGRPCError(w, err); return }

    WriteSuccess(w, map[string]any{
        "blob_id":        resp.BlobId,
        "flush_triggered": resp.FlushTriggered,
    })
}

// GET /api/v1/blobs/{user_id}/{blob_id}
func (h *BlobHandler) Get(w http.ResponseWriter, r *http.Request)

// DELETE /api/v1/blobs/{user_id}/{blob_id}
func (h *BlobHandler) Delete(w http.ResponseWriter, r *http.Request)
```

### `internal/adapter/handler/memobase/buffer_handler.go`

```go
// POST /api/v1/users/buffer/{user_id}/{buffer_type}
// Body: {"sync": false} (optional)
func (h *BufferHandler) Flush(w http.ResponseWriter, r *http.Request) {
    userID     := r.PathValue("user_id")
    bufferType := r.PathValue("buffer_type")  // "chat" | "doc" | "summary"
    projectID  := projectIDFromCtx(r.Context())

    var body struct { Sync bool `json:"sync"` }
    json.NewDecoder(r.Body).Decode(&body)

    resp, err := h.ingestionClient.FlushBuffer(r.Context(), &ingestionv1.FlushBufferRequest{
        UserId: userID, ProjectId: projectID, BlobType: bufferType, Sync: body.Sync,
    })
    if err != nil { WriteGRPCError(w, err); return }

    WriteSuccess(w, map[string]any{
        "blobs_flushed": resp.BlobsFlushed,
        "skipped":       resp.Skipped,
    })
}

// GET /api/v1/users/buffer/capacity/{user_id}/{buffer_type}
func (h *BufferHandler) GetCapacity(w http.ResponseWriter, r *http.Request)
```

### `internal/adapter/handler/memobase/profile_handler.go`

```go
// GET /api/v1/users/profile/{user_id}?prefer_topics=work,basic_info&max_token_size=500
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
    userID    := r.PathValue("user_id")
    projectID := projectIDFromCtx(r.Context())
    maxTokens, _ := strconv.Atoi(r.URL.Query().Get("max_token_size"))
    preferTopics  := splitComma(r.URL.Query().Get("prefer_topics"))

    resp, err := h.contextClient.GetProfiles(r.Context(), &contextv1.GetProfilesRequest{
        UserId: userID, ProjectId: projectID,
        MaxTokenSize: int32(maxTokens), PreferTopics: preferTopics,
    })
    if err != nil { WriteGRPCError(w, err); return }

    WriteSuccess(w, resp.Profiles)
}

// POST /api/v1/users/profile/{user_id}
// Body: {"content": "...", "topic": "work", "sub_topic": "company"}
func (h *ProfileHandler) Add(w http.ResponseWriter, r *http.Request)

// PUT /api/v1/users/profile/{user_id}/{profile_id}
// Body: {"content": "..."}
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request)

// DELETE /api/v1/users/profile/{user_id}/{profile_id}
func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request)
```

### `internal/adapter/handler/memobase/context_handler.go`

```go
// GET /api/v1/users/context/{user_id}
// Query: max_token_size, prefer_topics, only_topics, profile_event_ratio,
//        event_similarity_threshold, time_range_in_days, event_topk
// Body (optional POST): {"chats": [...], "customize_context_prompt": "..."}
func (h *ContextHandler) Get(w http.ResponseWriter, r *http.Request) {
    userID    := r.PathValue("user_id")
    projectID := projectIDFromCtx(r.Context())
    q := r.URL.Query()

    req := &contextv1.GetContextRequest{
        UserId:    userID, ProjectId: projectID,
        MaxTokenSize:             int32OrDefault(q.Get("max_token_size"), 500),
        PreferTopics:             splitComma(q.Get("prefer_topics")),
        OnlyTopics:               splitComma(q.Get("only_topics")),
        ProfileEventRatio:        float64OrDefault(q.Get("profile_event_ratio"), 0.7),
        EventSimilarityThreshold: float64OrDefault(q.Get("event_similarity_threshold"), 0.2),
        TimeRangeInDays:          int32OrDefault(q.Get("time_range_in_days"), 21),
        EventTopk:                int32OrDefault(q.Get("event_topk"), 5),
    }

    // Optional body with chats for semantic search
    if r.ContentLength > 0 {
        var body struct {
            Chats                  []ChatMessage `json:"chats"`
            CustomizeContextPrompt string        `json:"customize_context_prompt"`
        }
        json.NewDecoder(r.Body).Decode(&body)
        req.Chats = mapChats(body.Chats)
        req.CustomizeContextPrompt = body.CustomizeContextPrompt
    }

    resp, err := h.contextClient.GetContext(r.Context(), req)
    if err != nil { WriteGRPCError(w, err); return }

    WriteSuccess(w, map[string]any{
        "context":       resp.ContextStr,
        "profile_count": resp.ProfileCount,
        "event_count":   resp.EventCount,
        "tokens_used":   resp.TokensUsed,
    })
}
```

### `internal/adapter/handler/memobase/event_handler.go`

```go
// GET /api/v1/users/event/{user_id}?page=1&page_size=20
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request)

// GET /api/v1/users/event/search/{user_id}?query=...&topk=5
func (h *EventHandler) Search(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("query")
    if query == "" { WriteError(w, 400, 400, "query is required"); return }

    resp, err := h.eventClient.SearchEvents(r.Context(), &eventv1.SearchEventsRequest{
        UserId: r.PathValue("user_id"), ProjectId: projectIDFromCtx(r.Context()),
        Query: query,
        Topk:  int32OrDefault(r.URL.Query().Get("topk"), 10),
        TimeRangeInDays: int32OrDefault(r.URL.Query().Get("time_range_in_days"), 21),
    })
    if err != nil { WriteGRPCError(w, err); return }
    WriteSuccess(w, resp.Results)
}

// GET /api/v1/users/event_gist/search/{user_id}?query=...&topk=5
func (h *EventHandler) SearchGists(w http.ResponseWriter, r *http.Request)

// GET /api/v1/users/event_tags/search/{user_id}?has_event_tag=emotion&event_tag_equal=status:stressed
func (h *EventHandler) FilterByTags(w http.ResponseWriter, r *http.Request)

// PUT /api/v1/users/event/{user_id}/{event_id}
func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request)

// DELETE /api/v1/users/event/{user_id}/{event_id}
func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request)
```

### `internal/adapter/handler/memobase/admin_handler.go`

```go
// POST /api/v1/users — CreateUser
// GET /api/v1/users/{user_id} — GetUser
// PUT /api/v1/users/{user_id} — UpdateUser
// DELETE /api/v1/users/{user_id} — DeleteUser
```

### `internal/adapter/handler/memobase/project_handler.go`

```go
// GET /api/v1/project/profile_config
// POST /api/v1/project/profile_config  Body: {"config": "yaml string"}
// GET /api/v1/project/billing
// GET /api/v1/project/users?limit=100&cursor=...
// GET /api/v1/project/usage?days=30
```

---

## 3. Middleware

### `internal/infra/middleware/bearer_auth.go`

```go
// Dual-mode auth:
// 1. ACCESS_TOKEN (root token) → full admin access
// 2. "sk-proj-{id}-{secret}" → project-scoped access via admin.ValidateProjectToken
// 3. VNP Memory JWT → existing auth path
type BearerAuthMiddleware struct {
    rootToken   string
    adminClient AdminGRPCClient
    jwtValidator JWTValidator
}

func (m *BearerAuthMiddleware) Handle(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        if token == "" {
            WriteError(w, 401, 401, "Unauthorized")
            return
        }

        var projectID string
        switch {
        case token == m.rootToken:
            projectID = ""  // root: no project scoping
        case strings.HasPrefix(token, "sk-proj-"):
            result, err := m.adminClient.ValidateProjectToken(r.Context(), token)
            if err != nil || !result.Valid {
                if isGRPCCode(err, codes.PermissionDenied) {
                    WriteError(w, 403, 403, "Project suspended")
                } else {
                    WriteError(w, 401, 401, "Unauthorized")
                }
                return
            }
            projectID = result.ProjectID
        default:
            // VNP Memory JWT (existing auth)
            claims, err := m.jwtValidator.Validate(token)
            if err != nil { WriteError(w, 401, 401, "Unauthorized"); return }
            projectID = claims.TenantID
        }

        ctx := contextWithProjectID(r.Context(), projectID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### `internal/infra/middleware/timing.go`

```go
func TimingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        elapsed := time.Since(start).Milliseconds()
        w.Header().Set("X-Process-Time", fmt.Sprintf("%dms", elapsed))
    })
}
```

### `internal/infra/middleware/cors.go`

```go
// USE_CORS=true, API_HOSTS=http://localhost:3000,https://app.example.com
func NewCORSMiddleware(enabled bool, allowedHosts []string) func(http.Handler) http.Handler
// Handles OPTIONS preflight
// Sets Access-Control-Allow-Origin, Methods, Headers, Expose-Headers
```

### `internal/infra/middleware/ratelimit.go`

```go
// Redis INCR + EXPIRE (1-minute window) per project per path prefix
// Returns 429 + X-RateLimit-Remaining + Retry-After when exceeded
```

### `internal/infra/middleware/request_id.go`

```go
// Reads X-Request-ID from request header (or generates UUID)
// Sets X-Request-ID on response header
// Propagates via gRPC metadata
```

---

## 4. Route Registration

**File: `internal/infra/server/router.go`** (MODIFY — add to existing router)

```go
func (s *Server) registerMemobaseRoutes(mux *http.ServeMux) {
    h := newMemobaseHandlers(s.clients)

    protected := buildMiddlewareChain(
        s.middleware.Recovery,
        s.middleware.RequestID,
        s.middleware.Logger,
        s.middleware.Timing,
        s.middleware.CORS,
        s.middleware.BearerAuth,
        s.middleware.RateLimit,
    )

    // Health (no auth)
    mux.Handle("GET /api/v1/healthcheck", http.HandlerFunc(h.health.Check))

    // Users (admin)
    mux.Handle("POST /api/v1/users",                   protected(h.admin.CreateUser))
    mux.Handle("GET /api/v1/users/{user_id}",           protected(h.admin.GetUser))
    mux.Handle("PUT /api/v1/users/{user_id}",           protected(h.admin.UpdateUser))
    mux.Handle("DELETE /api/v1/users/{user_id}",        protected(h.admin.DeleteUser))

    // Blobs (ingestion)
    mux.Handle("POST /api/v1/blobs/insert/{user_id}",   protected(h.blob.Insert))
    mux.Handle("GET /api/v1/blobs/{user_id}/{blob_id}", protected(h.blob.Get))
    mux.Handle("DELETE /api/v1/blobs/{user_id}/{blob_id}", protected(h.blob.Delete))

    // Buffer (ingestion)
    mux.Handle("POST /api/v1/users/buffer/{user_id}/{buffer_type}",         protected(h.buffer.Flush))
    mux.Handle("GET /api/v1/users/buffer/capacity/{user_id}/{buffer_type}", protected(h.buffer.GetCapacity))

    // Profile (context)
    mux.Handle("GET /api/v1/users/profile/{user_id}",                      protected(h.profile.Get))
    mux.Handle("POST /api/v1/users/profile/{user_id}",                     protected(h.profile.Add))
    mux.Handle("PUT /api/v1/users/profile/{user_id}/{profile_id}",         protected(h.profile.Update))
    mux.Handle("DELETE /api/v1/users/profile/{user_id}/{profile_id}",      protected(h.profile.Delete))

    // Context (context)
    mux.Handle("GET /api/v1/users/context/{user_id}",   protected(h.ctx.Get))

    // Events (event)
    mux.Handle("GET /api/v1/users/event/{user_id}",                   protected(h.event.List))
    mux.Handle("PUT /api/v1/users/event/{user_id}/{event_id}",        protected(h.event.Update))
    mux.Handle("DELETE /api/v1/users/event/{user_id}/{event_id}",     protected(h.event.Delete))
    mux.Handle("GET /api/v1/users/event/search/{user_id}",            protected(h.event.Search))
    mux.Handle("GET /api/v1/users/event_gist/search/{user_id}",       protected(h.event.SearchGists))
    mux.Handle("GET /api/v1/users/event_tags/search/{user_id}",       protected(h.event.FilterByTags))

    // Project (admin)
    mux.Handle("POST /api/v1/project/profile_config", protected(h.project.UpdateConfig))
    mux.Handle("GET /api/v1/project/profile_config",  protected(h.project.GetConfig))
    mux.Handle("GET /api/v1/project/billing",         protected(h.project.GetBilling))
    mux.Handle("GET /api/v1/project/users",           protected(h.project.ListUsers))
    mux.Handle("GET /api/v1/project/usage",           protected(h.project.GetUsage))

    // Admin status (root token only)
    mux.Handle("GET /api/v1/admin/status_check", protected(h.admin.StatusCheck))
}
```

---

## Unit Tests

```
TestBlobHandler_Insert_ValidRequest        → 200 + blob_id in response
TestBlobHandler_Insert_InvalidBlobType    → 400 via gRPC InvalidArgument
TestBlobHandler_Insert_BadJSON            → 400 "invalid request body"
TestBlobHandler_Get_NotFound              → 404 via gRPC NotFound
TestBufferHandler_Flush_Success           → 200 + blobs_flushed
TestBufferHandler_Flush_Skipped           → 200 + skipped=true
TestBufferHandler_GetCapacity             → 200 + num_blobs, num_tokens
TestProfileHandler_Get_WithPreferTopics   → query params parsed → gRPC called
TestProfileHandler_Add_ValidBody          → 201 + profile id
TestContextHandler_Get_NoChats            → 200 + context_str
TestContextHandler_Get_WithChats          → chats sent → gRPC GetContext with chats
TestContextHandler_Get_DefaultValues      → missing query params → defaults applied
TestEventHandler_Search_NoQuery           → 400 "query is required"
TestEventHandler_Search_ValidQuery        → gRPC SearchEvents called → 200
TestEventHandler_FilterByTags_MultiTags   → multiple has_event_tag → parsed correctly
TestAdminHandler_CreateUser               → 201 + user_id
TestAdminHandler_DeleteUser_Cascade       → 200
TestProjectHandler_UpdateConfig_Valid     → 200
TestProjectHandler_ListUsers_WithCursor   → cursor param forwarded
TestBearerAuth_RootToken                  → passes through without admin call
TestBearerAuth_ProjectToken_Valid         → admin.ValidateProjectToken called → 200
TestBearerAuth_ProjectToken_Suspended     → 403
TestBearerAuth_NoToken                    → 401
TestBearerAuth_JWT                        → VNP Memory JWT → existing path
TestTimingMiddleware_HeaderSet            → X-Process-Time in response
TestCORSMiddleware_AllowedOrigin          → CORS headers set
TestCORSMiddleware_Preflight              → OPTIONS → 204
TestCORSMiddleware_Disabled              → USE_CORS=false → no CORS headers
TestRateLimitMiddleware_BelowLimit       → 200
TestRateLimitMiddleware_ExceedsLimit     → 429 + Retry-After
TestRequestIDMiddleware_GeneratesID      → missing → UUID generated
TestRequestIDMiddleware_PreservesExisting → existing X-Request-ID → same returned
TestMemobaseResponse_Success             → {data: {...}, errno: 0, errmsg: ""}
TestMemobaseResponse_Error              → {data: null, errno: 404, errmsg: "..."}
TestWriteGRPCError_NotFound             → codes.NotFound → 404
TestWriteGRPCError_Unauthenticated      → codes.Unauthenticated → 401
TestHealthHandler_AllHealthy            → 200 + status=healthy
TestHealthHandler_ServiceDown           → 200 + status=degraded
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./gateway/...
go test ./gateway/internal/adapter/handler/memobase/... -v -count=1
go test ./gateway/internal/infra/middleware/... -v -count=1

# Integration test với tất cả services running
go test ./gateway/... -tags integration -v -count=1
```

---

## Ghi chú triển khai

- Dùng Go 1.22+ `http.ServeMux` với `r.PathValue("param")` — không cần gorilla/mux
- `extractBearerToken`: `strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")`
- `projectIDFromCtx`: extract từ context key `contextKeyProjectID` (set bởi BearerAuth)
- **Rate limit**: Redis INCR + EXPIRE với 1-minute bucket key — simple và hiệu quả
- `WriteGRPCError` mapping: đảm bảo gRPC status codes ánh xạ đúng HTTP codes
- `buildMiddlewareChain`: compose từ ngoài vào trong (first=outermost)
- Existing VNP Memory routes không bị ảnh hưởng — chỉ thêm `/api/v1/` prefix group mới
