# kgs-gateway — Gateway Upgrade Plan

> **Strategy:** 🔄 UPGRADE  
> **Source:** `gateway/` (existing vnp-gateway)  
> **Target:** `kgs-platform/cmd/gateway/`  
> **Priority:** P0 — Phải hoàn thành trước mọi service khác

---

## 1. Phân Tích Codebase Hiện Tại

### 1.1 Những Gì Đã Có (`gateway/`)

```
gateway/
├── domain/entity.go         ← AuthContext, RouteTarget, ProtocolType, StoreRequest
├── domain/errors.go         ← Domain errors
├── domain/event.go          ← Auth events
├── usecase/auth.go          ← JWT RS256 + API Key authentication ✅
├── usecase/ratelimit.go     ← Rate limiting logic ✅
├── usecase/route.go         ← Route resolution ✅
├── usecase/console.go       ← Console-specific routing ✅
├── adapter/handler/         ← HTTP handlers
├── adapter/client/          ← Downstream service clients
├── adapter/mcp/             ← MCP protocol adapter
├── adapter/webdav/          ← WebDAV proxy
├── infra/                   ← Infrastructure configs
└── proto/                   ← Proto clients for downstream
```

**Đã có sẵn và tốt:**
- JWT RS256 authentication (`usecase/auth.go`) — hoàn chỉnh với RSA public key
- API Key authentication (SHA-256 hash-based)
- Rate limiting theo `rate_tier` (free/pro/enterprise)
- Dev mode bypass (`AUTH_DEV_MODE=true`)
- Event publishing khi auth fail

### 1.2 Những Gì Cần Thêm

| Tính năng | Trạng thái | Ghi chú |
|-----------|-----------|---------|
| API Key format `kgs_xxx` | ❌ Thiếu | Hiện dùng `vnp_xxx` prefix |
| App Context từ registry-service | ❌ Thiếu | Hiện dùng JWT claims |
| Redis cache App Context | ❌ Thiếu | Cần thêm cho performance |
| NATS subscription (key revoke) | ❌ Thiếu | Cần invalidate cache |
| Routing đến 9 services mới | ❌ Thiếu | Chỉ route đến kg-service hiện tại |
| Scope checking per endpoint | ⚠️ Partial | Hiện check roles, chưa check scopes |
| Async audit logging | ❌ Thiếu | Hiện log sync |

---

## 2. Kế Hoạch Upgrade

### 2.1 Thay Đổi Auth Model

**Hiện tại (gateway):** JWT → claims(tenant_id, user_id, roles, scopes)  
**Mới (kgs-gateway):** API Key `kgs_xxx` → registry-service.ValidateAPIKey() → AppContext

```go
// domain/entity.go — Thêm AppContext
type AppContext struct {
    AppID    string            `json:"app_id"`
    TenantID string            `json:"tenant_id"`
    Scopes   []string          `json:"scopes"`
    Quotas   map[string]int64  `json:"quotas"`
}

// Giữ lại AuthContext cho JWT auth (backward compat)
// Thêm AppContext cho API Key auth (mới)
```

**File cần sửa:** `gateway/domain/entity.go`
```diff
+type AppContext struct {
+    AppID    string           `json:"app_id"`
+    TenantID string           `json:"tenant_id"`
+    Scopes   []string         `json:"scopes"`
+    Quotas   map[string]int64 `json:"quotas"`
+}
```

### 2.2 Thêm Registry Service Client

**File mới:** `gateway/adapter/client/registry_client.go`

```go
package client

import (
    "context"
    "crypto/sha256"
    "fmt"
    "time"

    "google.golang.org/grpc"
    registrypb "kgs-platform/api/registry/v1"
    "github.com/redis/go-redis/v9"
)

type RegistryClient struct {
    grpcConn  *grpc.ClientConn
    client    registrypb.RegistryServiceClient
    redisCli  *redis.Client
    cacheTTL  time.Duration
}

// ValidateAPIKey calls registry-service and caches result in Redis
func (c *RegistryClient) ValidateAPIKey(ctx context.Context, plainKey string) (*AppContext, error) {
    hash := fmt.Sprintf("%x", sha256.Sum256([]byte(plainKey)))
    
    // Cache lookup first
    cacheKey := fmt.Sprintf("gateway:ctx:%s", hash)
    if cached := c.redisCli.Get(ctx, cacheKey); cached.Err() == nil {
        // Unmarshal cached AppContext
        ...
    }
    
    // Call registry-service
    resp, err := c.client.ValidateAPIKey(ctx, &registrypb.ValidateAPIKeyRequest{
        KeyHash: hash,
    })
    if err != nil {
        return nil, err
    }
    
    // Cache result
    c.redisCli.Set(ctx, cacheKey, marshal(resp), c.cacheTTL)
    
    return toAppContext(resp), nil
}
```

### 2.3 Cập Nhật Auth Middleware

**File sửa:** `gateway/usecase/auth.go`

```go
// AuthenticateKGSKey validates kgs_ prefixed API keys via registry-service
func (uc *AuthUseCase) AuthenticateKGSKey(ctx context.Context, key string) (*domain.AppContext, error) {
    if !strings.HasPrefix(key, "kgs_") {
        return nil, domain.ErrUnauthenticated.WithMessage("invalid KGS API key format")
    }
    
    appCtx, err := uc.registryClient.ValidateAPIKey(ctx, key)
    if err != nil {
        uc.publisher.Publish(ctx, domain.SubjectAuthFailed, domain.AuthFailed{
            Reason:    "kgs_key_invalid",
            Timestamp: time.Now(),
        })
        return nil, domain.ErrUnauthenticated.WithMessage("invalid KGS API key")
    }
    
    return appCtx, nil
}
```

### 2.4 Routing Table Update

**File sửa:** `gateway/usecase/route.go`

Thêm routing cho tất cả 9 services mới:

```go
var kgsRoutes = []RouteRule{
    // Registry Service (9001)
    {Method: "POST", Path: "/v1/apps",                   Backend: "registry",    Scope: "admin"},
    {Method: "GET",  Path: "/v1/apps",                   Backend: "registry",    Scope: "admin"},
    {Method: "GET",  Path: "/v1/apps/:app_id",           Backend: "registry",    Scope: "admin"},
    {Method: "POST", Path: "/v1/apps/:app_id/keys",      Backend: "registry",    Scope: "admin"},
    // ...
    
    // Ontology Service (9002)
    {Method: "POST", Path: "/v1/ontology/entity-types",  Backend: "ontology",    Scope: "ontology:write"},
    {Method: "GET",  Path: "/v1/ontology/entity-types",  Backend: "ontology",    Scope: "ontology:read"},
    // ...
    
    // Graph Service (9003)
    {Method: "POST", Path: "/v1/graph/nodes",            Backend: "graph",       Scope: "graph:write"},
    {Method: "GET",  Path: "/v1/graph/nodes/:id",        Backend: "graph",       Scope: "graph:read"},
    // ...
    
    // Query Intel Service (9004)
    {Method: "GET",  Path: "/v1/graph/nodes/:id/context", Backend: "query-intel", Scope: "graph:read"},
    {Method: "POST", Path: "/v1/query",                   Backend: "query-intel", Scope: "graph:read"},
    // ...
    
    // Rule Engine Service (9005)
    {Method: "POST", Path: "/v1/rules",                  Backend: "rule-engine", Scope: "rules:write"},
    // ...
    
    // Policy Service (9006)
    {Method: "POST", Path: "/v1/policies",               Backend: "policy",      Scope: "policies:write"},
    // ...
    
    // Search Service (9007)
    {Method: "POST", Path: "/v1/search",                 Backend: "search",      Scope: "graph:read"},
    // ...
    
    // Overlay Service (9008)
    {Method: "POST", Path: "/v1/overlay",                Backend: "overlay",     Scope: "graph:write"},
    // ...
}
```

### 2.5 Rate Limiting Upgrade

**File sửa:** `gateway/usecase/ratelimit.go`

Hiện tại rate limit theo `rate_tier` (free/pro/enterprise). Upgrade sang per `app_id` với quota từ AppContext:

```go
// Thêm method mới cho KGS rate limiting
func (uc *RateLimitUseCase) CheckKGSQuota(ctx context.Context, appID string, quota int64) (bool, int64, error) {
    key := fmt.Sprintf("gateway:rl:%s", appID)
    // Token bucket algorithm với Redis
    // quota = appContext.Quotas["requests_per_minute"]
    ...
}
```

### 2.6 NATS Integration (Cache Invalidation)

**File mới:** `gateway/infra/nats_subscriber.go`

```go
// Subscribe để invalidate cache khi API Key bị revoke
func (s *NATSSubscriber) Start(ctx context.Context) {
    s.nats.Subscribe("registry.apikey.revoked", func(msg *nats.Msg) {
        var event struct{ KeyHash string `json:"key_hash"` }
        json.Unmarshal(msg.Data, &event)
        
        cacheKey := fmt.Sprintf("gateway:ctx:%s", event.KeyHash)
        s.redis.Del(ctx, cacheKey)
        
        s.log.Info("gateway: invalidated cache for revoked key", "key_prefix", event.KeyHash[:8])
    })
}
```

---

## 3. Cấu Trúc File Sau Upgrade

```
kgs-platform/cmd/gateway/          ← Entry point mới
gateway/
├── domain/
│   ├── entity.go                  ← + AppContext struct
│   ├── errors.go                  ← Unchanged
│   └── event.go                   ← + KeyRevoked event
├── usecase/
│   ├── auth.go                    ← + AuthenticateKGSKey()
│   ├── ratelimit.go               ← + CheckKGSQuota()
│   ├── route.go                   ← + kgsRoutes table
│   └── console.go                 ← Unchanged (backward compat)
├── adapter/
│   ├── client/
│   │   ├── registry_client.go     ← NEW: gRPC client cho registry-service
│   │   ├── graph_client.go        ← NEW: gRPC client cho graph-service
│   │   └── ...                    ← NEW: clients cho các services khác
│   └── handler/
│       ├── kgs_handler.go         ← NEW: KGS routing handler
│       └── ...                    ← Existing handlers
└── infra/
    ├── nats_subscriber.go         ← NEW: NATS cache invalidation
    └── ...                        ← Existing infra
```

---

## 4. Backward Compatibility

| Endpoint cũ | Endpoint mới | Status |
|-------------|-------------|--------|
| `/v1/graphiti/**` | Giữ nguyên | ✅ Forward to graph-service |
| `/v1/cognee/**` | Giữ nguyên | ✅ Forward to graph-service |
| `/v1/console/graph/**` | Giữ nguyên | ✅ Forward to query-intel-service |
| `/v1/console/adaptive/**` | Giữ nguyên | ✅ Forward to graph-service |
| `X-Tenant-ID` header | Thêm `X-App-ID` | ✅ Cả hai đều support |

---

## 5. Configuration

```yaml
# configs/gateway.yaml
gateway:
  port: 8080
  
  # Dual auth mode: support cả JWT (cũ) và API Key (mới)
  auth:
    jwt:
      enabled: true
      pub_key_file: /certs/jwt.pub
      issuer: vnp-platform
      audience: kgs-api
    api_key:
      enabled: true
      prefix: "kgs_"
      cache_ttl: 60s
  
  # Rate limiting per app_id
  rate_limit:
    redis_addr: redis:6379
    default_rpm: 100
    use_app_quota: true  # Use quota from registry-service
  
  # Backend services
  backends:
    registry_service:  registry-service:9001
    ontology_service:  ontology-service:9002
    graph_service:     graph-service:9003
    query_intel:       query-intel-service:9004
    rule_engine:       rule-engine-service:9005
    policy_service:    policy-service:9006
    search_service:    search-service:9007
    overlay_service:   overlay-service:9008
  
  # Legacy backends (backward compat)
  legacy_backends:
    kg_service:        kg-service:8080  # Current monolith
  
  # NATS
  nats:
    addr: nats:4222
    subscriptions:
      - registry.apikey.revoked
  
  # Audit
  audit:
    enabled: true
    async: true
    registry_service: registry-service:9001  # Write audit to registry
  
  observability:
    metrics_port: 9090
    tracing_endpoint: http://jaeger:14268/api/traces
```

---

## 6. Migration Strategy

### Bước 1: Feature Flag
Deploy gateway với feature flag `KGS_NEW_ROUTING=false`. Mặc định route đến monolith.

### Bước 2: Partial Routing
Khi từng service được deploy: bật routing cho service đó.

```go
if os.Getenv("KGS_NEW_ROUTING") == "true" {
    // Route to new services
} else {
    // Route to legacy monolith
}
```

### Bước 3: Full Migration
Khi tất cả services sẵn sàng: bật `KGS_NEW_ROUTING=true`.

### Bước 4: Remove Legacy
Sau 30 ngày stable: remove legacy routing code.

---

## 7. Observability

```yaml
Metrics (Prometheus):
- gateway_requests_total{app_id, method, path, status}
- gateway_request_duration_seconds{path}
- gateway_rate_limit_blocked_total{app_id}
- gateway_auth_cache_hits_total
- gateway_auth_failures_total{reason}

Traces (OpenTelemetry):
- gateway → registry-service (auth lookup)
- gateway → downstream service (proxy)

Logs (slog):
- Structured JSON logs
- Auth success/failure events
- Rate limit events
```

---

## 8. Lý Do Upgrade Thay Vì Viết Mới

1. **gateway/** đã có JWT + API Key auth rất tốt** — không cần viết lại
2. **Rate limiting logic** đã battle-tested với Redis token bucket
3. **Dev mode** (`AUTH_DEV_MODE=true`) giúp development nhanh
4. **Middleware pipeline** (chi router) đã được optimize
5. Chỉ cần **thêm** registry-service client và cập nhật routing table

**Ước tính effort:** 3-4 ngày (vs 10+ ngày viết mới)
