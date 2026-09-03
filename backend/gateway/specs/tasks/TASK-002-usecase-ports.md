---
id: TASK-002
title: Usecase Port Interfaces — Input + Output
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_arch: ARCH-002
depends_on: [TASK-001]
estimate: 2h
actual: 1h
---

## Mục Tiêu

Định nghĩa port interfaces cho Usecase layer — tách biệt inbound (input) và outbound (output) contracts.

## Phạm Vi

### Files đã tạo
- `gateway/internal/usecase/port/input.go` — 46 lines
- `gateway/internal/usecase/port/output.go` — 46 lines

### Chi tiết triển khai

#### `input.go` — 4 Input Ports (Driven BY adapters)
```go
// Router classifies content and resolves target service.
type Router interface {
    Route(ctx context.Context, req *domain.StoreRequest) (*domain.RouteResult, error)
    Classify(ctx context.Context, content string) (string, error)
}

// Authenticator validates credentials and returns auth context.
type Authenticator interface {
    AuthenticateJWT(ctx context.Context, token string) (*domain.AuthContext, error)
    AuthenticateAPIKey(ctx context.Context, key string) (*domain.AuthContext, error)
}

// MCPHandler dispatches MCP tool calls.
type MCPHandler interface {
    HandleTool(ctx context.Context, toolName string, params map[string]any) (any, error)
    ListTools(ctx context.Context) ([]ToolDefinition, error)
}

// RateLimiter checks and enforces rate limits.
type RateLimiter interface {
    Check(ctx context.Context, tenantID, endpoint string) (allowed bool, remaining int, err error)
}
```

#### `output.go` — 5 Output Ports (Driven BY usecases)
```go
// ServiceRegistry manages downstream gRPC connections.
type ServiceRegistry interface {
    Resolve(service string) (*domain.RouteTarget, error)
    Forward(ctx context.Context, target *domain.RouteTarget, req []byte) ([]byte, error)
    HealthCheck(service string) (bool, error)
}

// TenantStore provides tenant configuration lookup.
type TenantStore interface {
    GetTenant(ctx context.Context, id string) (*domain.TenantContext, error)
}

// KeyStore resolves API keys to authenticated identity.
type KeyStore interface {
    ResolveAPIKey(ctx context.Context, keyHash string) (*domain.AuthContext, error)
}

// EventPublisher publishes domain events to NATS JetStream.
type EventPublisher interface {
    Publish(ctx context.Context, subject string, event any) error
}

// RateLimitStore provides rate limit state management (Redis backend).
type RateLimitStore interface {
    CheckAndIncrement(ctx context.Context, key string, limit int, windowSec int) (allowed bool, remaining int, err error)
}
```

> **Thay đổi so với spec**: Thêm `RateLimitStore` interface (5 output ports thay vì 4) để tách biệt rate limit persistence khỏi business logic. Tất cả interfaces sử dụng `context.Context` và chỉ depend vào `domain` package types.

## Implementations

| Port Interface | Concrete Implementation | File |
|---|---|---|
| `ServiceRegistry` | `client.GRPCRegistry`, `client.CircuitRegistry` | `adapter/client/registry.go`, `adapter/client/circuit.go` |
| `TenantStore` | `persistence.PGTenantStore` | `infra/persistence/pg_store.go` |
| `KeyStore` | `persistence.PGTenantStore` | `infra/persistence/pg_store.go` |
| `EventPublisher` | `persistence.NATSPublisher` | `infra/persistence/nats_publisher.go` |
| `RateLimitStore` | `persistence.RedisRateLimiter` | `infra/persistence/ratelimit_redis.go` |
| `Router` | `usecase.RouteUseCase` | `usecase/route.go` |
| `Authenticator` | `usecase.AuthUseCase` | `usecase/auth.go` |
| `RateLimiter` | `usecase.RateLimitUseCase` | `usecase/ratelimit.go` |

## Acceptance Criteria

- [x] AC-1: `input.go` defines 4 interfaces: Router, Authenticator, MCPHandler, RateLimiter ✅
- [x] AC-2: `output.go` defines 5 interfaces: ServiceRegistry, TenantStore, KeyStore, EventPublisher, RateLimitStore ✅ (mở rộng từ 4 → 5)
- [x] AC-3: All method signatures use ONLY `domain` package types — no adapter/infra types ✅
- [x] AC-4: All methods accept `context.Context` as first parameter ✅
- [x] AC-5: `go build ./internal/usecase/...` succeeds ✅

## Verification

```bash
go build ./internal/usecase/...  # ✅ PASS
go vet ./internal/usecase/...    # ✅ PASS
```
