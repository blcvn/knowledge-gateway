# 01 — API Gateway Service

> **HTTP**: 8080 | **Health**: 8081 | **Metrics**: 8082

---

## 1. Purpose

Unified entry point cho tất cả external traffic. Thực hiện authentication, authorization, rate limiting, request validation, tenant resolution, và routing đến domain services qua gRPC.

---

## 2. Clean Architecture

```
services/gateway/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Route, RateLimitRule, TenantContext
│   │   ├── value_object.go     # AuthMethod, PlanTier, Permission
│   │   └── errors.go           # ErrUnauthorized, ErrRateLimited, ErrForbidden
│   ├── usecase/
│   │   ├── authenticate.go     # Token/API Key validation → Auth Service
│   │   ├── authorize.go        # RBAC permission check
│   │   ├── rate_limit.go       # Per-key, per-org rate limiting
│   │   ├── port/
│   │   │   ├── input.go        # AuthenticateUseCase, AuthorizeUseCase
│   │   │   └── output.go       # AuthValidator, RateLimiter, TenantResolver
│   │   └── dto/
│   │       └── auth.go         # AuthResult, RateLimitResult
│   ├── adapter/
│   │   ├── http/
│   │   │   ├── router.go       # chi.Router setup, versioned route groups
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go          # JWT/API Key extraction + validation
│   │   │   │   ├── tenant.go        # Org resolution, context injection
│   │   │   │   ├── ratelimit.go     # Token bucket per API key
│   │   │   │   ├── cors.go          # CORS configuration
│   │   │   │   ├── request_id.go    # X-Request-ID injection
│   │   │   │   ├── logging.go       # Structured request logging
│   │   │   │   ├── recovery.go      # Panic recovery
│   │   │   │   └── metrics.go       # Prometheus histogram
│   │   │   └── handler/
│   │   │       ├── document.go      # /api/v1/documents/* → Document Service
│   │   │       ├── memory.go        # /api/v1/memories/*  → Memory Service
│   │   │       ├── search.go        # /api/v1/search/*    → Search Service
│   │   │       ├── profile.go       # /api/v1/profiles/*  → Profile Service
│   │   │       ├── connector.go     # /api/v1/connections/*→ Connector Service
│   │   │       ├── project.go       # /api/v1/projects/*  → Project Service
│   │   │       ├── analytics.go     # /api/v1/analytics/* → Analytics Service
│   │   │       ├── auth.go          # /api/v1/auth/*      → Auth Service
│   │   │       └── mcp.go           # /mcp/*              → MCP Service (SSE proxy)
│   │   ├── grpc_client/
│   │   │   ├── document.go     # Document Service gRPC client
│   │   │   ├── memory.go       # Memory Service gRPC client
│   │   │   ├── search.go       # Search Service gRPC client
│   │   │   ├── profile.go      # Profile Service gRPC client
│   │   │   ├── connector.go    # Connector Service gRPC client
│   │   │   ├── project.go      # Project Service gRPC client
│   │   │   ├── analytics.go    # Analytics Service gRPC client
│   │   │   └── auth.go         # Auth Service gRPC client
│   │   └── cache/
│   │       └── redis.go        # Rate limit counters, auth token cache
│   └── infra/
│       ├── config/config.go    # Gateway config (routes, rate limits, CORS)
│       └── wire/wire.go        # DI wiring
├── api/
│   └── openapi/v1.yaml         # OpenAPI 3.1 spec (generated)
└── Dockerfile
```

---

## 3. Route Mapping

| Method | Path | Target Service | gRPC Method |
|--------|------|---------------|-------------|
| POST | `/api/v1/documents` | Document | `CreateDocument` |
| GET | `/api/v1/documents/:id` | Document | `GetDocument` |
| POST | `/api/v1/documents/list` | Document | `ListDocuments` |
| DELETE | `/api/v1/documents/:id` | Document | `DeleteDocument` |
| DELETE | `/api/v1/documents/bulk` | Document | `BulkDeleteDocuments` |
| POST | `/api/v1/search` | Search | `HybridSearch` |
| POST | `/api/v1/search/memories` | Search | `MemorySearch` |
| POST | `/api/v1/memories/forget` | Memory | `ForgetMemory` |
| GET | `/api/v1/memories/:id/graph` | Memory | `GetMemoryGraph` |
| GET | `/api/v1/profiles/:containerTag` | Profile | `GetProfile` |
| POST | `/api/v1/connections/:provider` | Connector | `CreateConnection` |
| GET | `/api/v1/connections` | Connector | `ListConnections` |
| DELETE | `/api/v1/connections/:id` | Connector | `DeleteConnection` |
| GET | `/api/v1/projects` | Project | `ListProjects` |
| POST | `/api/v1/projects` | Project | `CreateProject` |
| DELETE | `/api/v1/projects/:id` | Project | `DeleteProject` |
| GET | `/api/v1/analytics/usage` | Analytics | `GetUsageAnalytics` |
| POST | `/api/v1/auth/login` | Auth | `Login` |
| POST | `/api/v1/auth/api-keys` | Auth | `CreateAPIKey` |
| GET | `/api/v1/auth/session` | Auth | `GetSession` |
| ALL | `/mcp` | MCP | SSE/JSON-RPC proxy |

---

## 4. Middleware Pipeline

```
Request → RequestID → CORS → RateLimit → Auth → Tenant → Logging → Metrics → Handler
                                                                                 │
Response ← Recovery ← ErrorMapping ← Logging ← Metrics ────────────────────── ◄─┘
```

---

## 5. Rate Limiting

```go
type RateLimitConfig struct {
    PerAPIKey   Rate  // 100 req/min (Pro), 1000 req/min (Scale), unlimited (Enterprise)
    PerOrg      Rate  // Aggregate org-level limits
    BurstFactor int   // 2x burst allowance
}

// Redis-backed sliding window
type SlidingWindowLimiter struct {
    redis  *redis.Client
    window time.Duration
    limit  int
}
```

| Plan | Requests/min | Burst | Documents/month |
|------|-------------|-------|-----------------|
| **Pro** | 100 | 200 | 10,000 |
| **Scale** | 1,000 | 2,000 | 100,000 |
| **Enterprise** | Custom | Custom | Unlimited |
