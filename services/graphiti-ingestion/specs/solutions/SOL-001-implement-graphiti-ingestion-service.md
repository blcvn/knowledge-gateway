---
id: SOL-001
title: Implement graphiti-ingestion Standalone Saga Orchestrator
service: graphiti-ingestion
version: 1.0.0
status: Approved
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_cr: null
approved_by: VNP Memory — Architecture Team
---

## Yêu Cầu Gốc

Implement `graphiti-ingestion` — standalone episode ingestion saga orchestrator. This service coordinates the multi-step pipeline across `graphiti-knowledge` and `graphiti-store` services via gRPC. Used when independent scaling of ingestion and knowledge processing is required (production high-throughput scenarios).

> **Note**: `graphiti-pipeline` is the consolidated alternative (ingestion + knowledge in single binary). Both share the same proto interface and can be swapped.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---------|---------------|-----------------|
| graphiti-ingestion | New implementation | Cao — saga orchestrator |
| graphiti-knowledge | Consumer | Trung bình — all extraction calls delegated here |
| graphiti-store | Consumer | Trung bình — bulk persistence calls |
| vnp-gateway | Config update | Thấp — alternative route to :9021 |

### Ràng Buộc Kiến Trúc

- Same GraphitiIngestionService proto as graphiti-pipeline
- gRPC calls to graphiti-knowledge for extraction/resolution (NOT local)
- gRPC calls to graphiti-store for persistence
- Saga state in PostgreSQL
- Per-group serialization for consistency

## Kế Hoạch Triển Khai

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Phụ thuộc | Ước tính |
|----|----------|-----------|-----------|----------|
| FEAT-ING-001 | Domain layer (Episode, Saga, pipeline types) | FEAT | — | 4h |
| FEAT-ING-002 | Usecase layer (saga orchestrator + gRPC delegation) | FEAT | FEAT-ING-001 | 8h |
| FEAT-ING-003 | gRPC handlers + knowledge/store clients | FEAT | FEAT-ING-002 | 8h |
| FEAT-ING-004 | PostgreSQL repository (saga state + episodes) | FEAT | FEAT-ING-002 | 4h |
| FEAT-ING-005 | NATS publisher + event adapter | FEAT | FEAT-ING-002 | 3h |
| FEAT-ING-006 | Infrastructure (config, server, wire, OTel) | FEAT | FEAT-ING-003..005 | 4h |
| QA-ING-001 | Unit + integration tests | QA | FEAT-ING-006 | 6h |

### Rollback Plan

Service is new — rollback = remove Docker image. Can swap to graphiti-pipeline.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: IngestEpisode delegates to graphiti-knowledge via gRPC (not local)
- [ ] SOL-AC-2: Saga orchestrator coordinates 7-step pipeline with compensating actions
- [ ] SOL-AC-3: Per-group serialization prevents concurrent ingestion
- [ ] SOL-AC-4: Circuit breaker protects against graphiti-knowledge/store failures
- [ ] SOL-AC-5: Same proto interface as graphiti-pipeline (swappable)
- [ ] SOL-AC-6: Unit test coverage ≥ 80%

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|----|------|--------|----------|--------|---------|
| FEAT-ING-001 | Domain layer | ✅ Done | AI | — | |
| FEAT-ING-002 | Usecase layer | ✅ Done | AI | — | |
| FEAT-ING-003 | gRPC handlers + clients | ✅ Done | AI | — | |
| FEAT-ING-004 | PostgreSQL repository | ✅ Done | AI | — | |
| FEAT-ING-005 | NATS publisher | ✅ Done | AI | — | |
| FEAT-ING-006 | Infrastructure | ✅ Done | AI | — | |
| QA-ING-001 | Tests | ✅ Done | AI | — | |
