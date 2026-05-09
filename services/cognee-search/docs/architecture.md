---
id: DOC-S03
service: cognee-search
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-search — Service Architecture

> **Group**: Cognee | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/cognee-search/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # SearchQuery, SearchResult, SearchStrategy
│   │   ├── value_object.go        # SearchType enum (15 types), ScoreType
│   │   └── errors.go              # NoDataError, InvalidStrategyError
│   ├── usecase/
│   │   ├── search.go              # Main search dispatcher
│   │   ├── retriever/             # Strategy implementations
│   │   │   ├── similarity.go      #   Vector cosine similarity
│   │   │   ├── graph_completion.go #  Graph + LLM reasoning
│   │   │   ├── rag_completion.go  #   Traditional RAG
│   │   │   ├── chunks.go          #   Raw chunk retrieval
│   │   │   ├── lexical.go         #   BM25/Jaccard lexical search
│   │   │   ├── summaries.go       #   Hierarchical summaries
│   │   │   ├── natural_language.go #  NL → Cypher
│   │   │   ├── temporal.go        #   Temporal-aware search
│   │   │   ├── cypher.go          #   Direct Cypher execution
│   │   │   └── feeling_lucky.go   #   Auto-strategy selection
│   │   └── port/
│   │       ├── input.go           # SearchUseCase interface
│   │       └── output.go          # VectorRepo, GraphRepo, LLMClient, CacheRepo
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   └── nats/subscriber.go     # cognee.pipeline.completed listener
│   └── infra/
│       ├── persistence/
│       │   ├── qdrant/            # Vector search adapter
│       │   ├── neo4j/             # Graph traversal adapter
│       │   └── redis/             # Result cache adapter
│       ├── llm/                   # Bifrost LLM client
│       └── wire/wire.go
```

## Retriever Pattern

```mermaid
graph LR
    REQ["SearchRequest"] --> DISPATCH["Search Dispatcher"]
    DISPATCH --> |SIMILARITY| SIM["Similarity Retriever"]
    DISPATCH --> |GRAPH_COMPLETION| GC["Graph Completion Retriever"]
    DISPATCH --> |RAG_COMPLETION| RAG["RAG Retriever"]
    DISPATCH --> |CHUNKS| CHK["Chunks Retriever"]
    DISPATCH --> |FEELING_LUCKY| FL["Auto-Select Retriever"]
    SIM --> QDRANT["Qdrant"]
    GC --> NEO4J["Neo4j"]
    GC --> LLM["Bifrost LLM"]
    RAG --> QDRANT
    RAG --> LLM
```

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| Qdrant | Vector similarity search |
| Neo4j | Graph traversal queries |
| Redis | Search result cache (TTL 5min) |
| Bifrost | LLM completion for graph/RAG strategies |

## Design Decisions

- **Strategy pattern**: Each search type is a separate retriever implementing a common interface
- **Result caching**: Redis caches results with 5min TTL for identical queries
- **Parallel retrieval**: FEELING_LUCKY runs multiple strategies in parallel, returns best
