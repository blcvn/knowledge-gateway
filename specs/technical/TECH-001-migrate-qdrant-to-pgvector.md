---
id: TECH-001
title: Unified Vector Store Interface (Qdrant + pgvector)
service: pkg/vectorstore
version: 2.0.0
status: In Progress
priority: P2
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
risk_level: Low
rollback_plan: N/A — additive change, both backends remain operational
---

## Mô Tả Thay Đổi (Revised)

~~Migrate Cognee embeddings from Qdrant to pgvector.~~

**Updated**: Maintain both Qdrant and pgvector as dual vector backends. Create a unified `VectorStore` interface in `pkg/vectorstore` that abstracts the backend choice. Each engine configures which backend to use via config.

## Lý Do

- Qdrant provides specialized ANN performance for high-throughput workloads
- pgvector integrates with existing PostgreSQL — zero additional infra for engines that prefer it
- Unified interface enables per-engine backend selection without code changes

## Kiến Trúc

```
pkg/vectorstore/
├── vectorstore.go        # VectorStore interface
├── pgvector/
│   └── adapter.go        # pgvector implementation
├── qdrant/
│   └── adapter.go        # Qdrant implementation
└── vectorstore_test.go   # Interface compliance tests
```

### Backend Assignment

| Engine | Vector Backend | Reason |
|--------|---------------|--------|
| Cognee | Qdrant | High-volume entity embeddings, ANN performance |
| Graphiti | pgvector | Entity/edge embeddings, co-located with graph metadata |
| Memobase | pgvector | Profile embeddings, low volume |
| OpenViking | pgvector | Resource embeddings, co-located with FS metadata |
| Zep | pgvector | Fact embeddings, co-located with messages |
| Supermemory | pgvector | Memory embeddings, co-located with documents |

## Verification Checklist

- [ ] `VectorStore` interface covers: Upsert, Search, Delete, CollectionExists
- [ ] pgvector adapter passes interface compliance tests
- [ ] Qdrant adapter passes interface compliance tests
- [ ] Each engine can be configured to use either backend
- [ ] Both backends operational in docker-compose
