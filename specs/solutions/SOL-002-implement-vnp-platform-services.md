---
id: SOL-002
title: Implement VNP Platform Services (5 services)
service: cross-service
version: 1.0.0
status: Approved
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_cr: ADR-0001
approved_by: Software Architect
---

## Yêu Cầu Gốc

Implement Go-native source code for 5 VNP Platform services following Clean Architecture (4-layer model) and the existing TDD specs. Each service must have buildable Go code with gRPC handlers, domain logic, infrastructure wiring, and tests.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Current State | Target State |
|---------|--------------|--------------|
| vnp-admin | docs + tdd.md only, **no Go code** | Full 4-layer Go service |
| vnp-event | docs + tdd.md only, **no Go code** | Full 4-layer Go service |
| vnp-gateway | **Already implemented** (20+ Go files) | Docs/specs alignment only |
| vnp-platform | Scaffolded (9 Go files, domain entities) | Complete usecase + adapter layers |
| vnp-search-hub | docs + tdd.md only, **no Go code** | Full 4-layer Go service |

### Breaking Changes

- [ ] API response format thay đổi? → **KHÔNG** (proto giữ nguyên)
- [ ] Database schema migration cần thiết? → **CÓ** (initial schema creation per data-model.md)
- [ ] Consumer downstream cần cập nhật? → **KHÔNG**

### Ràng Buộc Kiến Trúc

1. Clean Architecture 4-layer: Domain → Usecase → Adapter → Infra
2. gRPC service definitions match `api.md` and `tdd.md`
3. NATS event subjects match `deployment/nats/streams.yaml`
4. Config via environment variables (DATABASE_URL, REDIS_URL, NATS_URL, GRPC_PORT)
5. Health check endpoint on dedicated port
6. Follow gateway patterns (gateway/ is the reference implementation)

## Giải Pháp Đề Xuất

### Approach

Per-service implementation following each service's `specs/tdd.md` as the source of truth for domain model, API contract, and architecture decisions. Gateway is already implemented — only needs spec alignment.

### Thứ Tự Thực Hiện (Dependency Order)

```
T01: vnp-admin   ← No dependencies, foundation service
T02: vnp-event   ← After T01 (subscribes to admin events)
T03: vnp-platform ← After T01+T02 (absorbs admin+event logic)
T04: vnp-search-hub ← After T01 (needs auth context from admin)
T05: vnp-gateway   ← Already implemented, docs alignment only
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | Implement vnp-admin domain + usecase + adapter | FEAT | vnp-admin | — | 1 day |
| T02 | Implement vnp-event domain + usecase + adapter | FEAT | vnp-event | T01 | 1 day |
| T03 | Complete vnp-platform adapter + infra wiring | FEAT | vnp-platform | T01, T02 | 1 day |
| T04 | Implement vnp-search-hub domain + usecase + adapter | FEAT | vnp-search-hub | T01 | 1 day |
| T05 | Align vnp-gateway docs/specs with implementation | QA | vnp-gateway | — | 0.5 day |

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify |
|---|---|---|---|---|
| T01 | vnp-admin implementation | ⏳ Ready | — | — |
| T02 | vnp-event implementation | ⏳ Ready | — | — |
| T03 | vnp-platform completion | ⏳ Ready | — | — |
| T04 | vnp-search-hub implementation | ⏳ Ready | — | — |
| T05 | vnp-gateway docs alignment | ⏳ Ready | — | — |
