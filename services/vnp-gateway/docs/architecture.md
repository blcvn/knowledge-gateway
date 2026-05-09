---
id: DOC-S03
service: vnp-gateway
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — Service Architecture

> **Group**: Platform | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/vnp-gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # RouteTarget, ProtocolType, AuthContext
│   │   ├── errors.go              # GatewayError types
│   │   └── event.go               # RequestReceived, RequestRouted
│   ├── usecase/
│   │   ├── route.go               # RouteUseCase — classify + route
│   │   ├── auth.go                # AuthenticateUseCase — JWT/APIKey
│   │   ├── mcp.go                 # MCPServerUseCase — tool dispatch
│   │   └── port/
│   │       ├── input.go           # Router, Authenticator, MCPHandler
│   │       └── output.go          # ServiceRegistry, TenantStore, KeyStore
│   ├── adapter/
│   │   ├── http/                  # chi/v5 REST handlers
│   │   │   ├── router.go
│   │   │   ├── memory_handler.go
│   │   │   ├── cognee_handler.go
│   │   │   ├── graphiti_handler.go
│   │   │   ├── memobase_handler.go
│   │   │   ├── openviking_handler.go
│   │   │   └── admin_handler.go
│   │   ├── grpc/                  # gRPC-Web proxy
│   │   ├── mcp/                   # MCP SSE/HTTP handler
│   │   ├── webdav/                # WebDAV proxy → ov-fs
│   │   ├── ws/                    # WebSocket handler
│   │   └── client/                # gRPC clients to all 35 services
│   └── infra/
│       ├── config/config.go
│       ├── server/http.go
│       ├── middleware/            # auth, cors, ratelimit, circuit_breaker
│       └── wire/wire.go
```

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| Redis | Rate limiting, API key cache |
| PostgreSQL | Tenant data, API key storage |
| All 35 services | gRPC fan-out routing |

## Design Decisions

- **chi/v5 router**: Lightweight, composable middleware stack
- **gRPC client pool**: Connection pooling to all downstream services
- **Circuit breaker per service**: Prevent cascade failures
- **Auto-routing**: Content classifier determines memory type for `memory.store`
