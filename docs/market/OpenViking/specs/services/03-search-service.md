# 03 — OpenViking Search Service

> **Service**: `openviking-search`  
> **Port**: 9012 (gRPC) · 9092 (Health/Metrics)  
> **Origin**: L2 SearchService + L4 HierarchicalRetriever + L5 VikingDBManager  
> **Role**: Semantic search, hierarchical retrieval, reranking, hotness scoring, vector index management

---

## 1. Responsibilities

| Capability | Description |
|-----------|-------------|
| **Semantic Search** | `find()` — stateless semantic search via HierarchicalRetriever |
| **Context-Aware Search** | `search()` — session-aware retrieval with history context |
| **Hierarchical Retrieval** | 6-step recursive directory search with score propagation |
| **Vector Index** | Manage VikingDB collections, upsert/delete context vectors |
| **Hotness Scoring** | Blend semantic relevance with usage recency (active_count) |
| **Reranking** | Optional cross-encoder reranking via Bifrost |
| **Convergence Detection** | Stop traversal after 3 stable rounds |
| **Debug/Replay** | IO recording and search replay for diagnostics |

---

## 2. Clean Architecture Layout

```
services/openviking-search/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── query.go                    # TypedQuery, QueryResult, MatchedContext
│   │   ├── retriever_config.go         # RetrieverConfig, convergence params
│   │   ├── hotness.go                  # HotnessScore, decay function
│   │   ├── observer.go                 # RetrievalStats, SearchReplay
│   │   └── errors.go
│   ├── usecase/
│   │   ├── find.go                     # Stateless semantic search
│   │   ├── search.go                   # Session-aware search
│   │   ├── hierarchical_retrieve.go    # 6-step retrieval algorithm (core)
│   │   ├── index_content.go            # Embed + upsert to vector index
│   │   ├── remove_content.go           # Remove from vector index
│   │   ├── update_hotness.go           # Increment active_count
│   │   ├── replay_search.go            # Debug replay
│   │   ├── port/
│   │   │   ├── input.go               # SearchUseCase, IndexUseCase interfaces
│   │   │   └── output.go             # VectorStore, EmbedderClient, RerankerClient
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── vectordb/              # VikingDB/Qdrant adapter
│   │   │   │   ├── collection.go      #   Collection management
│   │   │   │   ├── search.go          #   Vector search operations
│   │   │   │   └── upsert.go          #   Upsert/delete operations
│   │   │   └── redis/                 # Search result cache
│   │   ├── client/
│   │   │   ├── embedder_client.go     # Bifrost embedding adapter
│   │   │   ├── reranker_client.go     # Bifrost reranking adapter
│   │   │   └── fs_client.go           # Read context from FS service
│   │   └── event/
│   │       ├── subscriber.go          # NATS: ov.content.written/deleted
│   │       └── publisher.go           # NATS: ov.search.indexed
│   └── infra/
```

---

## 3. HierarchicalRetriever — 6-Step Algorithm

```
Input: query, context_type?, target_directories?, limit, threshold

Step 1: Determine Starting Directories
  ├── From target_directories (explicit scope)
  └── From context_type → GetRootURIs():
         MEMORY  → [user/memories, agent/memories]
         RESOURCE → [viking://resources]
         SKILL   → [agent/skills]
         None    → all of the above

Step 2: Embed Query
  └── EmbedderClient.Embed(query, isQuery=true)
      → dense_vector (float32, 768-3072 dim)
      → sparse_vector (BM25/SPLADE-style)

Step 3: Global Vector Search
  └── VectorStore.SearchGlobalRoots(dense, sparse, accountID, topK=10)
      → top-K across all L0/L1 nodes in tenant

Step 4: Merge Starting Points
  ├── Root URIs from Step 1
  ├── Global hits from Step 3 (L0/L1 only, exclude L2)
  ├── Optional rerank on merged set
  └── Initialize priority queue (max-heap by score)

Step 5: Recursive Directory Search
  ┌─────────────────────────────────────────────┐
  │ while queue not empty:                       │
  │   dir = pop highest-score directory          │
  │   children = VectorStore.SearchChildren(     │
  │     dir.URI, dense, sparse, accountID)       │
  │   for child in children:                     │
  │     if child.Level == 2:                     │
  │       optional rerank                        │
  │       score = α·child + (1-α)·dir_score      │
  │       add to candidates (dedup by URI)       │
  │     elif child is directory (L0/L1):         │
  │       push to priority_queue                 │
  │   convergence: if top-K stable 3 rounds→STOP │
  └─────────────────────────────────────────────┘

Step 6: Post-Processing
  ├── Hotness blending: final = (1-α_hot)·semantic + α_hot·hotness
  ├── Sort by final score descending
  ├── Apply limit and threshold
  └── Fetch related contexts (if relations exist)

Output: QueryResult{MatchedContexts[], SearchedDirectories[]}
```

---

## 4. gRPC Service Definition

```protobuf
service SearchService {
  // Search operations
  rpc Find(FindRequest) returns (FindResponse);
  rpc Search(SearchRequest) returns (SearchResponse);

  // Index management (event-driven, also available via gRPC)
  rpc IndexContent(IndexContentRequest) returns (IndexContentResponse);
  rpc RemoveContent(RemoveContentRequest) returns (RemoveContentResponse);
  rpc UpdateHotness(UpdateHotnessRequest) returns (UpdateHotnessResponse);

  // Collection management
  rpc CreateCollection(CreateCollectionRequest) returns (CreateCollectionResponse);
  rpc CollectionExists(CollectionExistsRequest) returns (CollectionExistsResponse);

  // Debug
  rpc GetRetrievalStats(GetRetrievalStatsRequest) returns (GetRetrievalStatsResponse);
  rpc ReplaySearch(ReplaySearchRequest) returns (ReplaySearchResponse);
}
```

---

## 5. Vector Schema

```go
type ContextVector struct {
    URI            string            // Primary key
    ParentURI      string            // Parent directory
    ContextType    int32             // MEMORY=0, RESOURCE=1, SKILL=2, SESSION=3
    Level          int32             // 0=Abstract, 1=Overview, 2=Detail
    OwnerAccountID string
    OwnerUserID    string
    Abstract       string            // L0 text (~100 tokens)
    ActiveCount    int64             // Usage counter (hotness)
    DenseVector    []float32         // 768-3072 dimensions
    SparseVector   map[string]float32 // BM25/SPLADE
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

---

## 6. Key Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `GLOBAL_SEARCH_TOPK` | 10 | Global candidate count |
| `MAX_CONVERGENCE_ROUNDS` | 3 | Stop after N stable rounds |
| `DIRECTORY_DOMINANCE_RATIO` | 1.2 | Dir must exceed max child score |
| `score_propagation_alpha` | 0.7 | Parent→child blend weight |
| `hotness_alpha` | 0.1 | Semantic vs recency weight |

---

## 7. Event-Driven Index Sync

```
NATS Subscribe: ov.content.written
  → Parse event → Embed content → Upsert to vector index

NATS Subscribe: ov.content.deleted
  → Parse event → Delete from vector index

NATS Subscribe: ov.session.committed
  → Parse event → Update active_count for used URIs
```
