# graphiti-gateway — API Gateway Service

**Version:** 2.0 | **Date:** 2026-05-09  
**Origin:** Python L7 (FastAPI REST + FastMCP Server)  
**Architecture:** Clean Architecture | **Protocol:** REST + gRPC + MCP + WebSocket

---

## 1. Service Overview

API Gateway là **single entry point** duy nhất cho toàn bộ Graphiti platform. Nó xử lý authentication, authorization, rate limiting, protocol translation, và request routing tới các downstream services qua gRPC.

### Responsibilities

| Concern | Description |
|---------|-------------|
| **Authentication** | JWT validation, API key verification, mTLS for service mesh |
| **Authorization** | RBAC/ABAC policy enforcement per endpoint |
| **Rate Limiting** | Per-tenant, per-endpoint sliding window (Redis-backed) |
| **Protocol Translation** | REST → gRPC, MCP → gRPC, WebSocket → gRPC streaming |
| **Request Routing** | Path-based routing to downstream gRPC services |
| **Circuit Breaking** | Per-downstream-service circuit breakers |
| **Request/Response Transform** | JSON ↔ Protobuf marshaling, pagination, error mapping |
| **CORS** | Cross-origin policy enforcement |
| **Compression** | gzip/brotli response compression |
| **Request Logging** | Structured access logs, correlation ID injection |

---

## 2. Clean Architecture Layout

```
services/graphiti-gateway/
├── cmd/
│   └── server/
│       └── main.go                     # Entry point
├── internal/
│   ├── domain/                         # Layer 1: Entities
│   │   ├── auth.go                     #   AuthToken, Tenant, APIKey models
│   │   ├── route.go                    #   Route, Endpoint definitions
│   │   ├── ratelimit.go                #   RateLimit policy domain model
│   │   └── errors.go                   #   Domain error types
│   ├── usecase/                        # Layer 2: Use Cases
│   │   ├── authenticate.go             #   Validate JWT/API key
│   │   ├── authorize.go                #   Check RBAC/ABAC policies
│   │   ├── route_request.go            #   Route incoming request to service
│   │   ├── rate_limit.go               #   Enforce rate limits
│   │   ├── port/
│   │   │   ├── input.go                #   Use case interfaces
│   │   │   └── output.go              #   TokenValidator, RateLimiterRepo, ServiceRouter
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── http/                       #   REST handlers
│   │   │   ├── router.go              #   chi router setup
│   │   │   ├── v1/                    #   API v1 handlers
│   │   │   │   ├── ingest_handler.go  #     POST /v1/episodes, /v1/triplets
│   │   │   │   ├── search_handler.go  #     GET /v1/search, /v1/episodes
│   │   │   │   ├── entity_handler.go  #     CRUD /v1/entities, /v1/edges
│   │   │   │   ├── admin_handler.go   #     POST /v1/admin/*, /v1/groups
│   │   │   │   └── health_handler.go  #     GET /healthz, /readyz, /livez
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── cors.go
│   │   │   │   ├── ratelimit.go
│   │   │   │   ├── request_id.go
│   │   │   │   ├── logging.go
│   │   │   │   ├── recovery.go
│   │   │   │   └── compression.go
│   │   │   └── mapper/
│   │   │       ├── ingest_mapper.go    #   JSON ↔ gRPC proto mapping
│   │   │       ├── search_mapper.go
│   │   │       └── error_mapper.go     #   gRPC status → HTTP status
│   │   ├── mcp/                        #   MCP Protocol handler
│   │   │   ├── server.go              #   MCP server implementation
│   │   │   ├── tools.go               #   MCP tool definitions (9 tools)
│   │   │   └── mapper.go             #   MCP ↔ gRPC mapping
│   │   ├── ws/                         #   WebSocket handler
│   │   │   ├── handler.go
│   │   │   └── upgrader.go
│   │   ├── grpc_client/               #   Downstream gRPC clients
│   │   │   ├── ingestion_client.go
│   │   │   ├── search_client.go
│   │   │   ├── knowledge_client.go
│   │   │   ├── store_client.go
│   │   │   ├── admin_client.go
│   │   │   └── pool.go               #   Connection pool management
│   │   └── repository/
│   │       ├── redis_ratelimit.go     #   Redis-backed rate limiter
│   │       └── redis_session.go       #   Session/token cache
│   └── infra/
│       ├── config/
│       │   └── config.go               #   Gateway configuration
│       ├── server/
│       │   ├── http.go                 #   HTTP server lifecycle
│       │   └── graceful.go            #   Graceful shutdown
│       ├── telemetry/
│       │   ├── tracer.go
│       │   ├── metrics.go
│       │   └── logger.go
│       └── wire/
│           ├── wire.go
│           └── wire_gen.go
├── api/
│   ├── openapi/
│   │   └── graphiti-api-v1.yaml        # OpenAPI 3.1 spec
│   └── proto/
│       └── gateway/v1/
│           └── gateway.proto           # Gateway-specific messages
├── Dockerfile
├── Makefile
└── README.md
```

---

## 3. REST API Specification

### 3.1 Episode Ingestion

| Endpoint | Method | Description | Downstream |
|----------|--------|-------------|------------|
| `POST /v1/episodes` | POST | Ingest single episode | ingestion.IngestEpisode |
| `POST /v1/episodes/bulk` | POST | Batch episode ingestion | ingestion.IngestEpisodeBulk |
| `GET /v1/episodes` | GET | List recent episodes | ingestion.ListEpisodes |
| `GET /v1/episodes/{uuid}` | GET | Get episode by UUID | store.GetEpisode |
| `DELETE /v1/episodes/{uuid}` | DELETE | Delete episode + cascade | ingestion.RemoveEpisode |

### 3.2 Entity & Edge Management

| Endpoint | Method | Description | Downstream |
|----------|--------|-------------|------------|
| `POST /v1/entities` | POST | Create entity node | store.SaveEntityNode |
| `GET /v1/entities/{uuid}` | GET | Get entity by UUID | store.GetEntityNode |
| `DELETE /v1/entities/{uuid}` | DELETE | Delete entity | store.DeleteEntityNode |
| `POST /v1/triplets` | POST | Add (S,P,O) triple | ingestion.AddTriplet |
| `GET /v1/edges/{uuid}` | GET | Get edge by UUID | store.GetEntityEdge |
| `DELETE /v1/edges/{uuid}` | DELETE | Delete edge | store.DeleteEntityEdge |

### 3.3 Search

| Endpoint | Method | Description | Downstream |
|----------|--------|-------------|------------|
| `POST /v1/search` | POST | Hybrid search | search.Search |
| `POST /v1/search/advanced` | POST | Configurable multi-type search | search.SearchAdvanced |
| `POST /v1/search/nodes` | POST | Node-specific search | search.SearchNodes |
| `POST /v1/search/communities` | POST | Community-level search | search.SearchCommunities |

### 3.4 Admin

| Endpoint | Method | Description | Downstream |
|----------|--------|-------------|------------|
| `POST /v1/admin/indices` | POST | Build indices & constraints | admin.BuildIndices |
| `DELETE /v1/admin/indices` | DELETE | Delete all indices | admin.DeleteIndices |
| `POST /v1/admin/communities` | POST | Rebuild communities | admin.RebuildCommunities |
| `DELETE /v1/groups/{group_id}` | DELETE | Clear group data | admin.ClearGroup |
| `POST /v1/admin/clear` | POST | Clear entire graph | admin.ClearAll |
| `GET /v1/admin/health` | GET | Aggregate health status | admin.HealthCheck |
| `GET /v1/admin/metrics` | GET | Service metrics | admin.GetMetrics |

### 3.5 Health & Status

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /healthz` | GET | Liveness probe |
| `GET /readyz` | GET | Readiness probe (checks all downstream services) |
| `GET /livez` | GET | Startup probe |
| `GET /version` | GET | Build version info |

---

## 4. MCP Protocol Support

### 4.1 MCP Tools (mapped from Python FastMCP)

| MCP Tool | gRPC Call | Description |
|----------|-----------|-------------|
| `add_memory` | ingestion.IngestEpisode | Add episode to graph |
| `search_memory` | search.Search | Hybrid search |
| `get_episodes` | ingestion.ListEpisodes | List recent episodes |
| `delete_episode` | ingestion.RemoveEpisode | Delete episode |
| `delete_entity_node` | store.DeleteEntityNode | Delete entity |
| `delete_entity_edge` | store.DeleteEntityEdge | Delete edge |
| `get_entity_edge` | store.GetEntityEdge | Get edge details |
| `clear_graph` | admin.ClearAll | Clear graph |
| `get_status` | admin.HealthCheck | Server status |

### 4.2 MCP Server Implementation

```go
// internal/adapter/mcp/server.go
type MCPServer struct {
    ingestionClient  pb.IngestionServiceClient
    searchClient     pb.SearchServiceClient
    storeClient      pb.StoreServiceClient
    adminClient      pb.AdminServiceClient
}

// Implements MCP protocol over SSE (Server-Sent Events)
// with JSON-RPC 2.0 message framing
```

---

## 5. Authentication Flow

```
Client Request
  │
  ├─ Authorization: Bearer <JWT>
  │  OR
  ├─ X-API-Key: <key>
  │
  ▼
auth middleware
  │
  ├─ JWT path:
  │  1. Validate signature (RS256, JWKS endpoint)
  │  2. Check expiry, issuer, audience
  │  3. Extract tenant_id from claims
  │  4. Check RBAC roles
  │
  ├─ API Key path:
  │  1. Lookup key in Redis/Vault
  │  2. Validate scope + tenant binding
  │  3. Extract tenant_id
  │
  ▼
Inject tenant_id into context
  │
  ▼
Propagate as gRPC metadata:
  - x-tenant-id: <tenant_id>
  - x-request-id: <uuid>
  - x-trace-id: <otel_trace_id>
```

---

## 6. Rate Limiting

### 6.1 Strategy

| Dimension | Limit | Window |
|-----------|-------|--------|
| Per-tenant global | 1000 req/min | Sliding window |
| Per-tenant per-endpoint | 100 req/min (search), 50 req/min (ingest) | Sliding window |
| Per-IP (unauthenticated) | 30 req/min | Fixed window |
| Burst | 2x sustained limit | 10s burst window |

### 6.2 Implementation

```go
// Redis-backed sliding window
type RateLimiter interface {
    Allow(ctx context.Context, key string, limit Rate) (bool, RateLimitInfo, error)
}

type RateLimitInfo struct {
    Remaining int
    ResetAt   time.Time
    RetryAfter time.Duration
}
```

Headers returned: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After` (429).

---

## 7. Error Mapping

| gRPC Status | HTTP Status | Description |
|-------------|-------------|-------------|
| OK | 200 | Success |
| InvalidArgument | 400 | Bad request |
| Unauthenticated | 401 | Missing/invalid auth |
| PermissionDenied | 403 | Insufficient permissions |
| NotFound | 404 | Resource not found |
| AlreadyExists | 409 | Conflict |
| ResourceExhausted | 429 | Rate limit exceeded |
| Internal | 500 | Internal server error |
| Unavailable | 503 | Service unavailable (circuit open) |
| DeadlineExceeded | 504 | Timeout |

---

## 8. Configuration

```yaml
# config/gateway.yaml
server:
  http_port: 8080
  grpc_port: 8081
  mcp_port: 8082
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 15s

auth:
  jwt:
    issuer: "https://auth.vnp.io"
    audience: "graphiti-api"
    jwks_url: "https://auth.vnp.io/.well-known/jwks.json"
    cache_ttl: 5m
  api_key:
    header: "X-API-Key"
    cache_ttl: 10m

rate_limit:
  enabled: true
  redis_url: "redis://redis:6379/0"
  default_limit: 1000
  default_window: 60s

services:
  ingestion:
    address: "graphiti-ingestion:9001"
    timeout: 60s
    circuit_breaker:
      threshold: 5
      timeout: 30s
  search:
    address: "graphiti-search:9002"
    timeout: 30s
    circuit_breaker:
      threshold: 5
      timeout: 30s
  knowledge:
    address: "graphiti-knowledge:9003"
    timeout: 120s
  store:
    address: "graphiti-store:9004"
    timeout: 30s
  admin:
    address: "graphiti-admin:9005"
    timeout: 60s

cors:
  allowed_origins: ["*"]
  allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
  allowed_headers: ["Authorization", "Content-Type", "X-API-Key", "X-Tenant-ID"]
  max_age: 86400

telemetry:
  otel_endpoint: "otel-collector:4317"
  service_name: "graphiti-gateway"
  log_level: "info"
  log_format: "json"
```

---

## 9. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Custom gateway (not Kong/Envoy)** | Full control over MCP protocol, no vendor dependency, Go-native performance |
| **chi router** | Lightweight, idiomatic Go, composable middleware, zero allocation |
| **REST as primary external protocol** | Widest client compatibility, cacheable |
| **MCP via SSE** | Standard MCP transport for AI agent integration |
| **gRPC for internal** | Type-safe, efficient, bi-directional streaming for bulk operations |
| **Redis for rate limiting** | Distributed counter, sub-millisecond latency |
| **JWT + API Key dual auth** | JWT for user-facing apps, API keys for service-to-service / scripts |
| **Connection pool per downstream** | Reuse gRPC connections, configurable max connections |
