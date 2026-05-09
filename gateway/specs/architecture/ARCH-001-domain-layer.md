---
id: ARCH-001
title: Domain Layer — Entities, Errors, Events
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
linked_adr: TBD
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

Gateway chưa tồn tại. Cần thiết lập Domain layer (Layer 1) — foundation cho toàn bộ service.

## Kiến Trúc Mới Đề Xuất

Tạo `internal/domain/` với 3 files:
- `entity.go` — Core domain types
- `errors.go` — Domain error types
- `event.go` — Domain events for NATS publishing

## Phạm Vi Refactor

### Files cần tạo mới
- `gateway/internal/domain/entity.go` — AuthContext, RouteTarget, ProtocolType, TenantContext, StoreRequest
- `gateway/internal/domain/errors.go` — GatewayError, ErrUnauth, ErrForbidden, ErrRateLimit, ErrCircuitOpen, ErrTimeout
- `gateway/internal/domain/event.go` — RequestReceived, RequestRouted, AuthFailed, RateLimitExceeded

### Files cần sửa
- None (new service)

### Files cần xóa
- None

## Invariants (Không được thay đổi)
> Domain layer MUST NOT import any external package except stdlib.
- [ ] Zero external dependencies in `internal/domain/`
- [ ] All types are pure Go structs/interfaces
- [ ] No database, HTTP, or gRPC references

## Acceptance Criteria
- [ ] AC-1: `entity.go` defines AuthContext, RouteTarget, ProtocolType, TenantContext
- [ ] AC-2: `errors.go` defines GatewayError implementing `error` interface + sentinel errors
- [ ] AC-3: `event.go` defines event structs with JSON tags for NATS publishing
- [ ] AC-4: `go vet ./internal/domain/...` passes with zero warnings
- [ ] AC-5: No imports outside Go stdlib in `internal/domain/`

## Verification
- [ ] `go build ./gateway/internal/domain/...` succeeds
- [ ] `go vet ./gateway/internal/domain/...` clean
- [ ] No external imports verified via `grep -r "import" gateway/internal/domain/`
