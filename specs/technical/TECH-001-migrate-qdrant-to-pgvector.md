---
id: TECH-001
title: Migrate Cognee embeddings from Qdrant to pgvector
service: cognee-pipeline
version: 1.0.0
status: Ready
priority: P2
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
risk_level: Medium
rollback_plan: Re-enable Qdrant config, revert VectorDB adapter selection to qdrant driver
---

## Mô Tả Thay Đổi

Migrate Cognee entity/chunk embeddings từ Qdrant sang pgvector (PostgreSQL extension). 4 engines khác (Memobase, Zep, Supermemory, OpenViking) đã dùng pgvector. Loại bỏ Qdrant giảm 1 infrastructure dependency.

## Lý Do

- Qdrant chỉ được dùng bởi Cognee → single point of dependency
- pgvector đã mature (HNSW indexing, IVFFlat) và được dùng bởi 4 engines khác
- Giảm ops burden: 1 fewer StatefulSet in Kubernetes, 1 fewer container in dev

## Các Bước Thực Hiện

1. Benchmark pgvector HNSW vs Qdrant cho Cognee entity embeddings (vector dimension, dataset size)
2. Create pgvector migration: `CREATE TABLE cognee_embeddings (id, collection, vector, metadata, ...)`
3. Add HNSW index: `CREATE INDEX ON cognee_embeddings USING hnsw (vector vector_cosine_ops)`
4. Update `cognee-pipeline` VectorDB adapter config: `vectordb.driver = "pgvector"` (from "qdrant")
5. Migrate existing embeddings from Qdrant → pgvector
6. Remove Qdrant from Docker Compose and Kubernetes manifests
7. Remove Qdrant client dependency from `go.mod`

## Risk & Mitigation

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| pgvector search latency higher than Qdrant | Medium | Medium | Benchmark before migration; tune HNSW params (m=16, ef_construction=200) |
| Migration data loss | Low | High | Dry-run migration on staging first; keep Qdrant running until verified |
| pgvector memory usage spike | Low | Medium | Monitor PostgreSQL memory; consider dedicated pgvector PostgreSQL instance |

## Rollback Plan

1. Re-enable Qdrant in Docker Compose / Kubernetes
2. Set `vectordb.driver = "qdrant"` in cognee-pipeline config
3. Qdrant data is preserved (read-only during migration)

## Verification Checklist

- [ ] Benchmark: pgvector latency ≤ 1.2× Qdrant latency for top-K search
- [ ] All cognee-search retrieval strategies return correct results
- [ ] Qdrant container removed from dev and production
- [ ] `go.mod` no longer has qdrant-go dependency
