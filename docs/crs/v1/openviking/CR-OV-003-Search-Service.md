# Change Request: CR-OV-003 — Search Service (Hierarchical Retrieval Engine)

**CR ID:** CR-OV-003  
**Component:** `services/openviking-search` [NEW SERVICE]  
**Priority:** High  
**Status:** Implemented
**Reference:** OpenViking PRD §4.2-4.3, SRS §2.2-2.3, specs/services/03-search-service.md  
**Maps from Python:** `retrieve/hierarchical_retriever.py`, `service/search_service.py`, `storage/vikingdb_manager.py`

---

## 1. Mô tả

Xây dựng **openviking-search** — Search Engine phân cấp với thuật toán 6 bước độc đáo, thay thế RAG phẳng truyền thống:

1. **HierarchicalRetriever (6 bước)**: Intent Analysis → Embed Query → Global Vector Search → Merge Starting Points → Recursive Directory Search → Post-Processing.
2. **Dual Vector Search**: Dense (float32, 768-3072 dims) + Sparse (BM25/SPLADE) hybrid.
3. **Score Propagation**: `final_score = α × child_score + (1-α) × parent_score` — lan truyền điểm từ thư mục cha xuống con.
4. **Hotness Blending**: `blended = (1-α_hot) × semantic + α_hot × hotness` — tăng điểm cho files được dùng nhiều.
5. **Convergence Detection**: Dừng sau 3 rounds mà top-K không đổi — tránh tìm kiếm vô hạn.
6. **Reranking**: Optional cross-encoder reranking (Volcengine, OpenAI, Cohere, Jina, local).
7. **Session-Aware Search**: Tìm kiếm có context của session hiện tại.
8. **Debug/Replay**: IO recording và search replay để tối ưu retrieval.
9. **Event-Driven Index Sync**: Subscribe NATS events từ FS service để tự động index.

---

## 2. Vấn đề hiện tại

- VNP Memory chỉ có flat vector search, thiếu hierarchical/recursive search theo cấu trúc thư mục.
- Chưa có score propagation từ parent → child directories.
- Thiếu hotness/recency boosting cho files được dùng nhiều.
- Không có convergence detection → có thể search vô hạn.
- Thiếu session-aware search (dùng ngữ cảnh session hiện tại để rerank).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/openviking-search/` (Port gRPC: 9012)

### 3.2. Domain Layer

```go
// domain/query.go
type TypedQuery struct {
    Query             string
    ContextType       *ContextType    // MEMORY | RESOURCE | SKILL | SESSION | nil = all
    TargetDirectories []string        // Optional: restrict to specific URIs
    Limit             int
    Threshold         float64
    RerankerEnabled   bool
}

type QueryResult struct {
    MatchedContexts    []MatchedContext
    SearchedDirectories []string
    LatencyMs          int64
}

type MatchedContext struct {
    URI           string
    ParentURI     string
    ContextType   ContextType
    Level         int
    Abstract      string        // L0 summary
    Score         float64       // Final blended score
    SemanticScore float64       // Raw semantic similarity
    HotnessScore  float64       // Recency/usage score
}

// domain/retriever_config.go
type RetrieverConfig struct {
    GlobalSearchTopK          int     // default: 10
    MaxConvergenceRounds      int     // default: 3 rounds before stopping
    DirectoryDominanceRatio   float64 // default: 1.2 — dir must exceed max child
    ScorePropagationAlpha     float64 // default: 0.7 (parent weight)
    HotnessAlpha              float64 // default: 0.1 (recency weight)
    Threshold                 float64 // minimum score to return
    RerankerProvider          string  // volcengine | openai | cohere | jina | local
}
```

### 3.3. HierarchicalRetriever — 6-Step Algorithm (Core)

```
Input: TypedQuery{query, context_type?, target_dirs?, limit, threshold}

──────────────────────────────────────────────────────
Step 1: Determine Starting Directories
  ├── From TargetDirectories (explicit scope by caller)
  └── From ContextType → GetRootURIs():
         MEMORY   → ["viking://user/*/memories/", "viking://agent/*/memories/"]
         RESOURCE → ["viking://resources/"]
         SKILL    → ["viking://agent/*/skills/"]
         nil      → all of the above (cross-type search)

Step 2: Embed Query
  └── Bifrost.Embed(query, isQuery=true)
      → dense_vector  (float32[], 768-3072 dims, model-dependent)
      → sparse_vector (map[string]float32 — BM25/SPLADE features)

Step 3: Global Vector Search
  └── VectorDB.SearchGlobalRoots(
        dense_vector, sparse_vector, account_id,
        topK=10, levelFilter=[L0,L1])   ← exclude L2 in global search
      → top-K candidates across ALL L0/L1 nodes in tenant

Step 4: Merge Starting Points
  ├── Root URIs from Step 1 (always included)
  ├── Global hits from Step 3 (L0/L1 parents only)
  ├── Optional rerank on merged set (cross-encoder)
  └── Initialize max-heap priority queue (keyed by score)

Step 5: Recursive Directory Search
  ┌────────────────────────────────────────────────────────────────┐
  │ convergence_counter = 0                                         │
  │ top_k_snapshot = []                                             │
  │                                                                 │
  │ while queue not empty AND convergence_counter < 3:             │
  │   dir = queue.pop()  ← highest-score directory                │
  │   children = VectorDB.SearchChildren(                          │
  │     parent_uri=dir.URI,                                        │
  │     dense_vector, sparse_vector, account_id)                  │
  │                                                                 │
  │   for child in children:                                       │
  │     if child.Level == 2:           ← L2 = leaf file           │
  │       if reranker_enabled:                                     │
  │         child.score = Reranker.Score(query, child.Abstract)   │
  │       child.final_score = α·child.score + (1-α)·dir.score     │
  │       add_to_candidates(child)   ← dedup by URI               │
  │     elif child.is_directory:      ← L0/L1 = subdirectory      │
  │       queue.push(child)                                        │
  │                                                                 │
  │   new_top_k = candidates.top(limit)                            │
  │   if new_top_k == top_k_snapshot:                              │
  │     convergence_counter++                                      │
  │   else:                                                         │
  │     convergence_counter = 0; top_k_snapshot = new_top_k        │
  └────────────────────────────────────────────────────────────────┘

Step 6: Post-Processing
  ├── Hotness blending:
  │     final = (1-α_hot) × semantic_score + α_hot × hotness_score
  │     hotness_score = log(active_count + 1) / log(max_active_count + 1)
  ├── Sort by final_score descending
  ├── Apply limit and threshold filters
  ├── Fetch related contexts (if .relations.json exists for matched URIs)
  └── Return QueryResult{matched_contexts, searched_directories}
```

### 3.4. Vector Schema (VectorDB)

```go
type ContextVector struct {
    URI            string            // Primary key
    ParentURI      string            // Parent directory URI
    ContextType    ContextType       // MEMORY=0, RESOURCE=1, SKILL=2, SESSION=3
    Level          int               // 0=Abstract, 1=Overview, 2=Detail
    OwnerAccountID string
    OwnerUserID    string
    Abstract       string            // L0 text for quick display
    ActiveCount    int64             // Usage counter for hotness
    DenseVector    []float32         // 768-3072 dimensions
    SparseVector   map[string]float32 // BM25/SPLADE features
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// Collection per tenant: "openviking_{account_id}"
// Hybrid search: dense + sparse with configurable alpha
```

### 3.5. Use Cases

| Use Case | Mô tả |
|----------|-------|
| `Find` | Stateless semantic search via HierarchicalRetriever (no session context) |
| `Search` | Session-aware search: includes current session's WM v2 + used URIs as rerank signal |
| `IndexContent` | Embed content → upsert to vector index (triggered by NATS or direct gRPC) |
| `RemoveContent` | Delete from vector index |
| `UpdateHotness` | Increment `active_count` for accessed URIs after session commit |
| `ReplaySearch` | Replay recorded search with debug info |

### 3.6. gRPC Service Definition

```protobuf
service SearchService {
  // Search operations
  rpc Find(FindRequest) returns (FindResponse);
  rpc Search(SearchRequest) returns (SearchResponse);

  // Index management
  rpc IndexContent(IndexContentRequest) returns (IndexContentResponse);
  rpc RemoveContent(RemoveContentRequest) returns (RemoveContentResponse);
  rpc UpdateHotness(UpdateHotnessRequest) returns (UpdateHotnessResponse);

  // Collection management
  rpc CreateCollection(CreateCollectionRequest) returns (CreateCollectionResponse);
  rpc CollectionExists(CollectionExistsRequest) returns (CollectionExistsResponse);

  // Debug/Replay
  rpc GetRetrievalStats(GetRetrievalStatsRequest) returns (GetRetrievalStatsResponse);
  rpc ReplaySearch(ReplaySearchRequest) returns (ReplaySearchResponse);
}

message FindRequest {
  string query = 1;
  string account_id = 2;
  string user_id = 3;
  optional string context_type = 4;       // MEMORY | RESOURCE | SKILL | SESSION
  repeated string target_directories = 5; // Restrict scope
  int32 limit = 6;                        // default: 10
  double threshold = 7;                   // minimum score
  bool reranker_enabled = 8;
}

message FindResponse {
  repeated MatchedContext results = 1;
  repeated string searched_directories = 2;
  int64 latency_ms = 3;
}

message MatchedContext {
  string uri = 1;
  string parent_uri = 2;
  string abstract = 3;      // L0 summary text
  double score = 4;
  double semantic_score = 5;
  double hotness_score = 6;
  string context_type = 7;
  int32 level = 8;
}
```

### 3.7. Key Parameters

| Parameter | Default | Mô tả |
|-----------|---------|-------|
| `GLOBAL_SEARCH_TOPK` | 10 | Số candidates toàn cục |
| `MAX_CONVERGENCE_ROUNDS` | 3 | Dừng sau N rounds stable |
| `DIRECTORY_DOMINANCE_RATIO` | 1.2 | Dir phải vượt max child score |
| `score_propagation_alpha` | 0.7 | Trọng số parent trong score blending |
| `hotness_alpha` | 0.1 | Trọng số recency vs semantic |
| `rerank_threshold` | 0.35 | Minimum reranker score |

### 3.8. Event-Driven Index Sync (NATS)

```
Subscribe: ov.content.written
  → Parse: {uri, account_id, context_type, level}
  → FS.Read(uri) → get content
  → Bifrost.Embed(content, isQuery=false) → dense + sparse vectors
  → VectorDB.Upsert(uri, vectors, metadata)

Subscribe: ov.content.deleted
  → Parse: {uri, account_id}
  → VectorDB.Delete(uri)

Subscribe: ov.session.committed
  → Parse: {used_uris[], account_id}
  → For each uri: VectorDB.UpdateActiveCount(uri, +1)
    (implements hotness boosting)
```

### 3.9. Configuration

```yaml
search:
  grpc:
    port: 9012
  health:
    port: 9092
  vectordb:
    provider: "qdrant"       # qdrant | weaviate | embedded
    url: "http://qdrant:6333"
    collection_prefix: "openviking_"
  retrieval:
    global_search_topk: 10
    max_convergence_rounds: 3
    directory_dominance_ratio: 1.2
    score_propagation_alpha: 0.7
    hotness_alpha: 0.1
    threshold: 0.0
  reranker:
    provider: "jina"         # volcengine | openai | cohere | jina | local
    model: "jina-reranker-v2-base-multilingual"
    threshold: 0.35
    enabled: false            # disabled by default
  embedding:
    provider: "openai"       # via Bifrost
    model: "text-embedding-3-small"
    dense_dim: 1536
    sparse_enabled: false
  redis:
    url: "redis://redis:6379/2"
    cache_ttl: 120s           # search cache TTL
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
    consumer_group: "openviking-search"
```

---

## 4. Acceptance Criteria

- [ ] `Find(query="Alice coffee preference", context_type="MEMORY")` → chỉ search trong `viking://user/*/memories/`, không search resources.
- [ ] Score propagation hoạt động: file trong thư mục có score cao → file được boost điểm so với cùng file trong thư mục có score thấp.
- [ ] Convergence: search với 10,000 nodes → dừng đúng sau 3 rounds stable, không duyệt hết toàn bộ.
- [ ] Hotness blending: file có `active_count=100` phải rank cao hơn file tương tự với `active_count=1` khi `hotness_alpha > 0`.
- [ ] NATS trigger: ghi file mới → Search tự động index trong < 30s.
- [ ] `UpdateHotness([uri1, uri2])` → `active_count` tăng 1 cho mỗi URI.
- [ ] Reranker (khi enabled): kết quả thay đổi thứ tự khác với pure semantic ranking.
- [ ] `ReplaySearch(snapshot_id)` → reproduce kết quả search cũ từ recorded IO.
- [ ] Session-aware `Search` → WM v2 của session được dùng làm context bổ sung cho reranking.
