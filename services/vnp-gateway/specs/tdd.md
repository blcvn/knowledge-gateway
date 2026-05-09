---
id: TDD-vnp-gateway
title: Technical Design — vnp-gateway
service: vnp-gateway
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: Platform
---

# Technical Design — vnp-gateway

> **Group**: Platform | **Ports**: 8080 (REST), 8081 (gRPC), 8082 (MCP) | **Origin**: Unified

## 1. Service Overview

Unified API Gateway — single entry point for ALL memory APIs. REST (external) + gRPC-Web + MCP (SSE/JSON-RPC) + WebSocket + WebDAV. Handles auth (JWT/APIKey), rate limiting, CORS, protocol translation, circuit breaking, and request routing to 35 downstream services.

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── route.go                 # Route, ServiceTarget, ProtocolType
│   ├── auth.go                 # AuthResult, TokenClaims, APIKeyResult
│   ├── ratelimit.go            # RateLimitConfig, SlidingWindow
│   └── health.go               # ServiceHealth, AggregatedStatus
├── event.go                     # RequestReceived (audit)
└── errors.go                    # Unauthorized, RateLimited, ServiceUnavailable
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── routing.go                  # Route resolution: path → service target
├── auth.go                     # JWT validation + API key validation
├── ratelimit.go                # Sliding window rate limiter
├── circuit_breaker.go          # Per-service circuit breaking
├── memory_router.go            # Auto-routing for memory.store()
├── port/
│   ├── input.go               # GatewayUseCase
│   └── output.go             # AuthPort, RateLimitPort, ServiceRegistryPort
└── dto/
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── http/
│   ├── router.go               # chi/v5 REST router
│   ├── middleware/              # Auth, CORS, RequestID, Logging, Recovery
│   └── handler/                # REST endpoint handlers
├── grpc/
│   └── proxy.go                # gRPC-Web proxy
├── mcp/
│   └── server.go               # MCP SSE/JSON-RPC server
├── webdav/
│   └── proxy.go                # WebDAV proxy → ov-fs
└── client/
    ├── service_pool.go         # gRPC connection pool to all services
    └── vnp_admin_client.go     # Admin client (key validation)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── redis/
│   ├── ratelimit.go            # Redis sliding window
│   └── circuit_breaker.go      # Redis-backed circuit state
├── config/config.go
├── server/
│   ├── http.go                 # HTTP server (REST + MCP)
│   └── grpc.go                 # gRPC-Web server
└── wire/wire.go
```

## 3. Protocol Surface

| Protocol | Port | Use Case |
|----------|------|----------|
| REST (chi/v5) | 8080 | External API, OpenAPI 3.1 |
| gRPC-Web | 8081 | Browser clients, web apps |
| MCP (SSE) | 8082 | AI agent tool integration |
| WebDAV | 8080 `/webdav/` | OpenViking filesystem access |
| WebSocket | 8080 `/ws/` | Real-time streaming |

## 4. Auto-Routing (memory.store)

```go
func routeStore(req StoreRequest) Service {
    switch req.Type {
    case "semantic":       → cognee-ingestion
    case "episodic":       → graphiti-ingestion
    case "conversational": → memobase-ingestion
    case "procedural":     → ov-resource
    case "auto":           → classify(req.Data) → re-route
    }
}
```

## 5. Cross-Cutting Concerns

| Concern | Implementation |
|---------|---------------|
| Auth | JWT (RS256) + API Key (SHA-256 → vnp-admin validation) |
| Rate Limit | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | sony/gobreaker, per-downstream-service |
| CORS | Configurable origins |
| Request ID | UUID v7, X-Request-ID header |
| Timeout | 30s default, 120s ingestion, 10s search |

## 6. Multi-Tenancy

- JWT: `tenant_id` claim → `x-tenant-id` gRPC metadata
- API Key: validate via vnp-admin → extract tenant_id → propagate

---

> **Next Steps**: FEAT-001 (REST Routing), FEAT-002 (Auth Middleware), FEAT-003 (Rate Limiting), FEAT-004 (Circuit Breaking), FEAT-005 (MCP Server), FEAT-006 (WebDAV Proxy).
