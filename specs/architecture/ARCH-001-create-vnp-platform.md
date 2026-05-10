---
id: ARCH-001
title: Create vnp-platform consolidated service structure
service: vnp-platform
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

7 separate admin/auth/event services (vnp-admin, vnp-event, ov-admin, zep-admin, sm-auth, sm-analytics, sm-project) thực hiện cùng các capability:
- User/Tenant/APIKey CRUD
- Health aggregation
- RBAC management
- Event timeline
- Usage analytics

Mỗi service chạy binary riêng, gRPC port riêng, dẫn đến:
- 7 containers cho cross-cutting admin concerns
- Duplicate auth logic (JWT validation, API key management)
- Inconsistent RBAC across engines

## Kiến Trúc Mới Đề Xuất

Gộp thành **1 service `vnp-platform`** với sub-domain packages:

```
services/vnp-platform/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── admin/          # Tenant, User, APIKey entities
│   │   ├── event/          # UserEvent, Timeline entities
│   │   ├── auth/           # JWT, RBAC, Organization entities
│   │   ├── analytics/      # Usage, TokenEconomics entities
│   │   └── project/        # Space, Tag, Membership entities
│   ├── usecase/
│   │   ├── admin/          # CreateTenant, CreateUser, CreateAPIKey, AggregatedHealth
│   │   ├── event/          # CreateEvent, SearchEvents, GetTimeline
│   │   ├── auth/           # ValidateJWT, ValidateAPIKey, CheckRBAC
│   │   ├── analytics/      # TrackUsage, GetReport
│   │   └── project/        # CreateSpace, ManageTags
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── admin_handler.go      # VnpAdminService (from vnp-admin)
│   │   │   ├── event_handler.go      # VnpEventService (from vnp-event)
│   │   │   ├── ov_admin_handler.go   # OvAdminService (from ov-admin)
│   │   │   ├── zep_admin_handler.go  # ZepAdminService (from zep-admin) → projected
│   │   │   ├── auth_handler.go       # SmAuthService (from sm-auth)
│   │   │   ├── analytics_handler.go  # SmAnalyticsService (from sm-analytics)
│   │   │   └── project_handler.go    # SmProjectService (from sm-project)
│   │   ├── repository/
│   │   │   └── postgres/
│   │   └── event/
│   │       └── nats/
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go         # Register ALL 7 gRPC services
│       └── wire/wire.go
├── docs/
│   ├── README.md
│   ├── api.md
│   ├── architecture.md
│   ├── data-model.md
│   ├── configuration.md
│   ├── runbook.md
│   └── changelog.md
├── specs/
│   ├── solutions/
│   ├── architecture/
│   ├── tasks/
│   └── tdd.md
└── Dockerfile
```

## Phạm Vi Refactor

### Files cần tạo mới
- `services/vnp-platform/` — entire new service directory
- All domain/usecase/adapter/infra packages as described above

### Files cần sửa
- `vnp-gateway`: route targets cho admin/event/auth endpoints → vnp-platform:9050

### Files cần xóa (sau migration hoàn thành)
- `services/vnp-admin/` → absorbed
- `services/vnp-event/` → absorbed
- `services/ov-admin/` → absorbed
- `services/zep-admin/` → absorbed
- `services/sm-auth/` → absorbed
- `services/sm-analytics/` → absorbed
- `services/sm-project/` → absorbed

## Invariants (Không được thay đổi)

> AI phải đảm bảo những điều sau KHÔNG thay đổi sau refactor:
- [ ] Tất cả gRPC service definitions (proto) giữ nguyên
- [ ] API response format giữ nguyên cho mọi endpoint
- [ ] NATS event subjects giữ nguyên (admin.tenant.created, admin.tenant.deleted)
- [ ] PostgreSQL schema giữ nguyên

## Acceptance Criteria

- [ ] AC-1: `vnp-platform` binary builds successfully
- [ ] AC-2: Registers 7 gRPC services: VnpAdminService, VnpEventService, OvAdminService, ZepAdminService, SmAuthService, SmAnalyticsService, SmProjectService
- [ ] AC-3: Health check endpoint responds at :9103
- [ ] AC-4: NATS subscriber handles `admin.tenant.created`, `admin.tenant.deleted`
- [ ] AC-5: All existing proto service methods callable via vnp-platform:9050

## Verification

- [ ] Toàn bộ existing tests pass sau refactor
- [ ] Không có new linter errors
- [ ] `docker-compose up vnp-platform` starts successfully
