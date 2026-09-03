---
id: FEAT-001
title: Complete vnp-platform — Adapter + Infra Wiring
service: vnp-platform
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-002
linked_tdd: TDD-vnp-platform
---

## Mục Tiêu

Complete the vnp-platform consolidated service by implementing the adapter (gRPC handlers) and infra (PostgreSQL repositories, NATS publisher) layers. Domain entities and usecase ports are already scaffolded from SOL-001.

## Bối Cảnh Nghiệp Vụ

vnp-platform consolidates 7 legacy admin/auth/event services into a single binary for compact deployment mode. It exposes multiple gRPC service definitions on a single port and shares the database connection pool across all domains.

## Scope

### In Scope
- gRPC handlers for all 5 domains: Admin, Event, Auth, Analytics, Project
- PostgreSQL repositories implementing usecase/port interfaces
- NATS publisher for platform events (tenant lifecycle, key management)
- Wiring in cmd/server/main.go: DI, gRPC server, health check, graceful shutdown
- Database migration SQL files
- Integration with `pkg/tenant` for unified tenant context

### Out of Scope
- Domain entities (already exist in internal/domain/*)
- Usecase port interfaces (already exist in internal/usecase/port/interfaces.go)
- Admin usecase service (already exists in internal/usecase/admin/service.go)

## Thiết Kế Kỹ Thuật

### Existing Code (from SOL-001)
```
services/vnp-platform/
├── cmd/server/main.go                    ← Needs wiring update
├── internal/
│   ├── domain/
│   │   ├── admin/entity.go              ✅ Done
│   │   ├── analytics/entity.go          ✅ Done
│   │   ├── auth/entity.go               ✅ Done
│   │   ├── event/entity.go              ✅ Done
│   │   └── project/entity.go            ✅ Done
│   ├── usecase/
│   │   ├── port/interfaces.go           ✅ Done
│   │   └── admin/service.go             ✅ Done
│   └── infra/
│       └── config/config.go             ✅ Done
```

### New Code Required
```
├── internal/
│   ├── usecase/
│   │   ├── event/service.go             ← NEW
│   │   ├── auth/service.go              ← NEW
│   │   ├── analytics/service.go         ← NEW
│   │   └── project/service.go           ← NEW
│   ├── adapter/
│   │   └── grpc/
│   │       ├── admin_handler.go         ← NEW
│   │       ├── event_handler.go         ← NEW
│   │       ├── auth_handler.go          ← NEW
│   │       ├── analytics_handler.go     ← NEW
│   │       └── project_handler.go       ← NEW
│   └── infra/
│       ├── persistence/
│       │   ├── pg_tenant.go             ← NEW
│       │   ├── pg_apikey.go             ← NEW
│       │   ├── pg_user.go               ← NEW
│       │   ├── pg_event.go              ← NEW
│       │   └── pg_project.go            ← NEW
│       └── nats/publisher.go            ← NEW
├── migrations/
│   └── 001_initial.sql                  ← NEW
```

## Acceptance Criteria

- [ ] AC-1: `go build ./cmd/server/` compiles without errors
- [ ] AC-2: main.go registers all 5 gRPC services on single port 9050
- [ ] AC-3: All usecase port interfaces have concrete implementations
- [ ] AC-4: PostgreSQL repositories connect via shared pgxpool
- [ ] AC-5: NATS publisher sends TenantCreated/Deleted events
- [ ] AC-6: Health check endpoint responds on port 9103
- [ ] AC-7: Graceful shutdown: drain gRPC, close DB pool, close NATS

## Test Requirements
- **Unit tests:** All 5 usecase services
- **Integration tests:** gRPC round-trip for AdminService.CreateTenant
- **Minimum coverage:** 80%

## Definition of Done
- [ ] Code implements all Acceptance Criteria
- [ ] Unit tests pass, coverage ≥ 80%
- [ ] `docs/changelog.md` updated
- [ ] No lint errors
