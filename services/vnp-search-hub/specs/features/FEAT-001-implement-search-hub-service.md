---
id: FEAT-001
title: Implement vnp-search-hub — Cross-Engine Search Orchestrator
service: vnp-search-hub
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-002
linked_tdd: TDD-vnp-search-hub
---

## Mục Tiêu

Implement vnp-search-hub as a stateless search orchestrator that fan-outs queries to 7 engine search services in parallel, merges + deduplicates + reranks results, and returns a unified context window. This is the backend for `memory.recall()`.

## Bối Cảnh Nghiệp Vụ

vnp-search-hub is the "brain" of the memory platform — when an AI agent calls `memory.recall()`, the gateway routes to search-hub which queries all engines in parallel, applies reranking (RRF/MMR/CrossEncoder), and assembles a token-budgeted context window.

## Scope

### In Scope
- Domain: RecallRequest, RecallResponse, RerankStrategy (enum), EngineResult
- Usecase: RecallService (parallel fan-out, merge, rerank), MultiSearchService
- gRPC handlers: VnpSearchHubService (Recall, MultiSearch)
- Engine client registry: gRPC clients for 7 search services
- Reranking implementations: RRF (default), MMR (diversity), CrossEncoder (quality)
- Token budgeting: ensure context fits within token limit
- Circuit breaker per engine (graceful degradation)
- go.mod, cmd/server/main.go, config

### Out of Scope
- Individual engine search logic (each engine has its own search service)
- Embedding generation (done by engines or shared service)

## Thiết Kế Kỹ Thuật

### API Contract (from tdd.md §3)
```protobuf
service VnpSearchHubService {
  rpc Recall(RecallRequest) returns (RecallResponse);
  rpc MultiSearch(MultiSearchRequest) returns (MultiSearchResponse);
}
```

### Recall Pipeline (from tdd.md §4)
```
RecallRequest → validate + parse scope
  │
  ├→ cognee-search.Search()    ─┐
  ├→ graphiti-search.Search()  ─┤
  ├→ memobase-context.Query()  ─┤  parallel fan-out
  ├→ ov-search.Search()        ─┤  with circuit breaker
  ├→ zep-search.Search()       ─┤  per engine
  ├→ sm-search.Search()        ─┤
  └→ vnp-event.SearchEvents()  ─┘
                                │
                        merge + dedup
                                │
                        rerank (RRF/MMR)
                                │
                        token budgeting
                                │
                        RecallResponse
```

### Reranking Strategies
- **RRF** (Reciprocal Rank Fusion): `score = Σ 1/(k + rank)` — fast, default
- **MMR** (Maximal Marginal Relevance): balances relevance + diversity
- **CrossEncoder**: re-scores via LLM — highest quality, highest latency

### Internal Architecture
```
services/vnp-search-hub/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   ├── recall.go       # RecallRequest, RecallResponse
│   │   │   ├── rerank.go       # RerankStrategy enum, RRFConfig
│   │   │   └── engine.go       # EngineResult, EngineConfig
│   │   └── errors.go           # AllEnginesFailed, TokenBudgetExceeded
│   ├── usecase/
│   │   ├── recall_service.go   # Fan-out, merge, rerank, budget
│   │   ├── reranker/
│   │   │   ├── rrf.go          # Reciprocal Rank Fusion
│   │   │   ├── mmr.go          # Maximal Marginal Relevance
│   │   │   └── cross_encoder.go
│   │   └── token_budgeter.go   # Token-aware result truncation
│   ├── adapter/
│   │   ├── grpc/handler.go     # VnpSearchHubService handler
│   │   └── client/
│   │       └── engine_client.go # gRPC clients to 7 search engines
│   └── infra/
│       ├── config/config.go
│       └── circuit/breaker.go  # Per-engine circuit breaker
└── go.mod
```

## Acceptance Criteria

- [ ] AC-1: `go build ./cmd/server/` compiles without errors
- [ ] AC-2: Recall with scope=all → fans out to all 7 engines in parallel
- [ ] AC-3: Recall with scope=semantic → fans out only to Cognee + Graphiti + Zep
- [ ] AC-4: Engine failure → circuit breaker opens → partial results returned (not error)
- [ ] AC-5: RRF reranking produces correct scores: `Σ 1/(60 + rank_i)`
- [ ] AC-6: Token budgeting truncates results to fit within token_budget
- [ ] AC-7: p95 latency < 500ms when all engines respond < 200ms

## Test Requirements
- **Unit tests:** RRF/MMR reranker, TokenBudgeter, RecallService with mock engine clients
- **Integration tests:** Recall round-trip with mock engine gRPC servers
- **Minimum coverage:** 80%

## Definition of Done
- [ ] Code implements all Acceptance Criteria
- [ ] Unit tests pass, coverage ≥ 80%
- [ ] `docs/changelog.md` updated
- [ ] No lint errors
