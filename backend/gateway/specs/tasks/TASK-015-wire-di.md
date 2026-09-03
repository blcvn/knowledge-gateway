---
id: TASK-015
title: Wire DI — google/wire Code Generation
service: vnp-gateway
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
depends_on: [TASK-003]
estimate: 3h
---

## Mục Tiêu

Replace manual wiring trong `cmd/main.go` với Google Wire code generation. Giảm boilerplate, enable compile-time DI verification.

## Phạm Vi

### Files cần tạo
- `gateway/internal/infra/wire/providers.go` — Provider sets
- `gateway/internal/infra/wire/injector.go` — Wire injector (//go:generate)
- `gateway/internal/infra/wire/wire_gen.go` — Generated code
- Update `cmd/main.go` — use `wire.InitializeApp(cfg)`

### Acceptance Criteria

- [ ] AC-1: `go generate ./internal/infra/wire/...` produces `wire_gen.go`
- [ ] AC-2: `cmd/main.go` reduced to < 30 lines (config + wire + signal)
- [ ] AC-3: All dependencies resolved at compile time
- [ ] AC-4: Noop fallback providers for unavailable infra (Redis, PG, NATS)
- [ ] AC-5: Wire cleanup function handles all resource shutdown

## Verification

```bash
go generate ./internal/infra/wire/...
go build ./cmd/...
```
