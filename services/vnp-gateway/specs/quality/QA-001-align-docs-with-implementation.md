---
id: QA-001
title: Align vnp-gateway docs/specs with implementation
service: vnp-gateway
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-002
linked_tdd: TDD-vnp-gateway
---

## Mục Tiêu

Align vnp-gateway documentation and specs with the existing Go implementation (20+ source files). The gateway is already fully implemented — this task ensures docs accuracy and creates implementation verification specs.

## Bối Cảnh

The gateway already has a production-grade Go implementation in `gateway/` (separate from `services/vnp-gateway/`). The `services/vnp-gateway/` directory contains docs and specs that need to reference the actual implementation.

## Scope

### In Scope
- Verify `docs/api.md` matches actual router.go endpoints
- Verify `docs/architecture.md` matches actual layer structure
- Verify `docs/configuration.md` matches actual config.go env vars
- Verify `docs/data-model.md` matches actual PostgreSQL schema
- Update `specs/tdd.md` with accurate file paths and function signatures
- Create QA spec to validate gateway test coverage

### Out of Scope
- New feature development
- Code refactoring

## Các Bước Thực Hiện

### Step 1: Audit implementation vs docs
```
gateway/
├── cmd/main.go                    → verify against docs/architecture.md
├── internal/
│   ├── adapter/
│   │   ├── handler/router.go      → verify against docs/api.md
│   │   ├── mcp/server.go          → verify MCP endpoints in api.md
│   │   └── webdav/proxy.go        → verify WebDAV config in api.md
│   ├── usecase/auth.go            → verify auth flow in architecture.md
│   └── infra/
│       ├── config/config.go       → verify against docs/configuration.md
│       ├── middleware/*.go         → verify middleware chain in architecture.md
│       └── persistence/*.go       → verify against docs/data-model.md
```

### Step 2: Update docs
- Fix any discrepancies found in Step 1
- Add missing env vars to configuration.md
- Add missing endpoints to api.md

### Step 3: Update tdd.md
- Update file paths to match `gateway/internal/` structure
- Add function signatures for key handlers
- Mark all existing features as implemented

## Acceptance Criteria

- [ ] AC-1: Every endpoint in `gateway/internal/adapter/handler/router.go` is documented in `services/vnp-gateway/docs/api.md`
- [ ] AC-2: Every env var in `gateway/internal/infra/config/config.go` is documented in `services/vnp-gateway/docs/configuration.md`
- [ ] AC-3: `services/vnp-gateway/specs/tdd.md` file paths match actual gateway implementation
- [ ] AC-4: No undocumented public gRPC/REST/MCP endpoints
- [ ] AC-5: `docs/changelog.md` updated with alignment entry

## Definition of Done
- [ ] All discrepancies between docs and implementation resolved
- [ ] tdd.md updated with accurate implementation status
- [ ] Changelog updated
