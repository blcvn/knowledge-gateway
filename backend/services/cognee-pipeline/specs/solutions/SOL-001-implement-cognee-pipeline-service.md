---
id: SOL-001
title: Implement Cognee Domain Services — Pipeline + Ingestion + Cognify + Search
service: cross-service
version: 1.0.0
status: Approved
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_cr: null
approved_by: VNP Memory — Architecture Team
---

## Yêu Cầu Gốc

Implement 4 Cognee domain services từ architectural specs đã approved:
1. **cognee-pipeline** — Consolidated ingestion + cognify (single binary)
2. **cognee-ingestion** — Standalone ingestion (shared proto interface)
3. **cognee-cognify** — Standalone cognify (shared proto interface)
4. **cognee-search** — 15-strategy retrieval engine

> **Mô hình triển khai**: `cognee-pipeline` là consolidated service chạy cả ingestion + cognify trong 1 binary. `cognee-ingestion` và `cognee-cognify` là standalone services dùng khi cần scale riêng. Cùng proto interface, có thể swap.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---------|---------------|-----------------|
| cognee-pipeline | New implementation | Cao — core consolidated service |
| cognee-ingestion | New implementation | Cao — standalone ingestion |
| cognee-cognify | New implementation | Cao — standalone cognify |
| cognee-search | New implementation | Cao — retrieval engine |
| vnp-gateway | Config update | Thấp — route registration |
| vnp-admin | Config update | Thấp — health check targets |

### Breaking Changes

- [ ] Không có breaking change (tất cả là service mới)
- [ ] Proto definitions mới cần được generate

### Ràng Buộc Kiến Trúc

- Clean Architecture 4-layer bắt buộc: domain → usecase(+port) → adapter → infra
- gRPC internal communication, REST chỉ qua vnp-gateway
- NATS JetStream cho async events
- Multi-tenant isolation qua `x-tenant-id` gRPC metadata
- Google Wire cho DI
- Shared `pkg/` adapters cho GraphDB, VectorDB, LLM, Storage

## Giải Pháp Đề Xuất

### Approach

Implement theo thứ tự dependency:
1. **Proto definitions** trước — shared contract
2. **cognee-ingestion** — không dependency nào, entry point cho data
3. **cognee-cognify** — depends on ingestion data
4. **cognee-pipeline** — merge của 2 service trên, local function call
5. **cognee-search** — depends on cognify output (graph + vectors)

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|-------------|---------------|
| Implement tất cả cùng lúc | Quá phức tạp, không theo dependency order |
| Chỉ implement pipeline (skip standalone) | Cần standalone cho production scale |

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)

```
Phase 1: Foundation (song song)
  FEAT-ING-001: Ingestion domain + usecase
  FEAT-COG-001: Cognify domain + usecase
  FEAT-SEA-001: Search domain + usecase

Phase 2: Adapters (song song, sau Phase 1)
  FEAT-ING-002: Ingestion adapters (gRPC + NATS + repos)
  FEAT-COG-002: Cognify adapters (gRPC + NATS + repos)
  FEAT-SEA-002: Search adapters (gRPC + retrievers + repos)

Phase 3: Infrastructure (sau Phase 2)
  FEAT-ING-003: Ingestion infra (config + server + wire)
  FEAT-COG-003: Cognify infra (config + server + wire)
  FEAT-SEA-003: Search infra (config + server + wire)

Phase 4: Pipeline Consolidation (sau Phase 3)
  FEAT-PIP-001: Merge ingestion + cognify vào single binary

Phase 5: Integration Testing
  QA-001: Cross-service integration tests
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|----|----------|-----------|---------|-----------|----------|
| FEAT-ING-001 | Ingestion domain + usecase layer | FEAT | cognee-ingestion | — | 8h |
| FEAT-ING-002 | Ingestion adapter layer | FEAT | cognee-ingestion | FEAT-ING-001 | 8h |
| FEAT-ING-003 | Ingestion infra + wire | FEAT | cognee-ingestion | FEAT-ING-002 | 4h |
| FEAT-COG-001 | Cognify domain + usecase layer | FEAT | cognee-cognify | — | 12h |
| FEAT-COG-002 | Cognify adapter layer | FEAT | cognee-cognify | FEAT-COG-001 | 12h |
| FEAT-COG-003 | Cognify infra + wire | FEAT | cognee-cognify | FEAT-COG-002 | 4h |
| FEAT-SEA-001 | Search domain + usecase layer | FEAT | cognee-search | — | 8h |
| FEAT-SEA-002 | Search adapter layer (15 retrievers) | FEAT | cognee-search | FEAT-SEA-001 | 16h |
| FEAT-SEA-003 | Search infra + wire | FEAT | cognee-search | FEAT-SEA-002 | 4h |
| FEAT-PIP-001 | Pipeline consolidation | FEAT | cognee-pipeline | ING-003, COG-003 | 6h |
| QA-001 | Cross-service integration tests | QA | cross-service | All FEAT | 8h |

### Rollback Plan

Mỗi service được deploy độc lập — rollback bằng cách revert Docker image tag per-service.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: Tất cả 4 services pass `make build` và `make test`
- [ ] SOL-AC-2: gRPC API hoạt động end-to-end qua vnp-gateway
- [ ] SOL-AC-3: NATS events flow: ingestion → cognify → search
- [ ] SOL-AC-4: Multi-tenant isolation verified (tenant A không thấy data tenant B)
- [ ] SOL-AC-5: Docker Compose up toàn bộ 4 services + infra
- [ ] SOL-AC-6: Docs (README, api, architecture, config, data-model, runbook, changelog) updated

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|----|------|--------|----------|--------|---------|
| FEAT-ING-001 | Ingestion domain + usecase | ⏳ Draft | AI | — | |
| FEAT-ING-002 | Ingestion adapter layer | ⏳ Draft | AI | — | |
| FEAT-ING-003 | Ingestion infra + wire | ⏳ Draft | AI | — | |
| FEAT-COG-001 | Cognify domain + usecase | ⏳ Draft | AI | — | |
| FEAT-COG-002 | Cognify adapter layer | ⏳ Draft | AI | — | |
| FEAT-COG-003 | Cognify infra + wire | ⏳ Draft | AI | — | |
| FEAT-SEA-001 | Search domain + usecase | ⏳ Draft | AI | — | |
| FEAT-SEA-002 | Search adapter layer | ⏳ Draft | AI | — | |
| FEAT-SEA-003 | Search infra + wire | ⏳ Draft | AI | — | |
| FEAT-PIP-001 | Pipeline consolidation | ⏳ Draft | AI | — | |
| QA-001 | Integration tests | ⏳ Draft | AI | — | |
