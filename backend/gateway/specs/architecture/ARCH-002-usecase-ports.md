---
id: ARCH-002
title: Usecase Ports — Input/Output Interfaces
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

Cần định nghĩa port interfaces (input + output) cho Usecase layer để đảm bảo Clean Architecture dependency rule.

## Kiến Trúc Mới Đề Xuất

```
internal/usecase/port/
├── input.go    # Interfaces driven BY adapters (inbound)
└── output.go   # Interfaces driven BY usecases (outbound)
```

## Phạm Vi Refactor

### Files cần tạo mới
- `gateway/internal/usecase/port/input.go`:
  - `Router` — Route(ctx, req) → RouteResult
  - `Authenticator` — AuthenticateJWT, AuthenticateAPIKey
  - `MCPHandler` — HandleTool(ctx, toolName, params)
  - `RateLimiter` — Check(ctx, tenantID, endpoint)
- `gateway/internal/usecase/port/output.go`:
  - `ServiceRegistry` — Resolve, Forward, HealthCheck
  - `TenantStore` — GetTenant
  - `KeyStore` — ResolveAPIKey
  - `EventPublisher` — Publish(ctx, subject, event)

## Invariants
- [ ] Port interfaces reference only Domain types (no adapter/infra types)
- [ ] No circular dependencies
- [ ] All methods accept `context.Context` as first parameter

## Acceptance Criteria
- [ ] AC-1: `input.go` defines 4 interfaces (Router, Authenticator, MCPHandler, RateLimiter)
- [ ] AC-2: `output.go` defines 4 interfaces (ServiceRegistry, TenantStore, KeyStore, EventPublisher)
- [ ] AC-3: All interfaces use only `domain` types in method signatures
- [ ] AC-4: `go build ./gateway/internal/usecase/...` succeeds
