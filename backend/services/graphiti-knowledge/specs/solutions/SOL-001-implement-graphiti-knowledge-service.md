---
id: SOL-001
title: Implement graphiti-knowledge LLM Processing Engine
service: graphiti-knowledge
version: 1.0.0
status: Approved
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_cr: null
approved_by: VNP Memory — Architecture Team
---

## Yêu Cầu Gốc

Implement `graphiti-knowledge` — standalone LLM-intensive processing engine for entity extraction, edge extraction, entity/edge resolution, embedding generation, community detection, and cross-encoder reranking. All AI interactions routed through Bifrost gateway.

> **Note**: `graphiti-pipeline` is the consolidated alternative (ingestion + knowledge in single binary). Both share the same proto interface and can be swapped.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---------|---------------|-----------------|
| graphiti-knowledge | New implementation | Cao — LLM processing engine |
| graphiti-ingestion | Consumer | Trung bình — calls extraction RPCs |
| graphiti-store | Consumer | Thấp — reads for resolution queries |
| Bifrost | Consumer | Trung bình — all LLM calls via gateway |

### Ràng Buộc Kiến Trúc

- Same GraphitiKnowledgeService proto as graphiti-pipeline
- Stateless service (no persistent storage of its own, reads from graphiti-store)
- Bifrost gateway for ALL LLM calls (no direct provider SDK)
- Bulkhead pattern for concurrent LLM request limiting
- Prompt template management for all extraction/resolution tasks

## Kế Hoạch Triển Khai

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Phụ thuộc | Ước tính |
|----|----------|-----------|-----------|----------|
| FEAT-KNW-001 | Domain layer (extraction types, prompts, embeddings) | FEAT | — | 4h |
| FEAT-KNW-002 | Usecase layer (extract, resolve, embed, community) | FEAT | FEAT-KNW-001 | 10h |
| FEAT-KNW-003 | gRPC handlers | FEAT | FEAT-KNW-002 | 4h |
| FEAT-KNW-004 | Bifrost LLM client adapter | FEAT | FEAT-KNW-002 | 6h |
| FEAT-KNW-005 | Bifrost embedder adapter | FEAT | FEAT-KNW-002 | 3h |
| FEAT-KNW-006 | graphiti-store gRPC client (read-only) | FEAT | FEAT-KNW-002 | 3h |
| FEAT-KNW-007 | Prompt registry (templates for all extraction tasks) | FEAT | FEAT-KNW-002 | 4h |
| FEAT-KNW-008 | Infrastructure (config, server, wire, OTel) | FEAT | FEAT-KNW-003..007 | 4h |
| QA-KNW-001 | Unit + integration tests | QA | FEAT-KNW-008 | 8h |

### Rollback Plan

Service is new — rollback = remove Docker image. Can swap to graphiti-pipeline.

## Acceptance Criteria (Solution Level)

- [ ] SOL-AC-1: ExtractEntities returns structured entities from LLM response
- [ ] SOL-AC-2: ResolveEntities deduplicates with >0.85 cosine similarity threshold
- [ ] SOL-AC-3: ExtractEdges produces bi-temporal fact triples
- [ ] SOL-AC-4: ResolveEdges detects contradictions and invalidates old edges
- [ ] SOL-AC-5: GenerateEmbedding produces vectors of correct dimension
- [ ] SOL-AC-6: UpdateCommunity runs label propagation + LLM summarization
- [ ] SOL-AC-7: Bulkhead limits concurrent LLM requests to MAX_CONCURRENT
- [ ] SOL-AC-8: Same proto interface as graphiti-pipeline (swappable)
- [ ] SOL-AC-9: Unit test coverage ≥ 80%

### Trạng Thái Thực Thi

| ID | Task | Status | Assigned | Verify | Ghi chú |
|----|------|--------|----------|--------|---------|
| FEAT-KNW-001 | Domain layer | ⏳ Draft | AI | — | |
| FEAT-KNW-002 | Usecase layer | ⏳ Draft | AI | — | |
| FEAT-KNW-003 | gRPC handlers | ⏳ Draft | AI | — | |
| FEAT-KNW-004 | Bifrost LLM client | ⏳ Draft | AI | — | |
| FEAT-KNW-005 | Bifrost embedder | ⏳ Draft | AI | — | |
| FEAT-KNW-006 | Store client (read) | ⏳ Draft | AI | — | |
| FEAT-KNW-007 | Prompt registry | ⏳ Draft | AI | — | |
| FEAT-KNW-008 | Infrastructure | ⏳ Draft | AI | — | |
| QA-KNW-001 | Tests | ⏳ Draft | AI | — | |
