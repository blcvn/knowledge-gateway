# 04 — Cognee Search Service

> **gRPC**: 9013 | **Health**: 9093

---

## 1. Purpose

Multi-strategy retrieval engine: 15+ retriever strategies, hybrid search (vector + graph + keyword), reranking, và graph exploration.

---

## 2. Clean Architecture

```
services/cognee-search/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # SearchResult, RetrieverConfig, RerankScore
│   │   ├── value_object.go     # SearchStrategy, SearchScope, ScoreType
│   │   └── errors.go
│   ├── usecase/
│   │   ├── search.go           # Main search orchestrator
│   │   ├── explore_graph.go    # Graph neighborhood exploration
│   │   ├── rerank.go           # Cross-encoder reranking
│   │   ├── port/
│   │   │   ├── input.go        # SearchUseCase, ExploreGraphUseCase
│   │   │   └── output.go       # VectorSearcher, GraphSearcher, Reranker
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   ├── retriever/          # Strategy implementations
│   │   │   ├── similarity.go          # Embedding similarity (Qdrant)
│   │   │   ├── graph_completion.go    # Graph neighborhood traversal
│   │   │   ├── graph_summary.go       # Community summary search
│   │   │   ├── insights.go            # Insight extraction
│   │   │   ├── keyword.go             # BM25 keyword search
│   │   │   ├── chunks.go              # Raw chunk retrieval
│   │   │   ├── entities.go            # Entity-focused search
│   │   │   ├── relationships.go       # Edge-focused search
│   │   │   ├── communities.go         # Community-focused search
│   │   │   ├── hybrid.go              # Vector + Graph merge
│   │   │   ├── contextual.go          # Context-aware retrieval
│   │   │   ├── temporal.go            # Time-based filtering
│   │   │   ├── multi_hop.go           # Multi-hop reasoning
│   │   │   ├── ontology.go            # Ontology-guided search
│   │   │   ├── cross_dataset.go       # Cross-dataset federation
│   │   │   └── registry.go            # Strategy name → Retriever
│   │   ├── reranker/
│   │   │   └── cross_encoder.go       # Uses pkg/adapters/reranker
│   │   ├── repository/
│   │   │   ├── qdrant/                # Vector search queries
│   │   │   └── neo4j/                 # Graph traversal queries
│   │   └── event/subscriber.go        # cognee.cognify.pipeline.completed → reindex
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
```

---

## 3. Search Pipeline (3-Phase)

```
SearchRequest(query, strategies[], top_k, filters)
         │
         ▼
┌── Phase 1: RETRIEVE ─────────────────────┐
│  For each strategy in request:            │
│    - Run retriever concurrently           │
│    - Collect scored results               │
│    - Apply per-strategy top_k             │
└────────────┬─────────────────────────────┘
             ▼
┌── Phase 2: MERGE ────────────────────────┐
│  - Deduplicate by result ID              │
│  - RRF (Reciprocal Rank Fusion) merge    │
│  - Apply global filters (tenant, dataset) │
└────────────┬─────────────────────────────┘
             ▼
┌── Phase 3: RERANK ───────────────────────┐
│  - Cross-encoder reranking (if enabled)  │
│  - Apply final top_k                     │
│  - Attach metadata + provenance          │
└────────────┬─────────────────────────────┘
             ▼
SearchResponse(results[], total, scores[])
```

---

## 4. Retriever Strategy Enum

```go
type SearchStrategy string

const (
    StrategySimilarity      SearchStrategy = "SIMILARITY"
    StrategyGraphCompletion SearchStrategy = "GRAPH_COMPLETION"
    StrategyGraphSummary    SearchStrategy = "GRAPH_SUMMARY"
    StrategyInsights        SearchStrategy = "INSIGHTS"
    StrategyKeyword         SearchStrategy = "KEYWORD"
    StrategyChunks          SearchStrategy = "CHUNKS"
    StrategyEntities        SearchStrategy = "ENTITIES"
    StrategyRelationships   SearchStrategy = "RELATIONSHIPS"
    StrategyCommunities     SearchStrategy = "COMMUNITIES"
    StrategyHybrid          SearchStrategy = "HYBRID"
    StrategyContextual      SearchStrategy = "CONTEXTUAL"
    StrategyTemporal        SearchStrategy = "TEMPORAL"
    StrategyMultiHop        SearchStrategy = "MULTI_HOP"
    StrategyOntology        SearchStrategy = "ONTOLOGY"
    StrategyCrossDataset    SearchStrategy = "CROSS_DATASET"
)
```

---

## 5. Domain Model

```go
type SearchResult struct {
    ID         uuid.UUID
    Type       ResultType  // CHUNK, ENTITY, EDGE, COMMUNITY
    Content    string
    Score      float64
    Strategy   SearchStrategy
    Metadata   map[string]any
    DatasetID  uuid.UUID
    Provenance []string    // Source trail
}
```
