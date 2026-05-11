# Technical Design Document: Zep Search Service

## 1. System Architecture

`zep-search` is an independently scalable search head that implements complex reranking and multi-modal retrieval.

```text
zep-search/
├── internal/
│   ├── domain/
│   │   └── search/      # Query intent, Search Result models
│   ├── usecase/
│   │   ├── retriever/   # Session & Graph retrievers
│   │   └── ranker/      # RRF, MMR, Temporal Decay logic
│   ├── adapter/
│   │   └── grpc/        # ZepSearchService endpoints
│   └── infra/
│       ├── postgres/    # pgvector queries
│       └── neo4j/       # Cypher traversals
```

## 2. Component Design

### 2.1 Domain Layer
- **Query**: Structured representation of search intent including context filters (`project_uuid`, `user_uuid`).
- **ScoredResult**: Agnostic result container with a normalized `float32` score.

### 2.2 Usecase Layer
- **VectorRetriever**: Executes `pgvector` similarity searches.
- **GraphRetriever**: Executes Cypher queries against Neo4j to find linked entities.
- **Search Query Builder**: Parses `GraphSearchQuery` which supports:
  - `scope`: `edges`, `nodes`, or `episodes`.
  - `filters`: `node_labels`, `edge_types` to improve precision and reduce latency.
  - `thresholds`: `min_fact_rating`, `limit`, `mmr_lambda`.
- **Reranker Pipeline**:
  - Implements Reciprocal Rank Fusion (`rrf`) to merge Graph and Vector results.
  - Implements Maximal Marginal Relevance (`mmr`) for diversity.
  - Implements `node_distance`, `episode_mentions`, and `cross_encoder` rerankers.
  - Applies Temporal Decay to downweight older facts.

### 2.3 Adapter Layer
- **gRPC Server**: Exposes synchronous search endpoints. Highly concurrent.

### 2.4 Infrastructure Layer
- **PostgreSQL**: Read-replicas optimized for vector search operations.
- **Neo4j**: Graph queries via Bolt protocol.
- **Observability**: Traces latency of each retrieval branch to isolate bottlenecks.
