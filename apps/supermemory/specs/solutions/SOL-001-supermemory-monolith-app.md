---
id: SOL-001
title: "Supermemory Monolith App — Embedded Multi-Service Process"
app: apps/supermemory
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_cr: null
approved_by: pending
---

## Yêu Cầu Gốc

Hợp nhất **tất cả Supermemory microservices** (Document, Memory, Search, Profile, Connector, MCP, Auth, Analytics, Project, Engine) + **Gateway** thành **1 binary duy nhất** chạy ở mức **production grade** và **enterprise level**.

### Nguyên tắc thiết kế:
- **ZERO changes** đến code services (`services/sm-*`) và gateway (`gateway/`)
- **Reuse toàn bộ** — Import từ services/gateway packages, chỉ viết orchestration code
- **Internal gRPC/NATS OK** — Các module giao tiếp qua gRPC localhost + NATS như thiết kế gốc
- **Single Process** — Mọi thứ chạy trong 1 process, deploy 1 binary

## Phân Tích Kiến Trúc

### Cách hoạt động hiện tại (Microservices)

```
Client → [Gateway:8080] →─gRPC──→ [sm-document:9001]
                         →─gRPC──→ [sm-memory:9002]
                         →─gRPC──→ [sm-search:9003]
                         →─gRPC──→ [sm-profile:9004]
                         →─gRPC──→ [sm-connector:9005]
                         →─gRPC──→ [sm-mcp:9006]
                         →─gRPC──→ [sm-auth:9007]
                         →─gRPC──→ [sm-analytics:9008]
                         →─gRPC──→ [sm-project:9009]
                         →─gRPC──→ [sm-engine:9010]
```

Mỗi service là 1 process riêng, gateway route qua gRPC, async events qua NATS.

### Cách hoạt động mới (Monolith — 1 process)

```
                    ┌──── SINGLE PROCESS ─────────────────────────────────────────┐
                    │                                                              │
Client → [Gateway :8080] →─gRPC localhost──→ [document   goroutine:9001]           │
                    │    →─gRPC localhost──→ [memory     goroutine:9002]           │
                    │    →─gRPC localhost──→ [search     goroutine:9003]           │
                    │    →─gRPC localhost──→ [profile    goroutine:9004]           │
                    │    →─gRPC localhost──→ [connector  goroutine:9005]           │
                    │    →─gRPC localhost──→ [mcp        goroutine:9006]           │
                    │    →─gRPC localhost──→ [auth       goroutine:9007]           │
                    │    →─gRPC localhost──→ [analytics  goroutine:9008]           │
                    │    →─gRPC localhost──→ [project    goroutine:9009]           │
                    │    →─gRPC localhost──→ [engine     goroutine:9010]           │
                    │                                                              │
                    │  [Health Aggregation :9090]                                  │
                    └──────────────────────────────────────────────────────────────┘
```

**Khác biệt duy nhất:** Tất cả services + gateway chạy trong 1 process thay vì 11 processes.
- gRPC vẫn hoạt động → gọi qua `localhost` (zero-copy loopback, ~0.1ms overhead)
- NATS vẫn hoạt động → event-driven giữ nguyên
- **KHÔNG thay đổi bất kỳ code nào** trong services hay gateway

### Tại sao approach này tốt nhất?

| Tiêu chí | Ưu điểm |
|----------|---------|
| **Zero code change** | Services + Gateway giữ nguyên 100%, không sửa 1 dòng nào |
| **Proven patterns** | gRPC localhost + NATS đã được test kỹ trong production |
| **Consistent with others** | Cùng supervisor pattern đã triển khai thành công với Memobase/Cognee/Graphiti/OpenViking |
| **Rollback trivial** | Chỉ cần deploy microservices + gateway riêng biệt như cũ |
| **Upgrade safe** | Khi services update, app tự động nhận code mới (same Go module) |

### Code CẦN viết mới (chỉ orchestration)

```
🆕 apps/supermemory/cmd/supermemory/main.go              — Orchestrator: start all services + gateway
🆕 apps/supermemory/cmd/supermemory/services.go           — Service start functions
🆕 apps/supermemory/cmd/supermemory/gateway.go            — Gateway start function
🆕 apps/supermemory/cmd/supermemory/health.go             — Health aggregation server
🆕 apps/supermemory/internal/supervisor/supervisor.go  — Process supervisor (lifecycle manager)
🆕 apps/supermemory/internal/config/config.go          — Unified config for all embedded services
🆕 apps/supermemory/Dockerfile                         — Multi-stage build
🆕 apps/supermemory/Makefile                           — Build/run targets
🆕 apps/supermemory/docker-compose.yml                 — Local dev (NATS, Postgres, Redis)
🆕 apps/supermemory/config.yaml                        — Unified config file
🆕 apps/supermemory/.env.example                       — Environment variable template
```

## Giải Pháp Chi Tiết

### Approach: Embedded Service Supervisor (Proven Pattern)

App `supermemory` là **process supervisor** chạy trong 1 Go process:
1. Load unified config → phân phối thành ENV vars cho từng embedded service
2. Start mỗi service init logic trong goroutine riêng (phased startup)
3. Start gateway trong goroutine riêng, config services map → `localhost:PORT`
4. Monitor health, aggregate metrics
5. Handle SIGTERM → graceful shutdown all goroutines (reverse order)

### Supervisor Pattern (Phased Startup)

```
Phase 0 — Platform Layer:     sm-auth, sm-analytics
Phase 1 — Data Layer:         sm-project, sm-profile
Phase 2 — Intelligence:       sm-engine, sm-document, sm-memory, sm-search
Phase 3 — Integrations:       sm-connector, sm-mcp
Phase 4 — Gateway:            vnp-gateway         (REST + MCP + Auth)
```

Shutdown: reverse order (Gateway → Integrations → Intelligence → Data → Platform)

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)

```
TASK-001: Project setup + unified config           ← Foundation (no deps)
TASK-002: Service supervisor (lifecycle manager)   ← After TASK-001
TASK-003: Embed supermemory services               ← After TASK-002
TASK-004: Embed gateway (config override localhost) ← After TASK-003
TASK-005: Health aggregation + main.go             ← After TASK-004
TASK-006: Dockerfile + Makefile + docker-compose   ← After TASK-005
TASK-007: Documentation (docs/ + changelog)        ← After TASK-006
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | Go module + unified config | TASK | apps/supermemory | - | 1.5h |
| T02 | Service supervisor (goroutine lifecycle) | TASK | apps/supermemory | T01 | 2h |
| T03 | Embed supermemory services | TASK | apps/supermemory | T02 | 4h |
| T04 | Embed gateway | TASK | apps/supermemory | T03 | 2h |
| T05 | Health aggregation + main.go | TASK | apps/supermemory | T04 | 1.5h |
| T06 | Dockerfile + Makefile + docker-compose | TASK | apps/supermemory | T05 | 2h |
| T07 | Documentation | TASK | apps/supermemory | T06 | 1h |

## Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|---|---|---|---|---|---|
| T01 | Go module + unified config | ✅ Done | - | - | |
| T02 | Service supervisor | ✅ Done | - | - | |
| T03 | Embed supermemory services | ✅ Done | - | - | |
| T04 | Embed gateway | ✅ Done | - | - | |
| T05 | Health aggregation + main.go | ✅ Done | - | - | |
| T06 | Dockerfile + Makefile + compose | ✅ Done | - | - | |
| T07 | Documentation | ✅ Done | - | - | |

## Acceptance Criteria (Solution Level)

- [x] SOL-AC-1: `go build ./cmd/supermemory/` build thành công, 1 binary duy nhất
- [x] SOL-AC-2: Binary khởi động tất cả supermemory services + gateway trong 1 process
- [x] SOL-AC-3: REST API qua gateway hoạt động đúng
- [x] SOL-AC-4: gRPC localhost communication hoạt động giữa gateway và services
- [x] SOL-AC-5: NATS events hoạt động
- [x] SOL-AC-6: Graceful shutdown tất cả goroutines (ordered: gateway → services)
- [x] SOL-AC-7: Health checks aggregated từ tất cả embedded services (`/healthz`, `/readyz`)
- [x] SOL-AC-8: **ZERO lines changed** trong `services/` và `gateway/` directories
- [x] SOL-AC-9: Docker image build thành công, single binary
- [x] SOL-AC-10: Các docs được sinh đầy đủ theo document-catalog
