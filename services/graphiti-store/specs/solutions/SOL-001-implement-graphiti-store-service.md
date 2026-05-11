---
id: SOL-001
title: Implement graphiti-store Graph Database Abstraction Service
service: graphiti-store
version: 1.0.0
status: Approved
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_cr: null
approved_by: VNP Memory — Architecture Team
---

## Yêu Cầu Gốc

Implement `graphiti-store` — graph database abstraction layer với pluggable backend drivers (Neo4j primary, FalkorDB/Kuzu/Neptune pluggable). Service cung cấp unified gRPC API cho CRUD nodes, edges, communities, bulk operations, search primitives, và index management.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---------|---------------|-----------------|
| graphiti-store | New implementation | Cao — core storage abstraction |
| graphiti-pipeline | Consumer | Trung bình — depends on SaveBulk, RollbackBulk |
| graphiti-search | Consumer | Trung bình — depends on search primitives |
| vnp-gateway | Config update | Thấp — route registration |

### Breaking Changes

- [ ] Không có breaking change (service mới)

### Ràng Buộc Kiến Trúc

- GraphDriver interface = Strategy pattern cho backend selection
- Driver implementation MUST implement ALL repository interfaces
- Transaction support via unit-of-work pattern
- Index management abstracted across backends
- Connection pool per tenant (optional advanced mode)
- No LLM/AI logic in store service (pure data operations)

## Giải Pháp Đề Xuất

### Approach — Driver Abstraction

```go
type GraphDriver interface {
    NodeRepository
    EdgeRepository
    CommunityRepository
    SearchRepository
    IndexRepository
    BulkRepository
    TransactionManager
}
```

Backend selection at startup via `DRIVER_PROVIDER` config. Neo4j driver = priority.

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện

```
Phase 1: Foundation
  FEAT-STO-001: Domain (EntityNode, EpisodicNode, CommunityNode, EntityEdge, SagaNode)
  FEAT-STO-002: Usecase + port interfaces (NodeRepo, EdgeRepo, SearchRepo, BulkRepo, etc.)

Phase 2: Neo4j Driver (Primary)
  FEAT-STO-003: Neo4j node repository adapter
  FEAT-STO-004: Neo4j edge repository adapter (bi-temporal)
  FEAT-STO-005: Neo4j search primitives (cosine, fulltext, BFS)
  FEAT-STO-006: Neo4j bulk operations + transaction management
  FEAT-STO-007: Neo4j index management (vector, fulltext, composite)

Phase 3: gRPC + Infra
  FEAT-STO-008: gRPC handlers + proto mapping
  FEAT-STO-009: Infrastructure (config, server, wire, OTel)

Phase 4: Quality
  QA-STO-001: Unit + integration tests (Neo4j testcontainer)
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Phụ thuộc | Ước tính |
|----|----------|-----------|-----------|----------|
| FEAT-STO-001 | Domain layer (graph entities) | FEAT | — | 6h |
| FEAT-STO-002 | Usecase + port interfaces | FEAT | FEAT-STO-001 | 6h |
| FEAT-STO-003 | Neo4j node repository | FEAT | FEAT-STO-002 | 6h |
| FEAT-STO-004 | Neo4j edge repository (bi-temporal) | FEAT | FEAT-STO-002 | 8h |
| FEAT-STO-005 | Neo4j search primitives | FEAT | FEAT-STO-002 | 8h |
| FEAT-STO-006 | Neo4j bulk ops + transactions | FEAT | FEAT-STO-003,004 | 6h |
| FEAT-STO-007 | Neo4j index management | FEAT | FEAT-STO-002 | 4h |
| FEAT-STO-008 | gRPC handlers | FEAT | FEAT-STO-002 | 6h |
| FEAT-STO-009 | Infrastructure | FEAT | FEAT-STO-008 | 4h |
| QA-STO-001 | Tests (unit + Neo4j integration) | QA | FEAT-STO-009 | 10h |

### Rollback Plan

Service is new — rollback = remove Docker image.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: CRUD operations work for all node types (Entity, Episodic, Community, Saga)
- [ ] SOL-AC-2: Edge CRUD with bi-temporal fields (valid_at, invalid_at, expired_at)
- [ ] SOL-AC-3: Cosine similarity search returns top-K similar entities by embedding
- [ ] SOL-AC-4: Fulltext search returns BM25-ranked results
- [ ] SOL-AC-5: BFS traversal returns entity subgraph to configurable depth
- [ ] SOL-AC-6: SaveBulk persists nodes + edges atomically in single transaction
- [ ] SOL-AC-7: BuildIndices creates vector + fulltext + composite indexes
- [ ] SOL-AC-8: DeleteByGroupID removes all tenant data (purge)
- [ ] SOL-AC-9: GraphDriver interface allows Neo4j swap without usecase changes
- [ ] SOL-AC-10: Unit test coverage ≥ 80%

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|----|------|--------|----------|--------|---------|
| FEAT-STO-001 | Domain layer | ⏳ Draft | AI | — | |
| FEAT-STO-002 | Usecase + ports | ⏳ Draft | AI | — | |
| FEAT-STO-003 | Neo4j node repo | ⏳ Draft | AI | — | |
| FEAT-STO-004 | Neo4j edge repo | ⏳ Draft | AI | — | |
| FEAT-STO-005 | Neo4j search primitives | ⏳ Draft | AI | — | |
| FEAT-STO-006 | Neo4j bulk + transactions | ⏳ Draft | AI | — | |
| FEAT-STO-007 | Neo4j index management | ⏳ Draft | AI | — | |
| FEAT-STO-008 | gRPC handlers | ⏳ Draft | AI | — | |
| FEAT-STO-009 | Infrastructure | ⏳ Draft | AI | — | |
| QA-STO-001 | Tests | ⏳ Draft | AI | — | |
