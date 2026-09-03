---
id: TECH-001
title: Config + Wire DI Setup
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
risk_level: Low
rollback_plan: Remove generated files
---

## Mô Tả Thay Đổi

Setup Viper configuration loading and Google Wire dependency injection for vnp-gateway.

## Lý Do

Foundation infrastructure required before any usecase or adapter implementation.

## Các Bước Thực Hiện

1. Create `internal/infra/config/config.go` — Viper config struct with all env vars
2. Create `internal/infra/wire/wire.go` — Wire provider sets for all layers
3. Create `internal/infra/wire/injector.go` — Wire injector function
4. Create `cmd/main.go` — Entry point, config load, Wire init, graceful shutdown
5. Add `go:generate` directive for Wire code generation

## Risk & Mitigation
| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Wire code gen conflicts | Low | Low | Clear provider set boundaries |

## Verification Checklist
- [ ] `go generate ./gateway/internal/infra/wire/...` succeeds
- [ ] `go build ./gateway/cmd/...` succeeds
- [ ] Config loads from env vars correctly
- [ ] Graceful shutdown works (SIGTERM)
