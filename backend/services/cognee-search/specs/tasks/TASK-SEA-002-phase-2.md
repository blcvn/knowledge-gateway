---
id: TASK-SEA-002
title: Phase 2 - Adapter Layer Implementation
service: cognee-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-11
updated: 2026-05-11
linked_features: [FEAT-SEA-002]
---

# Kế Hoạch Triển Khai: Phase 2 - Adapter Layer

### Task 2.1: Implement Retriever Registry
- **File(s)**: `internal/adapter/retriever/registry.go`
- **Chi tiết**: Build map routing và method Get(Strategy) trả về đúng Retriever instance.

### Task 2.2: Implement Vector-based Retrievers
- **File(s)**: `internal/adapter/retriever/similarity.go`, `internal/adapter/retriever/chunks.go`, `internal/adapter/retriever/chunks_lexical.go`
- **Chi tiết**: SIMILARITY (Qdrant cosine), CHUNKS, CHUNKS_LEXICAL (BM25).

### Task 2.3: Implement Graph-based Retrievers
- **File(s)**: `internal/adapter/retriever/graph_completion.go`, `internal/adapter/retriever/natural_language.go`, `internal/adapter/retriever/summaries.go`, `internal/adapter/retriever/triplet_completion.go`, `internal/adapter/retriever/cypher.go`, `internal/adapter/retriever/temporal.go`
- **Chi tiết**: Graph traversal từ Neo4j (NL to Cypher, Cypher raw, community summary, triplet, temporal).

### Task 2.4: Implement Graph+LLM Retrievers
- **File(s)**: `internal/adapter/retriever/graph_cot.go`, `internal/adapter/retriever/graph_decomposition.go`, `internal/adapter/retriever/graph_context_ext.go`, `internal/adapter/retriever/graph_summary.go`
- **Chi tiết**: Sử dụng logic Chain-of-Thought, query decomposition, context extension, GraphSummary.

### Task 2.5: Implement Hybrid Retrievers
- **File(s)**: `internal/adapter/retriever/rag_completion.go`, `internal/adapter/retriever/feeling_lucky.go`
- **Chi tiết**: RAG_COMPLETION (Vector + LLM Answer), FEELING_LUCKY (heuristic chạy top 3 strategies).

### Task 2.6: Implement Repositories Adapter
- **File(s)**: `internal/adapter/repository/neo4j/graph_searcher.go`, `internal/adapter/repository/qdrant/vector_searcher.go`, `internal/adapter/repository/redis/cache_store.go`
- **Chi tiết**: Connect và query Neo4j, Qdrant, Redis. Implement TTL 5m cho query cache redis.

### Task 2.7: Implement LLM và Reranker Clients
- **File(s)**: `internal/adapter/client/llm_client.go`, `internal/adapter/client/reranker_client.go`
- **Chi tiết**: Call tới Bifrost LLM Client và Cross-encoder reranker.

### Task 2.8: Implement gRPC Handler & Mapper
- **File(s)**: `internal/adapter/grpc/handler.go`, `internal/adapter/grpc/mapper.go`
- **Chi tiết**: Implement interface gRPC `CogneeSearchServiceServer` và mapping DTO.

### Task 2.9: Implement NATS Subscriber
- **File(s)**: `internal/adapter/nats/subscriber.go`
- **Chi tiết**: Subscribe vào sự kiện `cognee.pipeline.completed` để xoá redis cache.

### Task 2.10: Unit & Integration Tests cho Phase 2
- **File(s)**: `internal/adapter/**/*_test.go`
- **Chi tiết**: Đảm bảo coverage >= 80%.
