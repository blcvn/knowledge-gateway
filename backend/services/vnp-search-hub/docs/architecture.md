---
id: DOC-S03
service: vnp-search-hub
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-search-hub — Service Architecture

> **Group**: Platform | **Pattern**: 4-layer Clean Architecture | **Orchestrator (Fan-Out)**

## Layer Structure

```
services/vnp-search-hub/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # RecallQuery, UnifiedResult, EngineResult
│   │   ├── value_object.go        # EngineName, RerankerStrategy, MergePolicy
│   │   ├── event.go               # RecallExecuted
│   │   └── errors.go              # ErrAllEnginesFailed, ErrTimeout
│   ├── usecase/
│   │   ├── recall.go              # Cross-engine recall orchestration
│   │   ├── search_engine.go       # Single-engine search
│   │   ├── merge.go               # Multi-engine result merging
│   │   ├── rerank.go              # Cross-engine reranking (RRF/Cross-Encoder)
│   │   └── port/
│   │       ├── input.go           # RecallUseCase, EngineSearchUseCase
│   │       └── output.go          # EngineSearchClient (interface per engine), CacheClient
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC server: Recall, SearchEngine
│   │   │   └── mapper.go
│   │   ├── client/
│   │   │   ├── cognee_client.go   # cognee-search gRPC client
│   │   │   ├── graphiti_client.go # graphiti-search gRPC client
│   │   │   ├── memobase_client.go # memobase-context gRPC client
│   │   │   ├── ov_client.go       # ov-search gRPC client
│   │   │   ├── zep_client.go      # zep-search gRPC client
│   │   │   ├── sm_client.go       # sm-search gRPC client
│   │   │   └── event_client.go    # vnp-event gRPC client
│   │   └── cache/
│   │       └── redis_cache.go     # Result caching
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

## Design Decisions

- **errgroup fan-out**: All engine searches run concurrently via `errgroup.Group` with 10s timeout per engine
- **Partial results**: If some engines fail, remaining results are still returned (degraded mode)
- **Circuit breaker per engine**: Prevents single engine failure from degrading overall latency
- **Pluggable reranking**: RRF (default, fast) or Cross-Encoder (high quality, higher latency)
- **Result normalization**: Each engine's result format is normalized to `UnifiedResult` before merging
