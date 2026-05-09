# 01 — Memobase API Gateway

> **REST**: 8080 | **gRPC**: 8081 | **MCP(SSE)**: 8082 | **Health**: 8083

---

## 1. Purpose

Single entry point cho toàn bộ Memobase platform. Handles:
- REST → gRPC translation cho tất cả backend services
- MCP Server-Sent Events (save_memory, get_profiles, search_memories)
- Auth (JWT + API Key + Root Token), Rate Limiting, CORS
- Circuit breaker, observability, multi-tenant (project) resolution

---

## 2. Clean Architecture Layout

```
services/memobase-gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── route.go              # Route, RouteGroup entities
│   │   ├── auth_claims.go        # AuthClaims, ProjectInfo
│   │   ├── project_token.go      # ProjectToken parsing (sk-proj-*)
│   │   └── errors.go             # GatewayError types
│   ├── usecase/
│   │   ├── route_resolver.go     # Resolve REST path → gRPC target
│   │   ├── auth_validator.go     # Validate JWT/APIKey/RootToken → project_id
│   │   ├── rate_limiter.go       # Check rate limit per project+endpoint
│   │   └── port/
│   │       └── output.go         # TokenVerifier, RateLimitStore, ServiceRegistry
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── router.go         # chi/v5 route registration
│   │   │   ├── user_routes.go    # /api/v1/users/*
│   │   │   ├── blob_routes.go    # /api/v1/blobs/*
│   │   │   ├── profile_routes.go # /api/v1/users/profile/*
│   │   │   ├── buffer_routes.go  # /api/v1/users/buffer/*
│   │   │   ├── event_routes.go   # /api/v1/users/event/*
│   │   │   ├── context_routes.go # /api/v1/users/context/*
│   │   │   ├── project_routes.go # /api/v1/project/*
│   │   │   ├── admin_routes.go   # /api/v1/admin/*
│   │   │   ├── health_routes.go  # /healthz, /readyz, /livez
│   │   │   └── middleware/       # HTTP middleware chain
│   │   ├── mcp/
│   │   │   ├── server.go         # MCP SSE transport
│   │   │   └── tools.go          # save_memory, get_profiles, search_memories
│   │   ├── client/
│   │   │   ├── pool.go           # gRPC connection pool manager
│   │   │   ├── ingestion_client.go
│   │   │   ├── context_client.go
│   │   │   ├── event_client.go
│   │   │   └── admin_client.go
│   │   └── repository/
│   │       └── redis/            # Rate limit state, session cache
│   └── infra/
│       ├── config/config.go
│       ├── server/http.go        # HTTP server + graceful shutdown
│       └── wire/wire.go
```

---

## 3. REST API Routing Table

### 3.1 User API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| POST | `/api/v1/users` | memobase-admin | `CreateUser` |
| GET | `/api/v1/users/{user_id}` | memobase-admin | `GetUser` |
| PUT | `/api/v1/users/{user_id}` | memobase-admin | `UpdateUser` |
| DELETE | `/api/v1/users/{user_id}` | memobase-admin | `DeleteUser` |

### 3.2 Blob API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| POST | `/api/v1/blobs/insert/{user_id}` | memobase-ingestion | `InsertBlob` |
| GET | `/api/v1/blobs/{user_id}/{blob_id}` | memobase-ingestion | `GetBlob` |
| DELETE | `/api/v1/blobs/{user_id}/{blob_id}` | memobase-ingestion | `DeleteBlob` |

### 3.3 Profile API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| GET | `/api/v1/users/profile/{user_id}` | memobase-context | `GetProfiles` |
| POST | `/api/v1/users/profile/{user_id}` | memobase-context | `AddProfile` |
| PUT | `/api/v1/users/profile/{user_id}/{profile_id}` | memobase-context | `UpdateProfile` |
| DELETE | `/api/v1/users/profile/{user_id}/{profile_id}` | memobase-context | `DeleteProfile` |

### 3.4 Buffer API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| POST | `/api/v1/users/buffer/{user_id}/{buffer_type}` | memobase-ingestion | `FlushBuffer` |
| GET | `/api/v1/users/buffer/capacity/{user_id}/{buffer_type}` | memobase-ingestion | `GetBufferCapacity` |

### 3.5 Event API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| GET | `/api/v1/users/event/{user_id}` | memobase-event | `GetEvents` |
| PUT | `/api/v1/users/event/{user_id}/{event_id}` | memobase-event | `UpdateEvent` |
| DELETE | `/api/v1/users/event/{user_id}/{event_id}` | memobase-event | `DeleteEvent` |
| GET | `/api/v1/users/event/search/{user_id}` | memobase-event | `SearchEvents` |
| GET | `/api/v1/users/event_gist/search/{user_id}` | memobase-event | `SearchEventGists` |
| GET | `/api/v1/users/event_tags/search/{user_id}` | memobase-event | `FilterByTags` |

### 3.6 Context API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| GET | `/api/v1/users/context/{user_id}` | memobase-context | `GetContext` |

### 3.7 Project API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| POST | `/api/v1/project/profile_config` | memobase-admin | `UpdateProfileConfig` |
| GET | `/api/v1/project/profile_config` | memobase-admin | `GetProfileConfig` |
| GET | `/api/v1/project/billing` | memobase-admin | `GetBilling` |
| GET | `/api/v1/project/users` | memobase-admin | `ListProjectUsers` |
| GET | `/api/v1/project/usage` | memobase-admin | `GetUsage` |

### 3.8 Admin API

| Method | Path | Target | gRPC Method |
|--------|------|--------|-------------|
| GET | `/api/v1/admin/status_check` | memobase-admin | `StatusCheck` |
| GET | `/api/v1/healthcheck` | local | `HealthCheck` (no auth) |

---

## 4. Middleware Chain

```go
r := chi.NewRouter()
r.Use(middleware.RequestID)            // X-Request-ID
r.Use(middleware.RealIP)
r.Use(pkg.OtelTrace)                   // OpenTelemetry span
r.Use(pkg.AccessLog)                   // Structured JSON access log
r.Use(pkg.Recover)                     // Panic recovery → 500
r.Use(middleware.CORS(corsOpts))
r.Use(middleware.Timeout(30*time.Second))
r.Use(gateway.AuthExtract)             // Extract Bearer → project_id
r.Use(gateway.TenantResolve)           // Set project_id in context
r.Use(gateway.RateLimit)               // Redis sliding window
r.Use(gateway.ProcessTimer)            // X-Process-Time header
```

---

## 5. Auth Flow

```
Request → AuthExtract middleware
  │
  ├── Path = /healthcheck → ALLOW (no auth)
  │
  ├── No Bearer token → 401 UNAUTHORIZED
  │
  ├── Token = ROOT_ACCESS_TOKEN env → root access
  │   └── project_id = "__root__", is_root = true
  │
  └── Token = sk-proj-* → ParseProjectToken()
      ├── Extract project_id from token format
      ├── gRPC call → memobase-admin.VerifyProjectSecret()
      ├── Check project_status ≠ "suspended" (403)
      └── Set project_id in gRPC metadata for downstream
```

---

## 6. MCP Server (Model Context Protocol)

```go
tools := []mcp.Tool{
    {
        Name: "save_memory",
        Handler: func(args) {
            // 1. InsertBlob → memobase-ingestion
            // 2. FlushBuffer → memobase-ingestion (sync)
        },
    },
    {
        Name: "get_user_profiles",
        Handler: func(args) {
            // GetProfiles → memobase-context
        },
    },
    {
        Name: "search_memories",
        Handler: func(args) {
            // SearchEvents → memobase-event
        },
    },
}
```

Transport: SSE (`:8082`) | Stdio (CLI mode)

---

## 7. gRPC Client Pool

```go
type ServicePool struct {
    ingestion *grpc.ClientConn  // → memobase-ingestion:9041
    engine    *grpc.ClientConn  // → memobase-engine:9042
    context   *grpc.ClientConn  // → memobase-context:9043
    event     *grpc.ClientConn  // → memobase-event:9044
    admin     *grpc.ClientConn  // → memobase-admin:9045
}
// Each with circuit breaker (gobreaker)
// Keep-alive: 30s interval, 10s timeout
// Max retry: 3 with exponential backoff
```

---

## 8. Configuration

```yaml
gateway:
  http:
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
  mcp:
    port: 8082
    enabled: true
  auth:
    root_token: "${ROOT_ACCESS_TOKEN}"
    jwt:
      issuer: "memobase"
      public_key_path: "/secrets/jwt/public.pem"
    api_key:
      header: "Authorization"
      prefix: "Bearer"
  rate_limit:
    enabled: true
    default_rps: 100
    burst: 200
    redis_url: "redis://redis:6379/0"
  backends:
    ingestion: "memobase-ingestion:9041"
    engine: "memobase-engine:9042"
    context: "memobase-context:9043"
    event: "memobase-event:9044"
    admin: "memobase-admin:9045"
  circuit_breaker:
    max_failures: 5
    timeout: 60s
```
