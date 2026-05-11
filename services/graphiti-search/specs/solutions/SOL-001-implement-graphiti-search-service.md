---
id: SOL-001
title: Implement graphiti-search Hybrid Search Service
service: graphiti-search
version: 1.0.0
status: Approved
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_cr: null
approved_by: VNP Memory — Architecture Team
---

## Yêu Cầu Gốc

Implement `graphiti-search` — hybrid search service combining vector similarity (cosine), full-text (BM25), BFS graph traversal with configurable multi-strategy reranking over the Graphiti temporal knowledge graph.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---------|---------------|-----------------|
| graphiti-search | New implementation | Cao — hybrid search engine |
| graphiti-store | Unchanged | Thấp — consumed via gRPC for search primitives |
| graphiti-pipeline | Unchanged | Thấp — consumes NATS events for cache invalidation |
| vnp-gateway | Config update | Thấp — route registration |

### Breaking Changes

- [ ] Không có breaking change (service mới)

### Ràng Buộc Kiến Trúc

- Clean Architecture 4-layer
- gRPC internal, REST qua vnp-gateway
- Redis for result caching + NATS for cache invalidation
- Reranking logic in search service (not knowledge service)
- Multi-tenant isolation qua group_id

## Giải Pháp Đề Xuất

### Approach

Implement inside-out theo Clean Architecture:
1. Domain: SearchQuery, SearchConfig, SearchResult, Reranker types
2. Usecase: HybridSearch orchestrator, individual search methods, reranking pipeline
3. Adapter: gRPC handlers, graphiti-store client, Redis cache, NATS subscriber
4. Infra: config, server, wire, OTel

### Key Design Decision: Reranker Architecture

5 reranking strategies implementable as interface:
- **RRF** (Reciprocal Rank Fusion) — default, combines multiple ranked lists
- **MMR** (Maximal Marginal Relevance) — diversity-promoting
- **Cross-Encoder** — neural reranking via graphiti-pipeline
- **Node Distance** — graph proximity weighting
- **Episode Mentions** — frequency-based relevance

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện

```
Phase 1: Foundation
  FEAT-SEA-001: Domain layer (search types, reranker interfaces)
  FEAT-SEA-002: Usecase layer (hybrid search, reranking pipeline)

Phase 2: Adapters
  FEAT-SEA-003: gRPC handlers (HybridSearch, NodeSearch, EdgeSearch, CommunitySearch)
  FEAT-SEA-004: graphiti-store client (search primitives delegation)
  FEAT-SEA-005: Redis cache adapter + NATS cache invalidation
  FEAT-SEA-006: Reranker implementations (5 strategies)

Phase 3: Infrastructure
  FEAT-SEA-007: Infra (config, server, wire, OTel)

Phase 4: Quality
  QA-SEA-001: Unit + integration tests
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Phụ thuộc | Ước tính |
|----|----------|-----------|-----------|----------|
| FEAT-SEA-001 | Domain layer | FEAT | — | 4h |
| FEAT-SEA-002 | Usecase layer | FEAT | FEAT-SEA-001 | 8h |
| FEAT-SEA-003 | gRPC handlers | FEAT | FEAT-SEA-002 | 4h |
| FEAT-SEA-004 | Store client adapter | FEAT | FEAT-SEA-002 | 4h |
| FEAT-SEA-005 | Redis cache + NATS invalidation | FEAT | FEAT-SEA-002 | 6h |
| FEAT-SEA-006 | Reranker implementations (5 strategies) | FEAT | FEAT-SEA-002 | 10h |
| FEAT-SEA-007 | Infrastructure | FEAT | FEAT-SEA-003..006 | 4h |
| QA-SEA-001 | Tests (unit + integration) | QA | FEAT-SEA-007 | 8h |

### Rollback Plan

Service is new — rollback = remove Docker image.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: HybridSearch returns results combining cosine + BM25 + BFS
- [ ] SOL-AC-2: All 5 reranking strategies produce valid ranked results
- [ ] SOL-AC-3: Redis cache hit returns results in <10ms
- [ ] SOL-AC-4: NATS cache invalidation triggers on `graphiti.episode.ingested`
- [ ] SOL-AC-5: Multi-tenant isolation (tenant A cannot see tenant B results)
- [ ] SOL-AC-6: OTel traces + Prometheus metrics operational
- [ ] SOL-AC-7: Unit test coverage ≥ 80%

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|----|------|--------|----------|--------|---------|
| FEAT-SEA-001 | Domain layer | ⏳ Draft | AI | — | |
| FEAT-SEA-002 | Usecase layer | ⏳ Draft | AI | — | |
| FEAT-SEA-003 | gRPC handlers | ⏳ Draft | AI | — | |
| FEAT-SEA-004 | Store client | ⏳ Draft | AI | — | |
| FEAT-SEA-005 | Redis cache + NATS | ⏳ Draft | AI | — | |
| FEAT-SEA-006 | Rerankers (5 strategies) | ⏳ Draft | AI | — | |
| FEAT-SEA-007 | Infrastructure | ⏳ Draft | AI | — | |
| QA-SEA-001 | Tests | ⏳ Draft | AI | — | |
