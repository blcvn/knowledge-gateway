---
id: SOL-001
title: Implement graphiti-pipeline Consolidated Service
service: graphiti-pipeline
version: 1.0.0
status: Approved
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_cr: null
approved_by: VNP Memory — Architecture Team
---

## Yêu Cầu Gốc

Implement `graphiti-pipeline` — consolidated service merging `graphiti-ingestion` (saga orchestrator) + `graphiti-knowledge` (LLM processing engine) into a single Go binary. This eliminates 5 cross-service gRPC round-trips per episode, reducing P95 latency by ~40%.

> **Deploy model**: `graphiti-pipeline` is the default deployment. `graphiti-ingestion` and `graphiti-knowledge` are standalone alternatives when independent scaling is needed.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---------|---------------|-----------------|
| graphiti-pipeline | New implementation | Cao — core consolidated service |
| graphiti-store | Unchanged | Thấp — consumed via gRPC |
| graphiti-search | Unchanged | Thấp — consumes same NATS events |
| vnp-gateway | Config update | Thấp — route registration to :9021 |

### Breaking Changes

- [ ] Không có breaking change (service mới, cùng proto interface)
- [ ] Proto definitions tương thích backward với graphiti-ingestion/knowledge standalone

### Ràng Buộc Kiến Trúc

- Clean Architecture 4-layer: domain → usecase(+port) → adapter → infra
- Saga steps MUST be local function calls (no internal gRPC)
- gRPC internal communication, REST chỉ qua vnp-gateway
- NATS JetStream cho async events
- Multi-tenant isolation qua `x-tenant-id` gRPC metadata
- Google Wire cho DI
- Bifrost gateway cho all LLM calls (no direct provider calls)

## Giải Pháp Đề Xuất

### Approach

Implement theo Clean Architecture layers, inside-out:
1. **Domain layer** — pure Go types, zero dependencies
2. **Usecase layer** — business logic + port interfaces
3. **Adapter layer** — gRPC handlers, LLM client, DB repos, NATS publisher
4. **Infra layer** — config, server, wire, telemetry

Merge ingestion + knowledge domains into single binary with shared usecase orchestration.

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|-------------|---------------|
| Keep separate services (ingestion + knowledge) | 5 gRPC round-trips per episode, higher latency |
| Single domain package (no subdomain split) | Ingestion and knowledge are distinct bounded contexts |

### Trade-offs

- **Ưu điểm**: ~40% latency reduction, simpler deployment, fewer failure points
- **Nhược điểm**: Can't scale ingestion and knowledge independently (use standalone services for that)

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)

```
Phase 1: Foundation (song song)
  FEAT-PIP-001: Domain layer (ingestion + knowledge entities)
  FEAT-PIP-002: Usecase layer (saga + extraction + resolution)

Phase 2: Adapters (sau Phase 1)
  FEAT-PIP-003: gRPC handlers (ingestion + knowledge)
  FEAT-PIP-004: LLM adapter (Bifrost client + prompt registry)
  FEAT-PIP-005: Repository adapters (PostgreSQL + Neo4j reader)
  FEAT-PIP-006: Event adapter (NATS publisher)
  FEAT-PIP-007: Store client adapter (gRPC → graphiti-store)

Phase 3: Infrastructure (sau Phase 2)
  FEAT-PIP-008: Infra layer (config + server + wire + telemetry)

Phase 4: Quality
  QA-PIP-001: Unit tests (coverage ≥ 80%)
  QA-PIP-002: Integration tests (end-to-end saga)
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|----|----------|-----------|---------|-----------|----------|
| FEAT-PIP-001 | Domain layer (entities, value objects, errors) | FEAT | graphiti-pipeline | — | 6h |
| FEAT-PIP-002 | Usecase layer (saga, extraction, resolution) | FEAT | graphiti-pipeline | FEAT-PIP-001 | 12h |
| FEAT-PIP-003 | gRPC handler adapters | FEAT | graphiti-pipeline | FEAT-PIP-002 | 6h |
| FEAT-PIP-004 | LLM adapter (Bifrost + prompts) | FEAT | graphiti-pipeline | FEAT-PIP-002 | 8h |
| FEAT-PIP-005 | Repository adapters (PostgreSQL + Neo4j) | FEAT | graphiti-pipeline | FEAT-PIP-002 | 8h |
| FEAT-PIP-006 | NATS event publisher adapter | FEAT | graphiti-pipeline | FEAT-PIP-002 | 3h |
| FEAT-PIP-007 | graphiti-store gRPC client adapter | FEAT | graphiti-pipeline | FEAT-PIP-002 | 4h |
| FEAT-PIP-008 | Infrastructure (config, server, wire, OTel) | FEAT | graphiti-pipeline | FEAT-PIP-003..007 | 6h |
| QA-PIP-001 | Unit tests (≥80% coverage) | QA | graphiti-pipeline | FEAT-PIP-008 | 8h |
| QA-PIP-002 | Integration tests (saga end-to-end) | QA | graphiti-pipeline | QA-PIP-001 | 6h |

### Rollback Plan

Service is new — rollback = remove Docker image. No data migration needed.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: `make build-graphiti-pipeline` succeeds, binary < 30MB
- [ ] SOL-AC-2: IngestEpisode RPC processes episode through full 6-step saga
- [ ] SOL-AC-3: BulkIngest RPC processes batch with cross-episode deduplication
- [ ] SOL-AC-4: ExtractEntities/ResolveEntities/ExtractEdges/ResolveEdges RPCs work independently
- [ ] SOL-AC-5: Saga compensating actions execute on failure (rollback via graphiti-store)
- [ ] SOL-AC-6: Per-group serialization prevents concurrent ingestion
- [ ] SOL-AC-7: NATS event `graphiti.episode.ingested` published on saga completion
- [ ] SOL-AC-8: OTel traces visible in Jaeger for full saga pipeline
- [ ] SOL-AC-9: Prometheus metrics exposed on :9094/metrics
- [ ] SOL-AC-10: Unit test coverage ≥ 80%
- [ ] SOL-AC-11: Docker Compose `docker compose up graphiti-pipeline` works

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|----|------|--------|----------|--------|---------|
| FEAT-PIP-001 | Domain layer | ⏳ Draft | AI | — | |
| FEAT-PIP-002 | Usecase layer | ⏳ Draft | AI | — | |
| FEAT-PIP-003 | gRPC handlers | ⏳ Draft | AI | — | |
| FEAT-PIP-004 | LLM adapter | ⏳ Draft | AI | — | |
| FEAT-PIP-005 | Repository adapters | ⏳ Draft | AI | — | |
| FEAT-PIP-006 | NATS publisher | ⏳ Draft | AI | — | |
| FEAT-PIP-007 | Store client | ⏳ Draft | AI | — | |
| FEAT-PIP-008 | Infrastructure | ⏳ Draft | AI | — | |
| QA-PIP-001 | Unit tests | ⏳ Draft | AI | — | |
| QA-PIP-002 | Integration tests | ⏳ Draft | AI | — | |
