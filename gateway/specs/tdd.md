---
id: TDD-vnp-gateway
title: Technical Design — vnp-gateway
service: vnp-gateway
version: 1.0.0
status: Active
created: 2026-05-09
updated: 2026-05-09
group: Platform
---

# Technical Design — vnp-gateway

> **Group**: Platform | **Ports**: 8080 (REST), 8081 (gRPC), 8082 (MCP) | **Origin**: Unified

---

## 1. Service Overview

Single entry point for ALL VNP Memory APIs — routes to 35 domain services across 6 cognitive engines. Handles authentication, rate limiting, protocol translation (REST↔gRPC, MCP, WebDAV), circuit breaking, and content-aware auto-routing.

### Key Metrics

| Metric | Target |
|--------|--------|
| **p99 latency** | < 50ms (gateway overhead only) |
| **Throughput** | 10K req/s per instance |
| **Availability** | 99.99% |
| **Startup time** | < 5s |

---

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── entity.go          # RouteTarget, ProtocolType, AuthContext, TenantContext
├── errors.go          # GatewayError — ErrUnauth, ErrForbidden, ErrRateLimit, ErrCircuitOpen
└── event.go           # RequestReceived, RequestRouted, AuthFailed (NATS events)
```

**Key Domain Types:**
```go
type AuthContext struct {
    TenantID  string
    UserID    string
    Roles     []string
    Scopes    []string
    RateTier  string // "free", "pro", "enterprise"
}

type RouteTarget struct {
    Service   string        // e.g., "cognee-ingestion"
    Address   string        // e.g., "cognee-ingestion:9011"
    Timeout   time.Duration // route-specific timeout
}

type ProtocolType int
const (
    ProtocolREST ProtocolType = iota
    ProtocolGRPC
    ProtocolMCP
    ProtocolWebDAV
    ProtocolWebSocket
)
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── route.go           # RouteUseCase — classify content + resolve target service
├── auth.go            # AuthenticateUseCase — JWT validation, API key resolution
├── mcp.go             # MCPServerUseCase — MCP tool → gRPC dispatch
├── ratelimit.go       # RateLimitUseCase — check + increment sliding window
└── port/
    ├── input.go       # Router, Authenticator, MCPHandler, RateLimiter interfaces
    └── output.go      # ServiceRegistry, TenantStore, KeyStore, EventPublisher
```

**Port Interfaces:**
```go
// input.go — driven by adapters
type Router interface {
    Route(ctx context.Context, req *RouteRequest) (*RouteResult, error)
    Classify(ctx context.Context, data []byte) (string, error)
}

type Authenticator interface {
    AuthenticateJWT(ctx context.Context, token string) (*AuthContext, error)
    AuthenticateAPIKey(ctx context.Context, key string) (*AuthContext, error)
}

// output.go — drives infrastructure
type ServiceRegistry interface {
    Resolve(service string) (*RouteTarget, error)
    Forward(ctx context.Context, target *RouteTarget, req []byte) ([]byte, error)
    HealthCheck(service string) (bool, error)
}

type TenantStore interface {
    GetTenant(ctx context.Context, id string) (*Tenant, error)
}

type KeyStore interface {
    ResolveAPIKey(ctx context.Context, keyHash string) (*AuthContext, error)
}
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── http/              # chi/v5 REST handlers (8 handler files)
├── grpc/              # gRPC-Web proxy
├── mcp/               # MCP SSE/HTTP Streamable server
├── webdav/            # WebDAV proxy → ov-fs
├── ws/                # WebSocket handler
└── client/            # gRPC clients to all 35 services (ServiceRegistry impl)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── config/            # Viper config loader
├── server/            # HTTP/gRPC/MCP server lifecycle
├── middleware/         # 9 middleware components
└── wire/              # Google Wire DI
```

---

## 3. REST API Routing Table

Total: **50+ routes** across 8 namespaces.

| Namespace | Routes | Target Services |
|-----------|--------|----------------|
| `/v1/memory/*` | 4 | Auto-route, vnp-search-hub, vnp-event |
| `/v1/cognee/*` | 4 | cognee-ingestion, cognee-cognify, cognee-search |
| `/v1/graphiti/*` | 4 | graphiti-ingestion, graphiti-search, graphiti-store |
| `/v1/memobase/*` | 5 | memobase-ingestion, memobase-context, vnp-event |
| `/v1/ov/*` | 11 | ov-fs, ov-search, ov-session, ov-resource |
| `/v1/zep/*` | 9 | zep-user, zep-memory, zep-search, zep-graph |
| `/v1/sm/*` | 9 | sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-project |
| `/v1/admin/*` | 4 | vnp-admin |

---

## 4. MCP Server (16 Tools)

| Tool | Description | Target gRPC Service |
|------|-------------|-------------------|
| `memory_store` | Store memory (auto-route) | Auto |
| `memory_recall` | Cross-engine recall | vnp-search-hub |
| `memory_search` | Semantic search | cognee-search |
| `memory_timeline` | Temporal events | vnp-event |
| `memory_profile` | Get user profile | memobase-context |
| `memory_forget` | Delete memory | Fan-out |
| `graph_query` | KG query | graphiti-store |
| `ov_read_file` | Read file | ov-fs |
| `ov_write_file` | Write file | ov-fs |
| `ov_search` | Hierarchical search | ov-search |
| `ov_list_dir` | List directory | ov-fs |
| `ov_grep` | Content grep | ov-fs |
| `ov_tree` | Directory tree | ov-fs |
| `ov_session_commit` | Commit session | ov-session |
| `ov_ingest` | Ingest resource | ov-resource |
| `ov_delete` | Delete file | ov-fs |

---

## 5. Middleware Pipeline

Request flows through middleware in this order:

```
Request → Recovery → RequestID → Logging → CORS → Auth → RateLimit → Timeout → Circuit → Handler
```

| Order | Middleware | Responsibility |
|-------|-----------|---------------|
| 1 | Recovery | Panic → 500 + log stack trace |
| 2 | RequestID | Generate/propagate UUID v7 |
| 3 | Logging | Structured access log (method, path, status, latency) |
| 4 | CORS | Cross-origin policy enforcement |
| 5 | Auth | JWT/APIKey → AuthContext extraction |
| 6 | RateLimit | Redis sliding window check |
| 7 | Timeout | Route-specific deadline propagation |
| 8 | CircuitBreaker | Per-service circuit breaker |
| 9 | Handler | Business logic dispatch |

---

## 6. NATS Events Published

| Subject | Event | When |
|---------|-------|------|
| `gateway.request.received` | `RequestReceived` | Every incoming request |
| `gateway.request.routed` | `RequestRouted` | After successful routing |
| `gateway.auth.failed` | `AuthFailed` | Authentication failure |
| `gateway.ratelimit.exceeded` | `RateLimitExceeded` | Rate limit hit |
| `gateway.circuit.opened` | `CircuitOpened` | Circuit breaker opened |

---

## 7. Observability

- **Metrics**: Prometheus counters/histograms at `:11080/metrics`
- **Traces**: OTel spans for every middleware + downstream call
- **Logs**: Structured JSON via slog with `request_id`, `tenant_id`, `service`, `latency_ms`
- **Health**: HTTP `/healthz` (liveness), `/readyz` (readiness), `/healthz/deep` (full cascade)

---

## 8. Multi-Tenancy

```
HTTP Header: Authorization: Bearer <jwt>
         └─ Decode → tenant_id, user_id, roles
         └─ Set gRPC metadata: x-tenant-id, x-user-id
         └─ All downstream services receive tenant context
         └─ Rate limiting scoped per tenant
```

---

> **Next Steps**: Decompose this TDD into individual FEAT/ARCH specs in `specs/features/` and `specs/architecture/` before implementation. See SOL-001 for the implementation plan.
