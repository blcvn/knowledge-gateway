---
id: TASK-004
title: "Embed Gateway — Override Services Map to localhost"
app: apps/cognee
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-003]
estimated: 2h
---

## Mục Tiêu

Embed gateway vào monolith process, override `services` map từ
remote addresses (`cognee-ingestion:9011`) thành localhost (`localhost:9011`).
Gateway sử dụng **cùng GRPCRegistry + middleware + router** patterns.

## Scope

### In Scope
- `cmd/cognee/gateway.go` — startGateway function
- Replicate `gateway/cmd/main.go` logic:
  - Load gateway config (override services map)
  - Setup infra (NATS publisher, PG store, Redis rate limiter)
  - Build usecase layer (AuthUseCase, RouteUseCase, RateLimitUseCase)
  - Create handlers + router
  - Start HTTPServer
- Override `Services` map → localhost:PORT cho embedded services

### Out of Scope
- Sửa bất kỳ file nào trong `gateway/` directory

## Thiết Kế Kỹ Thuật

### Gateway Embedding Strategy

Gateway `cmd/main.go` (264 lines) chứa tất cả logic inline (không dùng Wire).
Ta replicate pattern tương tự, thay đổi duy nhất: **services addresses → localhost**.

```go
// cmd/cognee/gateway.go
package main

import (
    // Gateway imports (same module — accessible vì gateway package là public)
    // NHƯNG: gateway dùng module path khác (github.com/vnp-community/vnp-memory/gateway)
    // → Cần dùng replace directive hoặc replicate logic

    // Option A: Nếu gateway là separate module → replicate logic
    // Option B: Nếu chung go workspace → import trực tiếp
)
```

### Approach A: Replicate Gateway Init (Recommended)

Replicate logic từ `gateway/cmd/main.go` vì:
1. Gateway dùng module path riêng (`github.com/vnp-community/vnp-memory/gateway`)
2. Gateway import `internal/` packages → app KHÔNG thể import trực tiếp
3. Nhưng logic chỉ ~200 lines, chủ yếu là wiring

```go
func startGateway(ctx context.Context, cfg *config.Config) error {
    logger := slog.Default()

    // Override services map: remote → localhost
    servicesMap := cfg.GatewayServicesMap()
    // → {"cognee-ingestion": "localhost:9011", "cognee-cognify": "localhost:9012", ...}

    // 1. NATS Publisher (optional, same as gateway)
    var publisher EventPublisher
    if cfg.NATSURL != "" {
        // Connect to NATS (same pattern as gateway/cmd/main.go lines 66-78)
        // ...
    }

    // 2. PostgreSQL (optional, for API key store)
    // ... same pattern as gateway

    // 3. Redis (optional, for rate limiting)
    // ... same pattern as gateway

    // 4. gRPC Registry — KEY DIFFERENCE: localhost addresses
    grpcReg, cleanup, err := NewLocalGRPCRegistry(servicesMap, timeout, logger)
    defer cleanup()

    // 5. Auth + Route + RateLimit usecases
    // ... same as gateway wiring

    // 6. Handlers + Router
    // ... same as gateway

    // 7. HTTP Server
    srv := NewHTTPServer(router, cfg.GatewayRESTPort, logger)

    // Block until ctx done
    errCh := make(chan error, 1)
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            errCh <- err
        }
        close(errCh)
    }()

    select {
    case err := <-errCh:
        return err
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        return srv.Shutdown(shutdownCtx)
    }
}
```

### Approach B: Import via Go Workspace (Alternative)

Nếu monorepo dùng `go.work`, có thể import gateway packages trực tiếp:

```go
// go.work
use (
    ./gateway
    ./apps/cognee
    ./pkg/telemetry
    ./pkg/tenant
)
```

Nhưng vẫn KHÔNG thể import `gateway/internal/` packages (Go restriction).

### gRPC Registry (replicate from gateway)

```go
// Replicate gateway/internal/adapter/client/registry.go patterns
// with hardcoded localhost addresses
type LocalGRPCRegistry struct {
    conns   map[string]*grpc.ClientConn
    targets map[string]*RouteTarget
    logger  *slog.Logger
}

func NewLocalGRPCRegistry(services map[string]string, timeout time.Duration, logger *slog.Logger) (*LocalGRPCRegistry, func(), error) {
    // Same logic as gateway GRPCRegistry but connect to localhost
}
```

### Middleware (replicate from gateway)

Replicate middleware functions nếu cannot import:
- Recovery, RequestID, Logger, CORS, Timeout — standard patterns
- Auth — JWT + API Key validation

**Total replicated gateway code: ~400 lines** (của 264 + ~188 lines patterns)

## Acceptance Criteria

- [x] AC-1: Gateway starts on configured REST port
- [x] AC-2: Gateway routes to embedded services via localhost gRPC
- [x] AC-3: POST `/v1/cognee/datasets` reaches ingestion service
- [x] AC-4: POST `/v1/cognee/search` reaches search service
- [x] AC-5: Auth middleware works (dev mode or JWT)
- [x] AC-6: Health endpoint responds on gateway port
- [x] AC-7: **ZERO changes** to gateway/ directory
- [x] AC-8: Graceful shutdown drains HTTP connections

## Definition of Done

- [x] Gateway embedded and routing to localhost services
- [x] Middleware chain functional
- [x] All cognee routes verified
- [x] No modifications to gateway/ codebase
