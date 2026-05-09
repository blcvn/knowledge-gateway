# 01 — Unified API Gateway (vnp-gateway)

> **gRPC**: 8081 | **REST**: 8080 | **MCP(SSE)**: 8082 | **Health**: 8083

---

## 1. Purpose

Single entry point cho toàn bộ VNP Cognitive Platform. Handles:
- REST → gRPC translation (Cognee + Graphiti)
- MCP Server-Sent Events (MCP protocol for IDE/Agent integration)
- Auth (JWT + API Key), Rate Limiting, CORS, Request Validation
- Circuit breaker, observability, multi-tenant resolution

---

## 2. Clean Architecture Layout

```
services/vnp-gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── route.go              # Route, RouteGroup entities
│   │   ├── auth_claims.go        # AuthClaims, TenantInfo
│   │   └── errors.go             # GatewayError types
│   ├── usecase/
│   │   ├── route_resolver.go     # Resolve REST path → gRPC target
│   │   ├── auth_validator.go     # Validate JWT/API Key, extract tenant
│   │   ├── rate_limiter.go       # Check rate limit for tenant+endpoint
│   │   └── port/
│   │       └── output.go         # TokenVerifier, RateLimitStore, ServiceRegistry
│   ├── adapter/
│   │   ├── http/                  # chi/v5 router, REST handlers
│   │   │   ├── router.go         # Route registration
│   │   │   ├── cognee_routes.go  # /api/v1/cognee/*, /api/v2/memory/*
│   │   │   ├── graphiti_routes.go # /api/v1/graphiti/*
│   │   │   ├── admin_routes.go   # /api/v1/admin/*
│   │   │   ├── health_routes.go  # /healthz, /readyz, /livez
│   │   │   └── middleware/       # HTTP middleware chain
│   │   ├── mcp/                   # MCP SSE server
│   │   │   ├── server.go
│   │   │   └── tool_registry.go  # Register MCP tools → gRPC backends
│   │   ├── grpc/                  # gRPC-Web proxy (optional)
│   │   ├── client/                # gRPC clients to backend services
│   │   │   ├── pool.go           # Connection pool manager
│   │   │   ├── cognee_clients.go
│   │   │   ├── graphiti_clients.go
│   │   │   └── admin_client.go
│   │   └── repository/
│   │       └── redis/             # Rate limit state, session cache
│   └── infra/
│       ├── config/config.go
│       ├── server/http.go         # HTTP server + graceful shutdown
│       └── wire/wire.go
```

---

## 3. REST API Routing Table

### 3.1 Cognee API (V1)

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| POST | `/api/v1/cognee/add` | cognee-ingestion | `AddData` |
| GET | `/api/v1/cognee/datasets` | cognee-ingestion | `ListDatasets` |
| DELETE | `/api/v1/cognee/datasets/{id}` | cognee-ingestion | `DeleteDataset` |
| POST | `/api/v1/cognee/cognify` | cognee-cognify | `StartCognify` |
| GET | `/api/v1/cognee/cognify/{id}/status` | cognee-cognify | `GetPipelineStatus` |
| POST | `/api/v1/cognee/search` | cognee-search | `Search` |
| GET | `/api/v1/cognee/search/explore` | cognee-search | `ExploreGraph` |

### 3.2 Memory API (V2)

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| POST | `/api/v2/memory/remember` | cognee-memory | `Remember` |
| POST | `/api/v2/memory/recall` | cognee-memory | `Recall` |
| POST | `/api/v2/memory/forget` | cognee-memory | `Forget` |
| GET | `/api/v2/memory/sessions` | cognee-memory | `ListSessions` |
| POST | `/api/v2/memory/sessions/{id}/persist` | cognee-memory | `PersistSession` |

### 3.3 Graphiti API (V1)

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| POST | `/api/v1/graphiti/episodes` | graphiti-ingestion | `IngestEpisode` |
| GET | `/api/v1/graphiti/episodes` | graphiti-ingestion | `ListEpisodes` |
| POST | `/api/v1/graphiti/search` | graphiti-search | `HybridSearch` |
| GET | `/api/v1/graphiti/entities/{id}` | graphiti-store | `GetEntity` |
| PUT | `/api/v1/graphiti/entities/{id}` | graphiti-store | `UpdateEntity` |
| GET | `/api/v1/graphiti/edges` | graphiti-store | `ListEdges` |

### 3.4 Admin API (V1)

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| GET | `/api/v1/admin/health` | vnp-admin | `AggregatedHealth` |
| POST | `/api/v1/admin/tenants` | vnp-admin | `CreateTenant` |
| GET | `/api/v1/admin/users` | vnp-admin | `ListUsers` |
| POST | `/api/v1/admin/apikeys` | vnp-admin | `CreateAPIKey` |

---

## 4. Middleware Chain (HTTP)

```go
r := chi.NewRouter()
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(middleware.OtelTrace)         // pkg/middleware/tracing
r.Use(middleware.AccessLog)          // pkg/middleware/logging
r.Use(middleware.Recover)            // pkg/middleware/recovery
r.Use(middleware.CORS)
r.Use(middleware.Timeout(30*time.Second))
r.Use(middleware.AuthExtract)        // Extract JWT/APIKey → context
r.Use(middleware.TenantResolve)      // Resolve tenant_id from token
r.Use(middleware.RateLimit)          // Redis sliding window
```

---

## 5. MCP Server (Model Context Protocol)

Gateway acts as an MCP server for IDE/Agent integration:

```go
// MCP tools registered, each proxies to gRPC backend
tools := []mcp.Tool{
    {Name: "cognee_add", Handler: proxyCogneeIngestion},
    {Name: "cognee_search", Handler: proxyCogneeSearch},
    {Name: "memory_remember", Handler: proxyCogneeMemory},
    {Name: "memory_recall", Handler: proxyCogneeMemory},
    {Name: "graphiti_ingest", Handler: proxyGraphitiIngestion},
    {Name: "graphiti_search", Handler: proxyGraphitiSearch},
}
```

---

## 6. gRPC Client Pool

```go
// Connection pool per backend service with circuit breaker
type ServicePool struct {
    cogneeIngestion  *grpc.ClientConn  // → cognee-ingestion:9011
    cogneeCognify    *grpc.ClientConn  // → cognee-cognify:9012
    cogneeSearch     *grpc.ClientConn  // → cognee-search:9013
    cogneeMemory     *grpc.ClientConn  // → cognee-memory:9014
    graphitiIngestion *grpc.ClientConn // → graphiti-ingestion:9021
    graphitiSearch   *grpc.ClientConn  // → graphiti-search:9022
    graphitiKnowledge *grpc.ClientConn // → graphiti-knowledge:9023
    graphitiStore    *grpc.ClientConn  // → graphiti-store:9024
    admin            *grpc.ClientConn  // → vnp-admin:9030
}
```

---

## 7. Configuration

```yaml
gateway:
  http:
    port: 8080
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s
  grpc:
    port: 8081
  mcp:
    port: 8082
    enabled: true
  auth:
    jwt:
      issuer: "vnp-cognitive"
      audience: "vnp-api"
      public_key_path: "/secrets/jwt/public.pem"
    api_key:
      header: "X-API-Key"
  rate_limit:
    enabled: true
    default_rps: 100
    burst: 200
    redis_url: "redis://redis:6379/0"
  backends:
    cognee_ingestion: "cognee-ingestion:9011"
    cognee_cognify: "cognee-cognify:9012"
    cognee_search: "cognee-search:9013"
    cognee_memory: "cognee-memory:9014"
    graphiti_ingestion: "graphiti-ingestion:9021"
    graphiti_search: "graphiti-search:9022"
    graphiti_knowledge: "graphiti-knowledge:9023"
    graphiti_store: "graphiti-store:9024"
    admin: "vnp-admin:9030"
  circuit_breaker:
    max_failures: 5
    timeout: 60s
    half_open_max: 3
```
