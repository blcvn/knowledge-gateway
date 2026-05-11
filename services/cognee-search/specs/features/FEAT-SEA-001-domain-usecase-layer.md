---
id: FEAT-SEA-001
title: Search Service — Domain + Usecase Layer (15 Retrieval Strategies)
service: cognee-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement Layer 1 (Domain) và Layer 2 (Usecase) cho cognee-search — 15 retrieval strategies, 3-phase search pipeline (retrieve → merge → rerank), RAG completion, và graph exploration.

## Bối Cảnh Nghiệp Vụ

cognee-search là multi-strategy retrieval engine. Mỗi search request chỉ định 1+ strategies, results được merge (RRF) và reranked. Also supports RAG completion (search + LLM answer) và graph exploration.

## Scope

### In Scope
- Domain entities: `SearchResult`, `RetrieverConfig`, `RerankScore`, `SearchSession`
- Domain value objects: `SearchStrategy` (15 types), `ResultType`, `SearchScope`
- Domain errors: `StrategyNotFoundError`, `EmptyQueryError`
- Usecase: `SearchUseCase` (3-phase orchestrator), `RAGCompleteUseCase`, `ExploreGraphUseCase`
- Port interfaces: `VectorSearcher`, `GraphSearcher`, `Reranker`, `LLMClient`, `CacheStore`
- Strategy pattern: `Retriever` interface + 15 implementations (usecase-level contracts)

### Out of Scope
- gRPC handlers, retriever implementations (FEAT-SEA-002)
- Config, Wire, server (FEAT-SEA-003)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
internal/
├── domain/
│   ├── entity.go           # SearchResult, SearchSession
│   ├── value_object.go     # SearchStrategy (15 types), ResultType, SearchScope
│   ├── event.go            # (none — search is stateless)
│   └── errors.go           # StrategyNotFound, EmptyQuery
├── usecase/
│   ├── search.go           # 3-phase search orchestrator
│   ├── rag_complete.go     # Search + LLM answer generation
│   ├── explore_graph.go    # Graph neighborhood exploration
│   ├── merge.go            # RRF (Reciprocal Rank Fusion) merge logic
│   ├── port/
│   │   ├── input.go        # SearchUseCase, RAGCompleteUseCase, ExploreGraphUseCase
│   │   └── output.go       # Retriever, VectorSearcher, GraphSearcher, Reranker, LLMClient
│   └── dto/
│       ├── request.go      # SearchRequest, RAGRequest, ExploreRequest
│       └── response.go     # SearchResponse, RAGResponse, ExploreResponse
```

### 15 Search Strategies

```go
type SearchStrategy string

const (
    Similarity                    SearchStrategy = "SIMILARITY"               // Qdrant cosine
    GraphCompletion               SearchStrategy = "GRAPH_COMPLETION"         // Neo4j + LLM
    RAGCompletion                 SearchStrategy = "RAG_COMPLETION"           // Qdrant + LLM
    NaturalLanguage               SearchStrategy = "NATURAL_LANGUAGE"         // NL → Cypher
    Chunks                        SearchStrategy = "CHUNKS"                   // Raw chunk retrieve
    ChunksLexical                 SearchStrategy = "CHUNKS_LEXICAL"           // BM25 keyword
    Summaries                     SearchStrategy = "SUMMARIES"                // Community summaries
    TripletCompletion             SearchStrategy = "TRIPLET_COMPLETION"       // Neo4j triplet + LLM
    GraphCompletionCoT            SearchStrategy = "GRAPH_COMPLETION_COT"     // Chain-of-thought
    GraphCompletionDecomposition  SearchStrategy = "GRAPH_COMPLETION_DECOMPOSITION"
    GraphCompletionContextExt     SearchStrategy = "GRAPH_COMPLETION_CONTEXT_EXTENSION"
    GraphSummaryCompletion        SearchStrategy = "GRAPH_SUMMARY_COMPLETION"
    Cypher                        SearchStrategy = "CYPHER"                   // Raw Cypher
    Temporal                      SearchStrategy = "TEMPORAL"                 // Time-based
    FeelingLucky                  SearchStrategy = "FEELING_LUCKY"            // Auto-select best
)
```

### 3-Phase Search Pipeline

```go
func (uc *SearchUseCase) Execute(ctx context.Context, req dto.SearchRequest) (*dto.SearchResponse, error) {
    // Phase 1: RETRIEVE — run selected strategies concurrently
    var allResults []domain.SearchResult
    g, ctx := errgroup.WithContext(ctx)
    for _, strategy := range req.Strategies {
        retriever := uc.registry.Get(strategy)
        g.Go(func() error {
            results, err := retriever.Retrieve(ctx, req.Query, req.TopK, req.Filters)
            mu.Lock(); allResults = append(allResults, results...); mu.Unlock()
            return err
        })
    }
    g.Wait()
    
    // Phase 2: MERGE — RRF deduplication + scoring
    merged := uc.merge(allResults, req.TopK*3)
    
    // Phase 3: RERANK — cross-encoder (optional)
    if req.Rerank {
        merged = uc.reranker.Rerank(ctx, req.Query, merged, req.TopK)
    }
    
    return &dto.SearchResponse{Results: merged[:min(req.TopK, len(merged))]}, nil
}
```

### Retriever Interface (Port)

```go
// port/output.go
type Retriever interface {
    Retrieve(ctx context.Context, query string, topK int, filters SearchFilters) ([]domain.SearchResult, error)
    Strategy() domain.SearchStrategy
    RequiresLLM() bool
}
```

### RRF Merge

```go
// Reciprocal Rank Fusion: score = Σ(1 / (k + rank_i)) across all strategies
func (uc *SearchUseCase) merge(results []domain.SearchResult, topK int) []domain.SearchResult {
    scores := map[string]float64{}  // result_id → fused_score
    k := 60.0 // RRF constant
    // Group by strategy, assign ranks, compute fused scores
    // Deduplicate by result content hash
    // Sort by fused score desc
}
```

## Acceptance Criteria

- [ ] AC-1: Given a SearchRequest with 3 strategies, When orchestrator runs, Then all 3 retrievers execute concurrently
- [ ] AC-2: Given duplicate results from different strategies, When merge runs, Then duplicates are fused with RRF scoring
- [ ] AC-3: Given rerank=true, When results are merged, Then reranker is called and results re-ordered
- [ ] AC-4: Given FEELING_LUCKY strategy, When retriever selection runs, Then auto-select top 3 strategies based on query type
- [ ] AC-5: Given all 15 strategies defined, When registry is queried, Then each returns correct Retriever
- [ ] AC-6: Port interfaces have no infrastructure dependencies
- [ ] AC-7: Given RAGCompleteRequest, When usecase runs, Then search results are fed to LLM for answer generation

## Test Requirements

- **Unit tests**: Orchestrator with mock retrievers, merge logic, RRF scoring
- **Domain tests**: SearchResult ranking, strategy validation
- **Coverage**: ≥ 80%
