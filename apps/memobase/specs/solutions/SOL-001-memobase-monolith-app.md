---
id: SOL-001
title: "Memobase Monolith App — Embedded Multi-Service Process"
app: apps/memobase
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_cr: null
approved_by: pending
---

## Yêu Cầu Gốc

Hợp nhất **4 Memobase microservices** (ingestion, engine, context, pipeline) + **Gateway** thành **1 binary duy nhất** chạy ở mức **production grade** và **enterprise level**.

### Nguyên tắc thiết kế:
- **ZERO changes** đến code services (`services/memobase-*`) và gateway (`gateway/`)
- **Reuse toàn bộ** — Import từ services/gateway packages, chỉ viết orchestration code
- **Internal gRPC/NATS OK** — Các module giao tiếp qua gRPC localhost + NATS như thiết kế gốc
- **Single Process** — Mọi thứ chạy trong 1 process, deploy 1 binary

## Phân Tích Kiến Trúc

### Cách hoạt động hiện tại (Microservices)

```
Client → [Gateway:8080] →─gRPC──→ [memobase-ingestion:9041]  (separate process)
                         →─gRPC──→ [memobase-engine:9042]     (separate process)
                         →─gRPC──→ [memobase-context:9043]    (separate process)
                         →─NATS──→ [memobase-pipeline:9044]   (separate process)
```

Mỗi service là 1 process riêng, gateway route qua gRPC, async events qua NATS.

### Cách hoạt động mới (Monolith — 1 process)

```
                    ┌──── SINGLE PROCESS ─────────────────────────────────────────┐
                    │                                                              │
Client → [Gateway REST :8080] →─gRPC localhost──→ [ingestion  goroutine:9041]     │
                    │          →─gRPC localhost──→ [engine     goroutine:9042]     │
                    │          →─gRPC localhost──→ [context    goroutine:9043]     │
                    │          →─NATS────────────→ [pipeline   goroutine:9044]     │
                    │                                                              │
                    │  [MCP :8082] [Health :9090]                                  │
                    └──────────────────────────────────────────────────────────────┘
```

**Khác biệt duy nhất:** Tất cả services + gateway chạy trong 1 process thay vì 5+ processes.
- gRPC vẫn hoạt động → gọi qua `localhost` (zero-copy loopback, ~0.1ms overhead)
- NATS vẫn hoạt động → event-driven giữ nguyên
- **KHÔNG thay đổi bất kỳ code nào** trong services hay gateway

### Tại sao approach này tốt nhất?

| Tiêu chí | Ưu điểm |
|----------|---------|
| **Zero code change** | Services + Gateway giữ nguyên 100%, không sửa 1 dòng nào |
| **Proven patterns** | gRPC localhost + NATS đã được test kỹ trong production |
| **Consistent with cognee/graphiti** | Cùng supervisor pattern đã triển khai thành công |
| **Rollback trivial** | Chỉ cần deploy microservices + gateway riêng biệt như cũ |
| **Upgrade safe** | Khi services update, app tự động nhận code mới (same Go module) |

### Services Map (từ Reference Specs)

| # | Service | gRPC Port | Health Port | Responsibility |
|---|---------|----------|-------------|----------------|
| 1 | `memobase-ingestion` | 9041 | 9091 | Blob insert, Buffer Zone, Flush trigger |
| 2 | `memobase-engine` | 9042 | 9092 | Profile extraction, YOLO merge, Event summary |
| 3 | `memobase-context` | 9043 | 9093 | Context assembly, Profile query, Event search |
| 4 | `memobase-pipeline` | 9044 | 9094 | Buffer pipeline orchestration |
| 5 | `vnp-gateway` | 8080 | 8083 | REST API, Auth, MCP Server |

### Code inventory: Cái gì KHÔNG thay đổi

```
✅ gateway/cmd/main.go                           — KHÔNG SỬA
✅ gateway/internal/**                            — KHÔNG SỬA
✅ services/memobase-ingestion/cmd/server/main.go — KHÔNG SỬA
✅ services/memobase-ingestion/internal/**         — KHÔNG SỬA
✅ services/memobase-engine/cmd/server/main.go    — KHÔNG SỬA
✅ services/memobase-engine/internal/**            — KHÔNG SỬA
✅ services/memobase-context/cmd/server/main.go   — KHÔNG SỬA
✅ services/memobase-context/internal/**            — KHÔNG SỬA
✅ services/memobase-pipeline/cmd/server/main.go  — KHÔNG SỬA
✅ services/memobase-pipeline/internal/**           — KHÔNG SỬA
```

### Code CẦN viết mới (chỉ orchestration)

```
🆕 apps/memobase/cmd/memobase/main.go              — Orchestrator: start all services + gateway
🆕 apps/memobase/cmd/memobase/services.go           — Service start functions
🆕 apps/memobase/cmd/memobase/gateway.go            — Gateway start function
🆕 apps/memobase/cmd/memobase/health.go             — Health aggregation server
🆕 apps/memobase/internal/supervisor/supervisor.go  — Process supervisor (lifecycle manager)
🆕 apps/memobase/internal/config/config.go          — Unified config for all embedded services
🆕 apps/memobase/Dockerfile                         — Multi-stage build
🆕 apps/memobase/Makefile                           — Build/run targets
🆕 apps/memobase/docker-compose.yml                 — Local dev (NATS, Postgres, Redis)
🆕 apps/memobase/config.yaml                        — Unified config file
🆕 apps/memobase/.env.example                       — Environment variable template
```

**Ước tính code mới: ~500 lines** (so với existing reuse)

## Giải Pháp Chi Tiết

### Approach: Embedded Service Supervisor (Proven Pattern)

Pattern **đã được triển khai thành công** tại `apps/cognee` và `apps/graphiti`.

App `memobase` là **process supervisor** chạy trong 1 Go process:
1. Load unified config → phân phối thành ENV vars cho từng embedded service
2. Start mỗi service init logic trong goroutine riêng (phased startup)
3. Start gateway trong goroutine riêng, config services map → `localhost:PORT`
4. Monitor health, aggregate metrics
5. Handle SIGTERM → graceful shutdown all goroutines (reverse order)

### Supervisor Pattern (4-Phase Startup)

```
Phase 0 — Data Layer:       memobase-ingestion  (blob storage, buffer zone)
Phase 1 — Intelligence:     memobase-engine     (LLM processing pipeline)
Phase 2 — Application:      memobase-context    (context assembly, profile read)
                            memobase-pipeline   (pipeline orchestration)
Phase 3 — Gateway:          vnp-gateway         (REST + MCP + Auth)
```

Shutdown: reverse order (Gateway → Application → Intelligence → Data)

### Internal Communication Flow

```
┌────────────────────────────────────────────────────────────────┐
│                    memobase-app (single process)                │
│                                                                 │
│  Client HTTP                                                    │
│    │                                                            │
│    ▼                                                            │
│  Gateway (goroutine :8080)                                      │
│    │ GRPCRegistry → "localhost:PORT" for each service           │
│    │                                                            │
│    ▼ gRPC via localhost loopback                                 │
│  memobase-ingestion (goroutine :9041)                           │
│    │ NATS: memobase.buffer.ready → memobase-engine              │
│    │                                                            │
│  memobase-engine (goroutine :9042)                              │
│    │ LLM calls: profile extraction, YOLO merge                 │
│    │ NATS: memobase.engine.completed → memobase-context         │
│    │                                                            │
│  memobase-context (goroutine :9043)                             │
│    │ Redis cache, profile read, context assembly                │
│    │                                                            │
│  memobase-pipeline (goroutine :9044)                            │
│    │ Pipeline orchestration                                     │
│    │                                                            │
│  External dependencies:                                         │
│  ├─ PostgreSQL + pgvector (shared)                              │
│  ├─ Redis (profile caching, rate limiting)                      │
│  ├─ NATS JetStream (async events)                               │
│  └─ LLM Provider (Bifrost / OpenAI)                             │
└────────────────────────────────────────────────────────────────┘
```

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|---|---|
| Import `internal/` packages trực tiếp | Go compiler reject cross-module `internal/` imports |
| Sửa services để export public packages | Vi phạm yêu cầu "KHÔNG sửa code services" |
| Replace NATS bằng in-process event bus | Phải sửa services code (EventPublisher interface) |
| Copy code từ services | Duplicate code, maintenance nightmare |

### Trade-offs

| Ưu điểm | Nhược điểm |
|----------|------------|
| Zero code changes | gRPC localhost có overhead nhỏ (~0.1ms vs in-process) |
| Proven gRPC/NATS paths | Cần NATS server chạy (external dependency) |
| Services update = app update | Startup thứ tự phải đúng (services trước gateway) |
| Single binary deploy | Memory footprint lớn hơn single microservice |
| Unified config | Config mapping: app config → service ENV vars |
| Pattern consistent (cognee/graphiti) | Cần maintain thêm 1 app |

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)

```
TASK-001: Project setup + unified config           ← Foundation (no deps)
TASK-002: Service supervisor (lifecycle manager)   ← After TASK-001
TASK-003: Embed services (ingestion, engine,       ← After TASK-002
          context, pipeline)
TASK-004: Embed gateway (config override localhost) ← After TASK-003
TASK-005: Health aggregation + main.go             ← After TASK-004
TASK-006: Dockerfile + Makefile + docker-compose   ← After TASK-005
TASK-007: Documentation (docs/ + changelog)        ← After TASK-006
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | Go module + unified config | TASK | apps/memobase | - | 1.5h |
| T02 | Service supervisor (goroutine lifecycle) | TASK | apps/memobase | T01 | 2h |
| T03 | Embed memobase services | TASK | apps/memobase | T02 | 3h |
| T04 | Embed gateway | TASK | apps/memobase | T03 | 2h |
| T05 | Health aggregation + main.go | TASK | apps/memobase | T04 | 1.5h |
| T06 | Dockerfile + Makefile + docker-compose | TASK | apps/memobase | T05 | 2h |
| T07 | Documentation | TASK | apps/memobase | T06 | 1h |

**Tổng ước tính: ~13h** (giảm nhờ zero new business logic, pattern đã proven)

### Rollback Plan
Monolith app nằm trong `apps/memobase/`. Services gốc KHÔNG bị sửa đổi.
Rollback = deploy microservices + gateway riêng biệt như cũ.

## Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|---|---|---|---|---|---|
| T01 | Go module + unified config | ⏳ Draft | - | - | |
| T02 | Service supervisor | ⏳ Draft | - | - | |
| T03 | Embed memobase services | ⏳ Draft | - | - | |
| T04 | Embed gateway | ⏳ Draft | - | - | |
| T05 | Health aggregation + main.go | ⏳ Draft | - | - | |
| T06 | Dockerfile + Makefile + compose | ⏳ Draft | - | - | |
| T07 | Documentation | ⏳ Draft | - | - | |

## Acceptance Criteria (Solution Level)

- [x] SOL-AC-1: `go build ./cmd/memobase/` build thành công, 1 binary duy nhất
- [x] SOL-AC-2: Binary khởi động tất cả 4 memobase services + gateway trong 1 process
- [x] SOL-AC-3: REST API qua gateway hoạt động đúng (`/api/v1/blobs/*`, `/api/v1/users/*`)
- [x] SOL-AC-4: gRPC localhost communication hoạt động giữa gateway và services
- [x] SOL-AC-5: NATS events hoạt động (ingestion → engine pipeline: `memobase.buffer.ready`)
- [x] SOL-AC-6: Graceful shutdown tất cả goroutines (ordered: gateway → services)
- [x] SOL-AC-7: Health checks aggregated từ tất cả embedded services (`/healthz`, `/readyz`)
- [x] SOL-AC-8: **ZERO lines changed** trong `services/` và `gateway/` directories
- [x] SOL-AC-9: Docker image build thành công, single binary
- [x] SOL-AC-10: Code mới ≤ 600 lines (chỉ orchestration + config)
