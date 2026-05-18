---
id: TASK-004
title: "Embed Gateway (Config Override → localhost)"
app: apps/graphiti
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-003]
---

## Mục Tiêu

Embed gateway trong monolith app, override services map để route tới `localhost:PORT` thay vì remote services.

## Scope

### In Scope
- `internal/embed/gateway.go` — Gateway startup function
- Override `Services` map trong gateway config → localhost addresses
- Start REST server (:8080) + MCP server (:8082)
- Sử dụng existing gateway packages (import gateway/internal trực tiếp)

### Out of Scope
- Sửa code gateway (KHÔNG sửa `gateway/`)
- Health aggregation endpoint (TASK-005)

## Thiết Kế Kỹ Thuật

### Gateway Embedding Strategy

Gateway nằm cùng Go module root, nên **có thể import** `gateway/internal/*` trực tiếp (khác với services).

```go
// internal/embed/gateway.go
package embed

import (
    "context"

    gwconfig "github.com/vnp-community/vnp-memory/gateway/internal/infra/config"
    // ... other gateway imports
)

func StartGateway(ctx context.Context, cfg *config.Config) error {
    // 1. Override gateway config — route to localhost
    gwCfg := gwconfig.DefaultConfig()
    gwCfg.Server.RESTPort = cfg.GatewayRESTPort
    gwCfg.Server.MCPPort = cfg.GatewayMCPPort
    gwCfg.Auth.DevMode = cfg.AuthDevMode
    gwCfg.NATS.URL = cfg.NATSURL

    // Override services → localhost
    gwCfg.Services = cfg.GatewayServicesMap()

    // 2. Replicate gateway/cmd/main.go logic:
    //    - Setup infra (NATS publisher, key store, rate limit, circuit breaker)
    //    - Create usecases (auth, route, ratelimit)
    //    - Create handlers (memory, cognee, graphiti, memobase, etc.)
    //    - Create router
    //    - Start REST server
    //    - Start MCP server
    //    - Wait for ctx.Done()

    // Key: GRPCRegistry uses cfg.Services which now points to localhost
    return nil
}
```

### Gateway Config Override

```go
// Cái cần override trong gateway config:
gwCfg.Services = map[string]string{
    "graphiti-ingestion": "localhost:9021",
    "graphiti-search":    "localhost:9022",
    "graphiti-knowledge": "localhost:9023",
    "graphiti-store":     "localhost:9024",
    "graphiti-pipeline":  "localhost:9025",
}
```

### Important: Gateway Dependencies

Gateway cần các external dependencies:
- **PostgreSQL** (key store) → optional, fallback to noop
- **Redis** (rate limiting) → optional, fallback to noop (fail-open)
- **NATS** (event publishing) → optional, fallback to noop

Tất cả đều graceful degrade — giống pattern trong `gateway/cmd/main.go`.

## Acceptance Criteria

- [x] AC-1: Gateway REST server starts on configured port (default :8080)
- [x] AC-2: Gateway MCP server starts on configured port (default :8082)
- [x] AC-3: Gateway routes `POST /v1/episodes` → graphiti-ingestion via localhost gRPC
- [x] AC-4: Gateway routes `POST /v1/search` → graphiti-search via localhost gRPC
- [x] AC-5: Gateway health endpoint responds correctly
- [x] AC-6: **ZERO files modified** in `gateway/`
- [x] AC-7: Circuit breaker works for downstream localhost services

## Definition of Done

- [x] Gateway compiles and starts within monolith
- [x] REST API responds to curl test
- [x] Không có lint errors
