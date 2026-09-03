# TASK-OV-016 — `services/openviking-gateway` REST API & Auth Middleware

**Wave:** 6 (Gateway — xây cuối cùng)  
**Ưu tiên:** Critical  
**Phụ thuộc:** Tất cả Wave 1-5 services phải có gRPC server  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-OV-001 §3, §4, §5, §9](../solutions/SOL-OV-001-Gateway-Service.md)  
**Port HTTP:** 8080

---

## Mục tiêu

Tạo phần REST của `services/openviking-gateway/` — chi router với 17 route groups, 3-mode authentication (DEV/API_KEY/TRUSTED), RBAC namespace check, rate limiting, circuit breakers cho từng upstream, và gRPC clients cho tất cả backend services.

---

## Cấu trúc thư mục

```
services/openviking-gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── identity.go    # AuthMode enum
│   │   └── errors.go
│   ├── usecase/
│   │   ├── resolve_identity.go  # Dispatch to DEV/API_KEY/TRUSTED
│   │   ├── check_namespace.go   # RBAC namespace validation
│   │   └── port/output.go       # AdminClientInterface
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── router.go              # chi router, all 17 route groups
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go            # 3-mode auth middleware
│   │   │   │   ├── tenant.go          # Tenant isolation
│   │   │   │   └── ratelimit.go       # Per-tenant rate limit
│   │   │   ├── filesystem_handler.go
│   │   │   ├── content_handler.go
│   │   │   ├── search_handler.go
│   │   │   ├── session_handler.go
│   │   │   ├── resource_handler.go
│   │   │   ├── relations_handler.go
│   │   │   ├── admin_handler.go       # Requires ADMIN/ROOT role
│   │   │   ├── observer_handler.go
│   │   │   ├── privacy_handler.go
│   │   │   ├── tasks_handler.go
│   │   │   ├── system_handler.go
│   │   │   ├── debug_handler.go
│   │   │   ├── bot_handler.go
│   │   │   ├── pack_handler.go
│   │   │   ├── maintenance_handler.go
│   │   │   ├── stats_handler.go
│   │   │   └── helpers.go             # writeJSON, httpError, mapGRPCError
│   │   └── client/
│   │       ├── fs_client.go
│   │       ├── search_client.go
│   │       ├── session_client.go
│   │       ├── resource_client.go
│   │       └── admin_client.go
│   └── infra/
│       ├── config/config.go
│       ├── server/http.go
│       └── wire/wire.go
```

---

## 1. Auth Middleware — 3 Modes

**File: `internal/adapter/http/middleware/auth.go`**

```go
type AuthMiddleware struct {
    mode      domain.AuthMode    // "dev" | "api_key" | "trusted"
    resolver  auth.KeyResolver   // From pkg/auth/
    localNets []net.IPNet        // For DEV: 127.0.0.0/8, ::1/128
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rc, err := m.resolve(r)
        if err != nil {
            httpError(w, http.StatusUnauthorized, err.Error())
            return
        }
        ctx := viking.WithContext(r.Context(), rc)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (m *AuthMiddleware) resolve(r *http.Request) (*viking.RequestContext, error) {
    switch m.mode {
    case domain.AuthModeDev:
        return m.resolveDev(r)
    case domain.AuthModeAPIKey:
        return m.resolveAPIKey(r)
    case domain.AuthModeTrusted:
        return m.resolveTrusted(r)
    }
    return nil, fmt.Errorf("unknown auth mode: %s", m.mode)
}

// DEV mode: only localhost, ROOT role, read account/user from headers (optional)
func (m *AuthMiddleware) resolveDev(r *http.Request) (*viking.RequestContext, error) {
    ip := net.ParseIP(getRealIP(r))
    for _, localNet := range m.localNets {
        if localNet.Contains(ip) {
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
    return nil, fmt.Errorf("dev mode: connection from non-localhost IP %s rejected", ip)
}

// API_KEY mode
func (m *AuthMiddleware) resolveAPIKey(r *http.Request) (*viking.RequestContext, error) {
    apiKey := r.Header.Get("X-Api-Key")
    if apiKey == "" {
        if bearer := r.Header.Get("Authorization"); strings.HasPrefix(bearer, "Bearer ") {
            apiKey = strings.TrimPrefix(bearer, "Bearer ")
        }
    }
    if apiKey == "" {
        return nil, fmt.Errorf("missing API key (X-Api-Key or Authorization: Bearer)")
    }
    resolved, err := m.resolver.Resolve(r.Context(), apiKey)
    if err != nil {
        return nil, fmt.Errorf("invalid API key")
    }
    return &viking.RequestContext{
        Role:     resolved.Role,
        APIKeyID: resolved.KeyID,
        User: viking.UserIdentifier{
            AccountID: resolved.AccountID,
            UserID:    resolved.UserID,
            AgentID:   r.Header.Get("X-OpenViking-Agent"),
        },
    }, nil
}

// TRUSTED mode: read headers directly (used within internal service mesh with mTLS)
func (m *AuthMiddleware) resolveTrusted(r *http.Request) (*viking.RequestContext, error) {
    accountID := r.Header.Get("X-OpenViking-Account")
    if accountID == "" {
        return nil, fmt.Errorf("X-OpenViking-Account header required in trusted mode")
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

// parseRoleHeader: "ROOT" → RoleRoot, "ADMIN" → RoleAdmin, etc.
func parseRoleHeader(s string) viking.Role {
    switch strings.ToUpper(s) {
    case "ROOT":  return viking.RoleRoot
    case "ADMIN": return viking.RoleAdmin
    case "BOT":   return viking.RoleBot
    default:      return viking.RoleUser
    }
}
```

---

## 2. Router Setup

**File: `internal/adapter/http/router.go`**

```go
func NewRouter(deps *Dependencies) http.Handler {
    r := chi.NewRouter()
    
    // Global middleware (no auth)
    r.Use(
        middleware.RequestID,
        middleware.RealIP,
        otelMiddleware.Middleware("openviking-gateway"),
        loggingMiddleware.HTTPLogger,
        middleware.Recoverer,
    )
    
    // Health (no auth)
    r.Get("/healthz", deps.System.HealthzHandler)
    r.Get("/readyz",  deps.System.ReadyzHandler)
    r.Handle("/metrics", promhttp.Handler())
    
    // API v1 (requires auth)
    r.Route("/api/v1", func(r chi.Router) {
        r.Use(deps.AuthMW.Handler)       // Auth resolves identity
        r.Use(deps.TenantMW.Handler)     // Validates tenant consistency
        r.Use(deps.RateLimitMW.Handler)  // Per-tenant rate limiting
        
        r.Mount("/filesystem",      newFilesystemRouter(deps))
        r.Mount("/content",         newContentRouter(deps))
        r.Mount("/search",          newSearchRouter(deps))
        r.Mount("/sessions",        newSessionRouter(deps))
        r.Mount("/resources",       newResourceRouter(deps))
        r.Mount("/relations",       newRelationsRouter(deps))
        r.Mount("/admin",           newAdminRouter(deps))      // RoleAdmin+
        r.Mount("/observer",        newObserverRouter(deps))
        r.Mount("/privacy-configs", newPrivacyRouter(deps))
        r.Mount("/tasks",           newTaskRouter(deps))
        r.Mount("/system",          newSystemRouter(deps))
        r.Mount("/debug",           newDebugRouter(deps))
        r.Mount("/bot",             newBotRouter(deps))
        r.Mount("/pack",            newPackRouter(deps))
        r.Mount("/maintenance",     newMaintenanceRouter(deps))  // RoleAdmin+
        r.Mount("/stats",           newStatsRouter(deps))
    })
    
    return r
}

// Example route group:
func newFilesystemRouter(deps *Dependencies) chi.Router {
    r := chi.NewRouter()
    h := http.NewFilesystemHandler(deps.FSClient, deps.NamespaceChecker)
    
    r.Post("/ls",    h.ListDirectory)
    r.Post("/tree",  h.Tree)
    r.Post("/mkdir", h.Mkdir)
    r.Post("/rm",    h.Remove)
    r.Post("/stat",  h.Stat)
    r.Post("/exists", h.Exists)
    return r
}

func newContentRouter(deps *Dependencies) chi.Router {
    r := chi.NewRouter()
    h := http.NewContentHandler(deps.FSClient, deps.NamespaceChecker)
    
    r.Post("/read",  h.Read)
    r.Post("/write", h.Write)
    r.Post("/mv",    h.Move)
    r.Post("/cp",    h.Copy)
    return r
}

func newSearchRouter(deps *Dependencies) chi.Router {
    r := chi.NewRouter()
    h := http.NewSearchHandler(deps.SearchClient, deps.SessionClient)
    
    r.Post("/find",   h.Find)      // Stateless search
    r.Post("/search", h.Search)    // Session-aware search
    r.Post("/grep",   h.Grep)
    r.Post("/glob",   h.Glob)
    return r
}
```

---

## 3. Sample Handler Implementation

**File: `internal/adapter/http/filesystem_handler.go`**

```go
type FilesystemHandler struct {
    fsClient  port.FSClient
    checker   *usecase.NamespaceChecker
}

// POST /api/v1/filesystem/ls
func (h *FilesystemHandler) ListDirectory(w http.ResponseWriter, r *http.Request) {
    var req struct {
        URI string `json:"uri"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
        return
    }
    if req.URI == "" {
        httpError(w, http.StatusBadRequest, "uri is required")
        return
    }
    if err := h.checker.CheckAccess(r.Context(), req.URI); err != nil {
        httpError(w, http.StatusForbidden, err.Error())
        return
    }
    
    entries, err := h.fsClient.Ls(r.Context(), req.URI)
    if err != nil {
        mapGRPCError(w, err)
        return
    }
    
    writeJSON(w, http.StatusOK, map[string]any{
        "uri":     req.URI,
        "entries": entries,
        "count":   len(entries),
    })
}

// POST /api/v1/filesystem/tree
func (h *FilesystemHandler) Tree(w http.ResponseWriter, r *http.Request) {
    var req struct {
        URI      string `json:"uri"`
        MaxDepth int    `json:"max_depth"`
        Format   string `json:"format"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    if req.MaxDepth == 0 { req.MaxDepth = 3 }
    if err := h.checker.CheckAccess(r.Context(), req.URI); err != nil {
        httpError(w, http.StatusForbidden, err.Error())
        return
    }
    
    tree, err := h.fsClient.Tree(r.Context(), req.URI, req.MaxDepth, req.Format)
    if err != nil { mapGRPCError(w, err); return }
    writeJSON(w, http.StatusOK, tree)
}
```

**File: `internal/adapter/http/helpers.go`**

```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]string{
        "error":   http.StatusText(status),
        "message": msg,
    })
}

func mapGRPCError(w http.ResponseWriter, err error) {
    if errors.Is(err, gobreaker.ErrOpenState) {
        httpError(w, http.StatusServiceUnavailable, "upstream service unavailable (circuit open)")
        return
    }
    st, ok := status.FromError(err)
    if !ok { httpError(w, http.StatusInternalServerError, err.Error()); return }
    
    switch st.Code() {
    case codes.NotFound:         httpError(w, http.StatusNotFound, st.Message())
    case codes.AlreadyExists:    httpError(w, http.StatusConflict, st.Message())
    case codes.PermissionDenied: httpError(w, http.StatusForbidden, st.Message())
    case codes.Unauthenticated:  httpError(w, http.StatusUnauthorized, st.Message())
    case codes.InvalidArgument:  httpError(w, http.StatusBadRequest, st.Message())
    case codes.ResourceExhausted: httpError(w, http.StatusTooManyRequests, st.Message())
    case codes.Unavailable:      httpError(w, http.StatusServiceUnavailable, st.Message())
    default:                     httpError(w, http.StatusInternalServerError, st.Message())
    }
}
```

---

## 4. gRPC Clients with Circuit Breakers

**File: `internal/adapter/client/fs_client.go`**

```go
type FSClient struct {
    client  fsv1.FileSystemServiceClient
    breaker *resilience.CircuitBreaker
}

func NewFSClient(addr string) (*FSClient, error) {
    conn, err := grpc.Dial(addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10*1024*1024)),
    )
    if err != nil { return nil, err }
    
    return &FSClient{
        client: fsv1.NewFileSystemServiceClient(conn),
        breaker: resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
            Name: "fs", MaxRequests: 3, Interval: 10*time.Second, Timeout: 30*time.Second,
        }),
    }, nil
}

func (c *FSClient) Ls(ctx context.Context, uri string) ([]domain.FileEntry, error) {
    result, err := c.breaker.Execute(func() (interface{}, error) {
        return c.client.Ls(ctx, &fsv1.LsRequest{Uri: uri})
    })
    if err != nil { return nil, err }
    return mapLsResponse(result.(*fsv1.LsResponse)), nil
}

// All FS methods wrapped with circuit breaker
```

---

## 5. Admin Handler (RBAC Required)

**File: `internal/adapter/http/admin_handler.go`**

```go
type AdminHandler struct {
    adminClient port.AdminClient
}

// POST /api/v1/admin/accounts (requires ROOT)
func (h *AdminHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
    if err := requireRole(r.Context(), viking.RoleRoot); err != nil {
        httpError(w, http.StatusForbidden, err.Error())
        return
    }
    var req struct {
        ID   string `json:"id"`
        Name string `json:"name"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    account, err := h.adminClient.CreateAccount(r.Context(), req.ID, req.Name)
    if err != nil { mapGRPCError(w, err); return }
    writeJSON(w, http.StatusCreated, account)
}

// GET /api/v1/admin/accounts (requires ROOT)
func (h *AdminHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
    if err := requireRole(r.Context(), viking.RoleRoot); err != nil {
        httpError(w, http.StatusForbidden, err.Error())
        return
    }
    // ...
}

// POST /api/v1/admin/api-keys (requires ADMIN for own account, ROOT for any)
func (h *AdminHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
    rc, _ := viking.FromContext(r.Context())
    var req struct {
        AccountID string `json:"account_id"`
        UserID    string `json:"user_id"`
        Name      string `json:"name"`
        Role      string `json:"role"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // ADMIN can only create keys for their own account
    if rc.Role < viking.RoleRoot && req.AccountID != rc.User.AccountID {
        httpError(w, http.StatusForbidden, "cannot create keys for other accounts")
        return
    }
    // ...
}

func requireRole(ctx context.Context, minRole viking.Role) error {
    rc, ok := viking.FromContext(ctx)
    if !ok || rc.Role < minRole {
        return fmt.Errorf("insufficient role: requires %v", minRole)
    }
    return nil
}
```

---

## 6. Config

```yaml
gateway:
  http:
    port: 8080
    max_request_size: "10MB"
    timeout: 60s
    read_timeout: 30s
    write_timeout: 60s
    cors:
      allowed_origins: ["*"]
      allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"]
      allowed_headers: ["*"]
  auth:
    mode: "dev"        # dev | api_key | trusted
    root_api_key: ""
  rate_limit:
    enabled: true
    redis_url: "redis://redis:6379/0"
    window: 60s
    requests_per_window: 1000
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

## Unit Tests

```
TestAuthMW_Dev_AcceptsLocalhost         → 127.0.0.1 → ROOT role, no error
TestAuthMW_Dev_Rejects192               → 192.168.x.x → 401
TestAuthMW_APIKey_ResolvesMockKey       → valid key → RequestContext set
TestAuthMW_APIKey_MissingKey_401        → no header → 401
TestAuthMW_Trusted_ReadsHeaders         → X-OpenViking-Account: acct1 → AccountID=acct1
TestAuthMW_Trusted_MissingAccount_401   → no account header → 401
TestParseRoleHeader_Root                → "ROOT" → RoleRoot
TestParseRoleHeader_Unknown             → "" → RoleUser
TestRBAC_AdminRoute_RequiresAdmin       → RoleUser → 403
TestRBAC_AdminRoute_AllowsAdmin         → RoleAdmin → 200
TestRBAC_RootRoute_RequiresRoot         → RoleAdmin → 403
TestRBAC_RootRoute_AllowsRoot           → RoleRoot → 200
TestNamespaceCheck_UserOwnURI           → RoleUser, own URI → no error
TestNamespaceCheck_UserOtherURI         → RoleUser, other user URI → 403
TestFilesystemHandler_Ls_MapsResponse   → FS client returns entries → JSON response
TestFilesystemHandler_Ls_InvalidURI_400 → empty URI → 400
TestFilesystemHandler_Ls_URINotFound_404 → FS returns NotFound → 404
TestCircuitBreaker_FSDown_503           → breaker open → 503 not timeout
TestMapGRPCError_NotFound               → gRPC NotFound → HTTP 404
TestMapGRPCError_PermissionDenied       → gRPC PermissionDenied → HTTP 403
TestRateLimitMW_AllowsUnderLimit        → < 1000/min → 200
TestRateLimitMW_Blocks429               → > 1000/min → 429
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./services/openviking-gateway/...
go test ./services/openviking-gateway/... -v -count=1

# Start gateway in dev mode
cd services/openviking-gateway && go run cmd/server/main.go

# Smoke test
curl -s http://localhost:8080/healthz
# → {"status": "ok"}

curl -s -X POST http://localhost:8080/api/v1/system/status \
  -H "X-OpenViking-Account: test"
# → {"version": "1.0.0", "uptime": "..."}
```

---

## Ghi chú triển khai

- `chi` router: `github.com/go-chi/chi/v5`
- CORS middleware: `github.com/go-chi/cors`
- Rate limit key: API key ID (khi api_key mode) hoặc IP (khi dev mode)
- All handlers sử dụng `req` struct với json tags, validate required fields
- `admin` route group: check role TRƯỚC khi call backend (không waste gRPC call nếu 403)
- System handler: `GET /api/v1/system/status` → `AdminClient.GetAggregatedHealth()`
