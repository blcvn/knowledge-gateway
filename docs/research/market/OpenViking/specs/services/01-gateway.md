# 01 — OpenViking Gateway Service

> **Service**: `openviking-gateway`  
> **Port**: 8080 (REST) · 8081 (gRPC) · 8082 (MCP)  
> **Origin**: L1 Presentation Layer  
> **Role**: Single entry point — REST API (17 route groups), MCP (9 tools), WebDAV, Auth

---

## 1. Responsibilities

| Capability | Description |
|-----------|-------------|
| **REST API** | 17 route groups mapping to downstream gRPC services |
| **MCP Server** | 9 MCP tools via Streamable HTTP at `/mcp` |
| **WebDAV** | Standard WebDAV proxy at `/webdav` |
| **Authentication** | 3 modes: DEV / API_KEY / TRUSTED |
| **Authorization** | RBAC (ROOT > ADMIN > USER) + namespace access check |
| **Rate Limiting** | Per-tenant, per-endpoint Redis sliding window |
| **Protocol Translation** | REST ↔ gRPC, MCP ↔ gRPC, WebDAV ↔ gRPC |
| **Circuit Breaker** | Per-downstream-service circuit breaker |
| **Request Validation** | OpenAPI schema validation before routing |
| **Observability** | Request timing, CORS, OTel trace injection |

---

## 2. Clean Architecture Layout

```
services/openviking-gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── route.go                    # Route definition types
│   │   ├── auth_mode.go                # DEV/API_KEY/TRUSTED enum
│   │   └── errors.go                   # Gateway-specific errors
│   ├── usecase/
│   │   ├── authenticate.go             # Auth resolution logic
│   │   ├── authorize.go                # RBAC + namespace check
│   │   ├── route_request.go            # Route to downstream service
│   │   ├── port/
│   │   │   ├── input.go               # AuthUseCase, RouteUseCase interfaces
│   │   │   └── output.go             # APIKeyStore, ServiceRegistry ports
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/
│   │   ├── http/                       # REST handlers (17 route groups)
│   │   │   ├── router.go              #   chi router setup
│   │   │   ├── filesystem_handler.go  #   /api/v1/filesystem
│   │   │   ├── content_handler.go     #   /api/v1/content
│   │   │   ├── search_handler.go      #   /api/v1/search
│   │   │   ├── sessions_handler.go    #   /api/v1/sessions
│   │   │   ├── resources_handler.go   #   /api/v1/resources
│   │   │   ├── relations_handler.go   #   /api/v1/relations
│   │   │   ├── admin_handler.go       #   /api/v1/admin
│   │   │   ├── observer_handler.go    #   /api/v1/observer
│   │   │   ├── privacy_handler.go     #   /api/v1/privacy-configs
│   │   │   ├── tasks_handler.go       #   /api/v1/tasks
│   │   │   ├── system_handler.go      #   /api/v1/system
│   │   │   ├── debug_handler.go       #   /api/v1/debug
│   │   │   ├── bot_handler.go         #   /api/v1/bot
│   │   │   ├── pack_handler.go        #   /api/v1/pack
│   │   │   ├── maintenance_handler.go #   /api/v1/maintenance
│   │   │   ├── stats_handler.go       #   /api/v1/stats
│   │   │   └── metrics_handler.go     #   /metrics
│   │   ├── mcp/                        # MCP Streamable HTTP
│   │   │   ├── server.go              #   MCP server setup
│   │   │   ├── tools.go               #   9 tool registrations
│   │   │   └── identity.go            #   Identity propagation middleware
│   │   ├── webdav/                     # WebDAV proxy
│   │   │   └── handler.go
│   │   ├── grpc/                       # gRPC-Web handler (optional)
│   │   │   └── handler.go
│   │   ├── client/                     # Downstream gRPC clients
│   │   │   ├── fs_client.go
│   │   │   ├── search_client.go
│   │   │   ├── session_client.go
│   │   │   ├── resource_client.go
│   │   │   ├── crypto_client.go
│   │   │   └── admin_client.go
│   │   └── middleware/                 # HTTP middleware
│   │       ├── auth.go                #   Auth resolution
│   │       ├── cors.go
│   │       ├── timing.go
│   │       └── error_mapping.go       #   Domain errors → HTTP status
│   └── infra/
│       ├── config/config.go
│       ├── server/http.go
│       ├── telemetry/
│       └── wire/
├── Dockerfile
└── README.md
```

---

## 3. REST API Routes (17 Groups)

| Route Group | Prefix | Methods | Downstream Service |
|-------------|--------|---------|--------------------|
| `filesystem` | `/api/v1/filesystem` | GET/POST | openviking-fs |
| `content` | `/api/v1/content` | GET/POST | openviking-fs |
| `search` | `/api/v1/search` | POST | openviking-search |
| `sessions` | `/api/v1/sessions` | POST/GET/DELETE | openviking-session |
| `resources` | `/api/v1/resources` | POST/GET/DELETE | openviking-resource |
| `relations` | `/api/v1/relations` | GET/POST/DELETE | openviking-fs |
| `admin` | `/api/v1/admin` | POST/GET/DELETE | openviking-admin |
| `observer` | `/api/v1/observer` | GET | openviking-search |
| `privacy_configs` | `/api/v1/privacy-configs` | GET/POST | openviking-fs |
| `tasks` | `/api/v1/tasks` | GET | openviking-admin |
| `system` | `/api/v1/system` | GET | openviking-admin |
| `debug` | `/api/v1/debug` | GET/POST | openviking-search |
| `bot` | `/api/v1/bot` | POST/GET | openviking-gateway (internal) |
| `pack` | `/api/v1/pack` | POST | openviking-fs |
| `maintenance` | `/api/v1/maintenance` | POST | openviking-admin |
| `stats` | `/api/v1/stats` | GET | openviking-admin |
| `metrics` | `/metrics` | GET | openviking-gateway (internal) |

---

## 4. MCP Tools (9 tools)

| Tool | Input | Downstream | Description |
|------|-------|-----------|-------------|
| `search` | query, target_uri, limit, min_score | openviking-search | Semantic search |
| `read` | uris (string\|list) | openviking-fs | Batch-capable read |
| `list` | uri, recursive | openviking-fs | Directory listing |
| `store` | messages [{role, content}] | openviking-session | Auto session + commit |
| `add_resource` | path (URL), description | openviking-resource | Async ingestion |
| `grep` | uri, pattern(s) | openviking-fs | Regex content search |
| `glob` | pattern, uri, node_limit | openviking-fs | Filename glob |
| `forget` | uri | openviking-fs | Permanent deletion |
| `health` | — | openviking-admin | Health check |

---

## 5. Authentication

### 5.1 Three Auth Modes

| Mode | Mechanism | Identity Source | Default Role |
|------|-----------|----------------|-------------|
| `DEV` | No auth, localhost only | Implicit ROOT | ROOT |
| `API_KEY` | Root + per-user keys | `APIKeyManager.Resolve()` | Per-key role |
| `TRUSTED` | Gateway trust | HTTP headers `X-OpenViking-*` | Header-based |

### 5.2 Identity Resolution Flow

```
Request → Extract API Key (X-Api-Key or Authorization: Bearer)
  │
  ├── DEV mode: → ROOT, account/user from headers or "default"
  │
  ├── API_KEY mode:
  │   ├── Root key match? → ROOT, override from headers
  │   ├── Admin key match? → ADMIN, locked to account
  │   └── User key match? → USER, locked to account+user
  │
  └── TRUSTED mode:
      ├── Optional root key validation
      ├── Read X-OpenViking-Account, X-OpenViking-User
      └── Lookup role via APIKeyManager

→ RequestContext { UserIdentifier, Role, NamespacePolicy }
→ Propagate via gRPC metadata: x-account-id, x-user-id, x-agent-id, x-role
```

### 5.3 RBAC Enforcement

| Role | Admin APIs | Own Account | Other Account | Shared Resources |
|------|-----------|-------------|--------------|-----------------|
| ROOT | ✅ | ✅ | ✅ | ✅ |
| ADMIN | ❌ | ✅ | ❌ | ✅ (read) |
| USER | ❌ | ✅ (own user) | ❌ | ✅ (read) |

---

## 6. Error Mapping

| Domain Error | HTTP Status | gRPC Status |
|-------------|------------|-------------|
| `InvalidArgumentError` | 400 | `INVALID_ARGUMENT` |
| `UnauthenticatedError` | 401 | `UNAUTHENTICATED` |
| `PermissionDeniedError` | 403 | `PERMISSION_DENIED` |
| `NotFoundError` | 404 | `NOT_FOUND` |
| `AlreadyExistsError` | 409 | `ALREADY_EXISTS` |
| `FailedPreconditionError` | 412 | `FAILED_PRECONDITION` |
| `ResourceBusyError` | 423 | `RESOURCE_EXHAUSTED` |
| `NotInitializedError` | 503 | `UNAVAILABLE` |
| `InternalError` | 500 | `INTERNAL` |

Response format:
```json
{"error": {"code": "NOT_FOUND", "message": "...", "details": {}}}
```

---

## 7. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| chi/v5 over gin/echo | Lightweight, stdlib-compatible, middleware composability |
| Separate MCP port (8082) | Isolate long-lived SSE connections from REST traffic |
| WebDAV as proxy | Thin proxy to FS service, no business logic in gateway |
| Auth in gateway only | Services trust gateway; zero auth logic duplication |
| Per-service circuit breaker | Isolate failures; search timeout doesn't block FS ops |
