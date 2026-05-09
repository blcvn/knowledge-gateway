---
id: TDD-ov-search
title: Technical Design — ov-search
service: ov-search
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: OpenViking
---

# Technical Design — ov-search

> **Group**: OpenViking | **gRPC Port**: 9052 | **Origin**: OpenViking (HierarchicalRetriever)

## 1. Service Overview

Hierarchical retrieval service with score propagation, hotness scoring, convergence detection, and tiered loading. Consumed by `vnp-search-hub` for cross-engine recall.

**Origin mapping**: `openviking/retrieve/hierarchical_retriever.py` + `openviking/retrieve/intent_analyzer.py` + `openviking/core/hotness.py`.

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── search_result.go        # SearchResult, Score, MatchedContext
│   ├── hotness.go              # HotnessScore, DecayConfig
│   ├── embedding.go            # EmbeddingVector, UpsertPayload
│   └── context_type.go         # ContextType, QueryPlan, TypedQuery
├── repository/
│   ├── vector_repo.go          # VectorRepository (upsert/search/delete)
│   └── hotness_repo.go         # HotnessRepository
├── event.go                    # HotnessUpdated event
└── errors.go                   # IndexNotFound, EmbeddingFailed
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── hierarchical_search.go      # 7-step search pipeline
├── context_retrieval.go        # RetrieveContext with tiered loading
├── hotness.go                  # Hotness scoring + exponential decay
├── embedding_ops.go            # UpsertEmbedding, DeleteEmbedding
├── port/
│   ├── input.go               # SearchUseCase, EmbeddingUseCase
│   └── output.go              # EmbedderPort, FileReaderPort, RerankPort
└── dto/
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/handler.go             # OvSearchService gRPC
├── event/subscriber.go         # ov.content.written/deleted + ov.session.committed
└── client/
    ├── bifrost_client.go       # Embedding generation
    └── fs_client.go            # ov-fs gRPC (tiered loading L0→L1→L2)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/
│   ├── qdrant_repo.go          # Qdrant vector search
│   ├── pgvector_repo.go        # pgvector fallback
│   └── hotness_repo.go         # PostgreSQL hotness scores
├── config/config.go
└── wire/wire.go
```

## 3. gRPC API

```protobuf
service OvSearchService {
  rpc HierarchicalSearch(SearchRequest) returns (SearchResponse);
  rpc RetrieveContext(ContextRequest) returns (ContextResponse);
  rpc GetHotness(HotnessRequest) returns (HotnessResponse);
  rpc UpsertEmbedding(UpsertRequest) returns (google.protobuf.Empty);
  rpc DeleteEmbedding(DeleteRequest) returns (google.protobuf.Empty);
}
```

## 4. NATS Events

### Subscribed

| Subject | Action |
|---------|--------|
| `ov.content.written` | Generate embedding → upsert to vector DB |
| `ov.content.deleted` | Remove embedding from vector DB |
| `ov.resource.ingested` | Index newly parsed resource chunks |
| `ov.session.committed` | Update hotness scores for referenced files |

## 5. Data Model

- **Qdrant collection**: `ov_embeddings` (1536-dim dense + BM25 sparse)
- **PostgreSQL**: `ov_hotness_scores`, `ov_search_metadata`
- **Key algorithms**: Score propagation (child→parent), Hotness decay (`exp(-λt)`), Convergence detection

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Purpose |
|---------|-----------|----------|---------|
| Qdrant | Outbound | Native | Vector similarity search |
| Bifrost | Outbound | gRPC | Embedding generation |
| ov-fs | Outbound | gRPC | File content for tiered loading |
| ov-fs | Inbound (NATS) | Async | Content change notifications |
| ov-session | Inbound (NATS) | Async | Session commit → hotness boost |
| ov-resource | Inbound (NATS) | Async | Resource ingested → indexing |

## 7. Observability

- **Metrics**: Query count/latency, embedding upsert count, hotness recompute duration
- **Traces**: OTel spans: `ov-search.HierarchicalSearch` (with sub-spans per pipeline step)
- **Logs**: Structured JSON with `request_id`, `account_id`, `query_hash`
- **Health**: gRPC Health v1 + HTTP `/healthz` on port 9105

## 8. Multi-Tenancy

- **Isolation Key**: `account_id` — Qdrant payload filter + PostgreSQL WHERE clause
- **Vector Isolation**: Payload index on `account_id` for filtered search

---

> **Next Steps**: Decompose into FEAT specs: FEAT-001 (Hierarchical Search Pipeline), FEAT-002 (Hotness Scoring), FEAT-003 (Score Propagation), FEAT-004 (Embedding Management).
