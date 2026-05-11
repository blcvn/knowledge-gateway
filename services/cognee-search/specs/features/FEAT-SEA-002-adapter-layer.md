---
id: FEAT-SEA-002
title: Search Service — Adapter Layer (gRPC + 15 Retrievers + NATS)
service: cognee-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_feat: FEAT-SEA-001
---

## Mục Tiêu

Implement Layer 3 (Adapter) cho cognee-search — gRPC handlers, 15 retriever implementations, NATS subscriber for reindex, Neo4j graph querier, Qdrant vector searcher.

## Scope

### In Scope
- gRPC handler: `CogneeSearchServiceServer` (Search, RAGComplete, GetChunks)
- NATS subscriber: `cognee.pipeline.completed` → reindex cache
- 15 Retriever implementations (adapter-level, using port interfaces):
  - Vector-based: Similarity, Chunks, ChunksLexical
  - Graph-based: GraphCompletion, NaturalLanguage, Summaries, TripletCompletion, Cypher, Temporal
  - Graph+LLM: GraphCompletionCoT, GraphCompletionDecomposition, GraphCompletionContextExt, GraphSummaryCompletion
  - Hybrid: RAGCompletion, FeelingLucky
- Neo4j graph search adapter
- Qdrant vector search adapter
- Reranker adapter (cross-encoder via Bifrost)
- Redis cache adapter (query → result TTL cache)

### Out of Scope
- Domain/Usecase (FEAT-SEA-001)
- Config/Wire (FEAT-SEA-003)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
internal/adapter/
├── grpc/
│   ├── handler.go               # CogneeSearchServiceServer impl
│   └── mapper.go                # Proto ↔ Domain
├── nats/
│   └── subscriber.go            # cognee.pipeline.completed → cache invalidation
├── retriever/
│   ├── registry.go              # Strategy → Retriever routing
│   ├── similarity.go            # SIMILARITY: Qdrant cosine search
│   ├── chunks.go                # CHUNKS: Raw chunk retrieval from Qdrant
│   ├── chunks_lexical.go        # CHUNKS_LEXICAL: BM25 keyword search
│   ├── graph_completion.go      # GRAPH_COMPLETION: Graph + LLM synthesis
│   ├── natural_language.go      # NATURAL_LANGUAGE: NL → Cypher → results
│   ├── summaries.go             # SUMMARIES: Community summary search
│   ├── triplet_completion.go    # TRIPLET_COMPLETION: Triplet + LLM
│   ├── graph_cot.go             # CoT: Chain-of-thought graph traversal
│   ├── graph_decomposition.go   # Query decomposition + sub-graph search
│   ├── graph_context_ext.go     # Context extension via graph neighbors
│   ├── graph_summary.go         # Summary + LLM completion
│   ├── rag_completion.go        # RAG: Vector + LLM answer
│   ├── cypher.go                # Direct Cypher execution
│   ├── temporal.go              # Time-based filtering
│   └── feeling_lucky.go         # Auto-select best strategies
├── repository/
│   ├── neo4j/
│   │   └── graph_searcher.go    # Neo4j Cypher search adapter
│   ├── qdrant/
│   │   └── vector_searcher.go   # Qdrant vector search adapter
│   └── redis/
│       └── cache_store.go       # Query result cache (TTL 5m)
├── client/
│   ├── llm_client.go            # Bifrost LLM client
│   └── reranker_client.go       # Cross-encoder reranker
```

### gRPC Service

```protobuf
service CogneeSearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}

message SearchRequest {
  string query = 1;
  repeated string strategies = 2;    // SIMILARITY, GRAPH_COMPLETION, etc.
  int32 top_k = 3;                   // default: 10
  bool rerank = 4;                   // apply cross-encoder reranking
  SearchFilters filters = 5;         // dataset_id, time_range, entity_types
}
```

### Retriever Implementation Example

```go
// similarity.go
type SimilarityRetriever struct {
    vectorSearcher port.VectorSearcher
}

func (r *SimilarityRetriever) Retrieve(ctx context.Context, query string, topK int, filters domain.SearchFilters) ([]domain.SearchResult, error) {
    embedding, err := r.embedder.Embed(ctx, query)
    results, err := r.vectorSearcher.SearchSimilar(ctx, embedding, topK, filters.TenantID, filters.DatasetID)
    return mapToSearchResults(results, domain.Similarity), nil
}

func (r *SimilarityRetriever) Strategy() domain.SearchStrategy { return domain.Similarity }
func (r *SimilarityRetriever) RequiresLLM() bool { return false }
```

## Acceptance Criteria

- [ ] AC-1: Given gRPC Search request with `["SIMILARITY"]`, When processed, Then Qdrant vector search returns top-K results
- [ ] AC-2: Given `["GRAPH_COMPLETION"]`, When processed, Then Neo4j graph traversal + LLM synthesis returns enriched results
- [ ] AC-3: Given `["NATURAL_LANGUAGE"]`, When processed, Then NL query → Cypher translation → execution → results
- [ ] AC-4: Given `["FEELING_LUCKY"]`, When processed, Then auto-selects and runs top 3 strategies
- [ ] AC-5: Given RAGComplete request, When processed, Then search results fed to LLM → synthesized answer returned
- [ ] AC-6: Given `cognee.pipeline.completed` NATS event, When received, Then affected cache entries are invalidated
- [ ] AC-7: All 15 retrievers are registered in RetrieverRegistry

## Test Requirements

- **Unit tests**: Each retriever with mock ports
- **Integration tests**: gRPC → handler → usecase → mock retrievers
- **Coverage**: ≥ 80%
