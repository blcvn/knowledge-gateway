# 01 — Zep API Gateway (zep-gateway)

> **REST**: 8080 | **gRPC**: 8081 | **MCP(SSE/stdio)**: 8082 | **Health**: 8083

---

## 1. Purpose

Single entry point cho toàn bộ Zep Context Engineering Platform. Handles:
- REST → gRPC translation cho tất cả 6 backend services
- MCP Server (13 read-only tools) cho IDE/Agent integration
- Auth (JWT + API Key + SharedSecret), Rate Limiting, CORS
- Request validation, 5MB payload limit, 30s timeout
- Circuit breaker, observability, multi-tenant resolution

---

## 2. Clean Architecture Layout

```
services/zep-gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── route.go              # Route, RouteGroup entities
│   │   ├── auth_claims.go        # AuthClaims, ProjectInfo
│   │   ├── mcp_tool.go           # MCPTool, MCPToolPhase entities
│   │   └── errors.go             # GatewayError types
│   ├── usecase/
│   │   ├── route_resolver.go     # Resolve REST path → gRPC target
│   │   ├── auth_validator.go     # Validate JWT/APIKey/SharedSecret
│   │   ├── rate_limiter.go       # Check rate limit for tenant+endpoint
│   │   ├── mcp_dispatcher.go     # Dispatch MCP tool calls to gRPC
│   │   └── port/
│   │       └── output.go         # TokenVerifier, RateLimitStore, ServiceRegistry
│   ├── adapter/
│   │   ├── http/                  # chi/v5 router, REST handlers
│   │   │   ├── router.go         # Route registration
│   │   │   ├── session_routes.go # /api/v2/sessions/*
│   │   │   ├── memory_routes.go  # /api/v2/sessions/{id}/memory
│   │   │   ├── message_routes.go # /api/v2/sessions/{id}/messages/*
│   │   │   ├── user_routes.go    # /api/v2/users/*
│   │   │   ├── fact_routes.go    # /api/v2/facts/*
│   │   │   ├── search_routes.go  # /api/v2/sessions/search, graph.search()
│   │   │   ├── health_routes.go  # /healthz, /readyz, /livez
│   │   │   └── middleware/       # HTTP middleware chain
│   │   │       ├── cors.go
│   │   │       ├── request_log.go
│   │   │       ├── heartbeat.go
│   │   │       ├── size_limiter.go
│   │   │       ├── request_id.go
│   │   │       ├── timeout.go
│   │   │       ├── real_ip.go
│   │   │       ├── clean_path.go
│   │   │       ├── version_header.go
│   │   │       └── otel_trace.go
│   │   ├── mcp/                   # MCP Server (13 read-only tools)
│   │   │   ├── server.go         # MCP protocol handler (stdio + HTTP)
│   │   │   ├── tool_registry.go  # Register 13 tools → gRPC backends
│   │   │   └── handlers/
│   │   │       ├── search.go     # search_graph, get_user_context
│   │   │       ├── graph.go      # get_user_nodes, get_user_edges, get_episodes
│   │   │       ├── detail.go     # get_node, get_edge, get_episode
│   │   │       └── thread.go     # list_threads, get_thread_messages
│   │   ├── grpc/                  # gRPC-Web proxy (optional)
│   │   ├── client/                # gRPC clients to backend services
│   │   │   ├── pool.go           # Connection pool manager
│   │   │   ├── user_client.go
│   │   │   ├── thread_client.go
│   │   │   ├── memory_client.go
│   │   │   ├── graph_client.go
│   │   │   ├── search_client.go
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

### 3.1 Session Routes (`/api/v2`)

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| GET | `/api/v2/sessions` | zep-thread | `ListSessions` |
| POST | `/api/v2/sessions` | zep-thread | `CreateSession` |
| GET | `/api/v2/sessions-ordered` | zep-thread | `ListOrderedSessions` |
| POST | `/api/v2/sessions/search` | zep-search | `SearchSessions` |
| GET | `/api/v2/sessions/{sessionId}` | zep-thread | `GetSession` |
| PATCH | `/api/v2/sessions/{sessionId}` | zep-thread | `UpdateSession` |

### 3.2 Memory Routes

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| GET | `/api/v2/sessions/{id}/memory` | zep-memory | `GetMemory` |
| POST | `/api/v2/sessions/{id}/memory` | zep-memory | `PutMemory` |
| DELETE | `/api/v2/sessions/{id}/memory` | zep-memory | `DeleteMemory` |

### 3.3 Message Routes

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| GET | `/api/v2/sessions/{id}/messages` | zep-memory | `GetMessagesForSession` |
| GET | `/api/v2/sessions/{id}/messages/{uuid}` | zep-memory | `GetMessage` |
| PATCH | `/api/v2/sessions/{id}/messages/{uuid}` | zep-memory | `UpdateMessageMetadata` |

### 3.4 User Routes

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| POST | `/api/v2/users` | zep-user | `CreateUser` |
| GET | `/api/v2/users` | zep-user | `ListAllUsers` |
| GET | `/api/v2/users-ordered` | zep-user | `ListAllOrderedUsers` |
| GET | `/api/v2/users/{userId}` | zep-user | `GetUser` |
| PATCH | `/api/v2/users/{userId}` | zep-user | `UpdateUser` |
| DELETE | `/api/v2/users/{userId}` | zep-user | `DeleteUser` |
| GET | `/api/v2/users/{userId}/sessions` | zep-thread | `ListUserSessions` |

### 3.5 Fact Routes

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| GET | `/api/v2/facts/{factUUID}` | zep-graph | `GetFact` |
| DELETE | `/api/v2/facts/{factUUID}` | zep-graph | `DeleteFact` |

### 3.6 Graph Routes (SDK-compatible)

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| POST | `/api/v2/graph/search` | zep-search | `GraphSearch` |
| POST | `/api/v2/graph/add` | zep-graph | `AddGraphData` |
| PUT | `/api/v2/graph/ontology` | zep-graph | `SetOntology` |

### 3.7 Admin Routes

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| GET | `/api/v2/admin/health` | zep-admin | `AggregatedHealth` |
| POST | `/api/v2/admin/projects` | zep-admin | `CreateProject` |
| GET | `/api/v2/admin/projects` | zep-admin | `ListProjects` |
| POST | `/api/v2/admin/apikeys` | zep-admin | `CreateAPIKey` |

---

## 4. Middleware Chain (HTTP)

Bảo tồn nguyên vẹn middleware stack 10 layers từ legacy CE, mở rộng thêm auth/tenant/rate-limit:

```go
r := chi.NewRouter()

// 1. CORS (AllowOriginFunc: configurable, default allow all)
r.Use(corsMiddleware(cfg.CORS))

// 2. Request Logging (structured: proto, method, path, request_id, duration, status, response_size)
r.Use(middleware.AccessLog(logger))

// 3. Heartbeat (/healthz → 200 OK)
r.Use(middleware.Heartbeat("/healthz"))

// 4. Request Size Limiter (5MB = 5 << 20)
r.Use(middleware.RequestSizeLimiter(5 << 20))

// 5. Request ID Injection (X-Request-Id header or UUID)
r.Use(middleware.RequestID)

// 6. Context Timeout (30s)
r.Use(middleware.Timeout(30 * time.Second))

// 7. Real IP Extraction
r.Use(middleware.RealIP)

// 8. Clean Path
r.Use(middleware.CleanPath)

// 9. Version Header (X-Zep-Version)
r.Use(middleware.SendVersion(version))

// 10. OpenTelemetry Tracing (otelchi)
r.Use(otelchi.Middleware("zep-gateway"))

// 11. Auth Extract (JWT/APIKey/SharedSecret → context)
r.Use(middleware.AuthExtract(authValidator))

// 12. Tenant Resolve (project_uuid from auth token)
r.Use(middleware.TenantResolve)

// 13. Rate Limit (Redis sliding window, per-project per-endpoint)
r.Use(middleware.RateLimit(rateLimitStore))
```

---

## 5. MCP Server (13 Read-Only Tools)

Gateway integrates the MCP server, proxying tool calls to gRPC backends:

### 5.1 Transport Modes

| Mode | Flag | Protocol | Use Case |
|------|------|----------|----------|
| stdio | `--stdio` | stdin/stdout JSON-RPC | Claude Desktop, Cline |
| HTTP | default (port 8082) | MCP Streamable HTTP (2025-03-26 spec) | Claude Code, web clients |

### 5.2 Tool Registry (3 Phases)

**Phase 1 — Core Search & Retrieval:**

| Tool | Target Service | gRPC Method |
|------|---------------|-------------|
| `search_graph` | zep-search | `GraphSearch` |
| `get_user_context` | zep-memory | `GetUserContext` |
| `get_user` | zep-user | `GetUser` |
| `list_threads` | zep-thread | `ListUserSessions` |

**Phase 2 — Graph Query:**

| Tool | Target Service | gRPC Method |
|------|---------------|-------------|
| `get_user_nodes` | zep-graph | `GetUserNodes` |
| `get_user_edges` | zep-graph | `GetUserEdges` |
| `get_episodes` | zep-graph | `GetEpisodes` |

**Phase 3 — Detail Retrieval:**

| Tool | Target Service | gRPC Method |
|------|---------------|-------------|
| `get_thread_messages` | zep-memory | `GetMessagesForSession` |
| `get_node` | zep-graph | `GetNode` |
| `get_edge` | zep-graph | `GetEdge` |
| `get_episode` | zep-graph | `GetEpisode` |
| `get_node_edges` | zep-graph | `GetNodeEdges` |
| `get_episode_mentions` | zep-graph | `GetEpisodeMentions` |

### 5.3 MCP Tool Handler Pattern

```go
func (s *MCPServer) searchGraph(ctx context.Context, input SearchGraphInput) (*mcp.Result, error) {
    // 1. Validate input
    if err := validate(input); err != nil {
        return nil, mcp.InvalidParams(err.Error())
    }
    
    // 2. Call gRPC backend
    resp, err := s.searchClient.GraphSearch(ctx, &searchpb.GraphSearchRequest{
        Query:       input.Query,
        UserID:      input.UserID,
        Scope:       mapScope(input.Scope),
        Reranker:    mapReranker(input.Reranker),
        NodeLabels:  input.NodeLabels,
        EdgeTypes:   input.EdgeTypes,
        Limit:       input.Limit,
    })
    if err != nil {
        return nil, mcp.InternalError(err.Error())
    }
    
    // 3. Format JSON response
    return mcp.NewResult(resp), nil
}
```

---

## 6. gRPC Client Pool

```go
type ServicePool struct {
    user     *grpc.ClientConn  // → zep-user:9041
    thread   *grpc.ClientConn  // → zep-thread:9042
    memory   *grpc.ClientConn  // → zep-memory:9043
    graph    *grpc.ClientConn  // → zep-graph:9044
    search   *grpc.ClientConn  // → zep-search:9045
    admin    *grpc.ClientConn  // → zep-admin:9046
}

// Each connection wrapped with:
// - Circuit breaker (sony/gobreaker): max 5 failures, 60s timeout
// - Keep-alive: 30s ping, 10s timeout
// - Retry: 3 attempts with exponential backoff
// - OTel interceptor for trace propagation
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
    max_request_size: 5242880  # 5MB
  grpc:
    port: 8081
  mcp:
    port: 8082
    enabled: true
    stdio: false
  auth:
    mode: "apikey"  # "jwt" | "apikey" | "shared_secret"
    jwt:
      issuer: "zep-platform"
      audience: "zep-api"
      public_key_path: "/secrets/jwt/public.pem"
    api_key:
      header: "Authorization"
    shared_secret:
      key: "${ZEP_API_SECRET}"
  cors:
    allow_all_origins: true
    allowed_methods: ["GET", "POST", "PATCH", "DELETE"]
    allowed_headers: ["Authorization", "Content-Type", "X-Request-Id"]
  rate_limit:
    enabled: true
    default_rps: 100
    burst: 200
    redis_url: "redis://redis:6379/0"
  backends:
    user: "zep-user:9041"
    thread: "zep-thread:9042"
    memory: "zep-memory:9043"
    graph: "zep-graph:9044"
    search: "zep-search:9045"
    admin: "zep-admin:9046"
  circuit_breaker:
    max_failures: 5
    timeout: 60s
    half_open_max: 3
  telemetry:
    disabled: false
    organization_name: ""
```
