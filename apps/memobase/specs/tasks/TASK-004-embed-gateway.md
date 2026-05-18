---
id: TASK-004
title: "Embed Gateway with localhost Routing"
app: apps/memobase
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-003]
---

## Mục Tiêu

Tạo `cmd/memobase/gateway.go` — embed vnp-gateway vào monolith process, override service registry để route gRPC calls qua `localhost:PORT` thay vì external service addresses.

## Scope

### In Scope
- `apps/memobase/cmd/memobase/gateway.go` — Gateway start function
- `apps/memobase/internal/gateway/override.go` — Service map override (optional)
- Override `ServiceRegistry` config → localhost:PORT mapping
- MCP server embedding (same process)

### Out of Scope
- Gateway code changes (ZERO modifications)
- Auth/rate limiting changes
- Service start functions (TASK-003)

## Thiết Kế Kỹ Thuật

### Gateway Config Override

Gateway đọc service addresses từ config (`cfg.Services`). Trong monolith mode:

```go
func startGateway(ctx context.Context, cfg *config.Config) error {
    // Override gateway services config → localhost:PORT
    os.Setenv("SERVICES_MEMOBASE_INGESTION", fmt.Sprintf("localhost:%d", cfg.Services.IngestionPort))
    os.Setenv("SERVICES_MEMOBASE_ENGINE", fmt.Sprintf("localhost:%d", cfg.Services.EnginePort))
    os.Setenv("SERVICES_MEMOBASE_CONTEXT", fmt.Sprintf("localhost:%d", cfg.Services.ContextPort))
    os.Setenv("SERVICES_MEMOBASE_PIPELINE", fmt.Sprintf("localhost:%d", cfg.Services.PipelinePort))

    // Replicate gateway/cmd/main.go startup logic
    // - config.Load() (reads ENV overrides)
    // - Setup infrastructure (NATS, PG, Redis)
    // - Setup usecases (auth, route, rate limit)
    // - Setup handlers (memobase, admin, etc.)
    // - Start HTTP + MCP servers
    // - Block until ctx.Done()
}
```

### Key Routing Table (from gateway source)

```go
// gateway/internal/adapter/handler/router.go registers:
memobaseH := handler.NewMemobaseHandler(registry, logger)
// Routes: /api/v1/blobs/*, /api/v1/users/profile/*, /api/v1/users/buffer/*,
//         /api/v1/users/context/*, /api/v1/users/event/*
```

### Service Registry Mapping

| Gateway Config Key | Monolith Value |
|-------------------|----------------|
| `memobase-ingestion` | `localhost:9041` |
| `memobase-engine` | `localhost:9042` |
| `memobase-context` | `localhost:9043` |
| `memobase-pipeline` | `localhost:9044` |

### Gateway Ports

| Component | Port | ENV |
|-----------|------|-----|
| REST API | 8080 | `REST_PORT` |
| MCP Server | 8082 | `MCP_PORT` |
| Health/Metrics | 8083 | `HEALTH_PORT` |

### Constraint: ZERO gateway code changes
- Gateway reads config via `config.Load()` which uses ENV vars
- We override ENV vars BEFORE calling gateway init
- All gateway functionality (auth, CORS, rate limit, MCP) works unchanged

## Acceptance Criteria

- [x] AC-1: Gateway starts on configured REST port (default 8080)
- [x] AC-2: Gateway routes REST → gRPC to `localhost:PORT` for each memobase service
- [x] AC-3: MCP server starts on configured port (default 8082)
- [x] AC-4: Auth middleware works (JWT/API Key extraction)
- [x] AC-5: Circuit breaker works for localhost connections
- [x] AC-6: Graceful shutdown when context is cancelled
- [x] AC-7: **ZERO lines changed** in `gateway/` directory
- [x] AC-8: `GET /api/v1/healthcheck` returns 200

## Test Requirements
- Integration test: start all services + gateway, verify REST → gRPC routing
- Verify MCP server responds
- Verify health endpoint
