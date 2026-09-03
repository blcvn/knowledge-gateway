# Solution: SOL-OV-003 — Search Service (Hierarchical Retrieval Engine)

**CR:** [CR-OV-003](../CR-OV-003-Search-Service.md)  
**Wave:** 4 (Search — sau Filesystem service)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/openviking-search` — Core innovation của OpenViking: **HierarchicalRetriever** 6-step algorithm thay thế flat RAG. Kết quả: +49% task completion, 83-91% token reduction.

### Chiến lược chính

| Innovation | Giải pháp |
|---|---|
| Flat vector search → Hierarchical | 6-step recursive retriever với max-heap priority queue |
| Score không phản ánh context | Score propagation: `α × child + (1-α) × parent` |
| Fresh/popular items không được ưu tiên | Hotness blending: `(1-α_hot) × semantic + α_hot × hotness` |
| Search vô hạn không kết thúc | Convergence detection: dừng sau 3 rounds top-K stable |
| Index phải manual trigger | Event-driven: subscribe `ov.content.written` → auto-embed + upsert |
| Session không aware khi search | Session-aware search: WM v2 + used_uris làm rerank signal |

---

## 2. Codebase Structure

```
services/openviking-search/
├── cmd/server/main.go
├── api/proto/search/v1/search.proto
├── internal/
│   ├── domain/
│   │   ├── query.go            # TypedQuery, QueryResult, MatchedContext
│   │   ├── retriever_config.go # RetrieverConfig với all tuning params
│   │   ├── io_record.go        # SearchIO for debug/replay recording
│   │   └── errors.go
│   ├── usecase/
│   │   ├── find.go             # Stateless HierarchicalRetriever
│   │   ├── search.go           # Session-aware search (extends Find)
│   │   ├── index_content.go    # Embed + upsert to VectorDB
│   │   ├── remove_content.go   # Delete from VectorDB
│   │   ├── update_hotness.go   # Increment active_count
│   │   ├── replay_search.go    # Debug replay
│   │   ├── retriever/
│   │   │   ├── hierarchical.go # Core 6-step algorithm
│   │   │   ├── priority_queue.go # max-heap for scored contexts
│   │   │   └── convergence.go  # Convergence detection logic
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go       # VectorStore, EmbedderClient, RerankerClient, FSClient
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   ├── vectordb/qdrant/    # Qdrant VectorStore adapter
│   │   ├── cache/redis/        # Search result caching
│   │   └── event/
│   │       ├── publisher.go
│   │       └── subscriber.go   # ov.content.written, ov.content.deleted, ov.session.committed
│   └── infra/
```

---

## 3. Domain Model

```go
// internal/domain/query.go

type TypedQuery struct {
    Query             string
    AccountID         string
    UserID            string
    ContextType       *viking.ContextType // nil = all types
    TargetDirectories []string            // Optional: restrict scope
    Limit             int                 // default: 10
    Threshold         float64             // minimum score (default: 0.0)
    RerankerEnabled   bool
    SessionContext    *SessionContext     // For session-aware search
}

type SessionContext struct {
    WorkingMemory string   // WM v2 content (7 sections)
    UsedURIs      []string // URIs accessed in current session
}

type QueryResult struct {
    MatchedContexts     []MatchedContext
    SearchedDirectories []string
    LatencyMs           int64
    SearchIO            *SearchIO  // Recorded if debug mode
}

type MatchedContext struct {
    URI           string
    ParentURI     string
    ContextType   viking.ContextType
    Level         int
    Abstract      string
    Score         float64  // Final blended score
    SemanticScore float64  // Raw cosine similarity
    HotnessScore  float64  // log(active_count+1)/log(max+1)
}

// internal/domain/retriever_config.go
type RetrieverConfig struct {
    GlobalSearchTopK        int     // default: 10
    MaxConvergenceRounds    int     // default: 3
    DirectoryDominanceRatio float64 // default: 1.2
    ScorePropagationAlpha   float64 // default: 0.7 (parent weight)
    HotnessAlpha            float64 // default: 0.1
    Threshold               float64 // default: 0.0
    RerankerProvider        string  // volcengine | openai | cohere | jina | local
    DenseDim                int     // embedding dimension
    SparseEnabled           bool    // hybrid search
}
```

---

## 4. HierarchicalRetriever — 6-Step Algorithm Implementation

```go
// internal/usecase/retriever/hierarchical.go

type HierarchicalRetriever struct {
    vectorStore  port.VectorStore
    embedder     port.EmbedderClient
    reranker     port.RerankerClient
    fsClient     port.FSClient
    config       *domain.RetrieverConfig
}

func (r *HierarchicalRetriever) Retrieve(ctx context.Context, query *domain.TypedQuery) (*domain.QueryResult, error) {
    startTime := time.Now()
    
    // ─────────────────────────────────────────────────────
    // STEP 1: Determine Starting Directories
    // ─────────────────────────────────────────────────────
    startingDirs := query.TargetDirectories
    if len(startingDirs) == 0 && query.ContextType != nil {
        startingDirs = query.ContextType.RootURIs()
    } else if len(startingDirs) == 0 {
        // Cross-type: all root URIs for this account
        startingDirs = []string{
            "viking://user/" + query.AccountID + "/",
            "viking://resources/",
            "viking://agent/" + query.AccountID + "/",
        }
    }
    
    // ─────────────────────────────────────────────────────
    // STEP 2: Embed Query
    // ─────────────────────────────────────────────────────
    embedResult, err := r.embedder.Embed(ctx, query.Query, true)  // isQuery=true
    if err != nil {
        return nil, fmt.Errorf("embed query: %w", err)
    }
    denseVec  := embedResult.DenseVector
    sparseVec := embedResult.SparseVector  // empty if SparseEnabled=false
    
    // ─────────────────────────────────────────────────────
    // STEP 3: Global Vector Search (L0/L1 only, exclude L2)
    // ─────────────────────────────────────────────────────
    globalHits, err := r.vectorStore.SearchGlobalRoots(ctx, denseVec, sparseVec, query.AccountID, r.config.GlobalSearchTopK)
    if err != nil {
        return nil, fmt.Errorf("global search: %w", err)
    }
    
    // ─────────────────────────────────────────────────────
    // STEP 4: Merge Starting Points into Priority Queue
    // ─────────────────────────────────────────────────────
    pq := NewMaxHeap()
    
    // Add static root dirs with neutral score
    for _, dir := range startingDirs {
        pq.Push(&ScoredNode{URI: dir, Score: 0.5, IsDirectory: true})
    }
    
    // Add global hits (directories/L0/L1 nodes)
    for _, hit := range globalHits {
        if hit.Level <= 1 { // Only L0/L1 nodes as starting points
            pq.Push(&ScoredNode{URI: hit.URI, Score: hit.Score, IsDirectory: true, Raw: &hit})
        }
    }
    
    // Optional pre-rerank on merged set
    if query.RerankerEnabled && len(globalHits) > 0 {
        r.rerankGlobalHits(ctx, query.Query, pq)
    }
    
    // ─────────────────────────────────────────────────────
    // STEP 5: Recursive Directory Search with Convergence
    // ─────────────────────────────────────────────────────
    candidates := make(map[string]*domain.MatchedContext)  // dedup by URI
    convergenceCounter := 0
    var topKSnapshot []string
    searchedDirs := []string{}
    
    for pq.Len() > 0 && convergenceCounter < r.config.MaxConvergenceRounds {
        node := pq.Pop()
        searchedDirs = append(searchedDirs, node.URI)
        
        // Search children of this directory node
        children, err := r.vectorStore.SearchChildren(ctx, node.URI, denseVec, sparseVec, query.AccountID)
        if err != nil {
            continue
        }
        
        for _, child := range children {
            if child.Level == 2 {  // L2 = leaf file (full content)
                // Score propagation: child score weighted by parent score
                propagated := r.config.ScorePropagationAlpha*child.Score +
                    (1-r.config.ScorePropagationAlpha)*node.Score
                
                // Optional cross-encoder reranking on abstract
                finalScore := propagated
                if query.RerankerEnabled && child.Abstract != "" {
                    rerankResults, _ := r.reranker.Rerank(ctx, query.Query, []string{child.Abstract}, 1)
                    if len(rerankResults) > 0 && rerankResults[0].Score >= r.config.RerankerThreshold {
                        finalScore = rerankResults[0].Score
                    }
                }
                
                candidates[child.URI] = &domain.MatchedContext{
                    URI:           child.URI,
                    ParentURI:     child.ParentURI,
                    ContextType:   child.ContextType,
                    Level:         child.Level,
                    Abstract:      child.Abstract,
                    Score:         finalScore,
                    SemanticScore: child.Score,
                }
            } else if child.IsDirectory || child.Level <= 1 {
                // L0/L1 = subdirectory → recurse
                pq.Push(&ScoredNode{URI: child.URI, Score: child.Score, IsDirectory: true})
            }
        }
        
        // Convergence check
        newTopK := topKURIs(candidates, query.Limit)
        if slicesEqual(newTopK, topKSnapshot) {
            convergenceCounter++
        } else {
            convergenceCounter = 0
            topKSnapshot = newTopK
        }
    }
    
    // ─────────────────────────────────────────────────────
    // STEP 6: Post-Processing
    // ─────────────────────────────────────────────────────
    
    // 6a. Hotness blending
    maxActiveCount := findMaxActiveCount(candidates)
    for uri, ctx := range candidates {
        hotnessScore := 0.0
        if maxActiveCount > 0 {
            // Logarithmic normalization: log(count+1)/log(max+1)
            raw := candidates[uri].Raw
            if raw != nil {
                hotnessScore = math.Log(float64(raw.ActiveCount)+1) / math.Log(float64(maxActiveCount)+1)
            }
        }
        ctx.HotnessScore = hotnessScore
        ctx.Score = (1-r.config.HotnessAlpha)*ctx.SemanticScore + r.config.HotnessAlpha*hotnessScore
        candidates[uri] = ctx
    }
    
    // 6b. Session-aware reranking (if session context provided)
    if query.SessionContext != nil {
        r.applySessionReranking(ctx, candidates, query.SessionContext)
    }
    
    // 6c. Sort, filter, limit
    results := toSortedSlice(candidates)
    results = filterByThreshold(results, r.config.Threshold)
    if len(results) > query.Limit {
        results = results[:query.Limit]
    }
    
    // 6d. Fetch related contexts for top results
    r.enrichWithRelations(ctx, results)
    
    return &domain.QueryResult{
        MatchedContexts:     results,
        SearchedDirectories: searchedDirs,
        LatencyMs:           time.Since(startTime).Milliseconds(),
    }, nil
}
```

---

## 5. Priority Queue Implementation

```go
// internal/usecase/retriever/priority_queue.go

type ScoredNode struct {
    URI         string
    Score       float64
    IsDirectory bool
    Raw         *vectordb.ScoredContext
}

// MaxHeap — Go heap.Interface for max-score priority
type MaxHeap []*ScoredNode

func (h MaxHeap) Len() int            { return len(h) }
func (h MaxHeap) Less(i, j int) bool  { return h[i].Score > h[j].Score }  // MAX heap
func (h MaxHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *MaxHeap) Push(x interface{}) { *h = append(*h, x.(*ScoredNode)) }
func (h *MaxHeap) Pop() interface{} {
    old := *h
    n := len(old)
    x := old[n-1]
    *h = old[:n-1]
    return x
}

type PriorityQueue struct {
    h *MaxHeap
}

func NewMaxHeap() *PriorityQueue {
    h := &MaxHeap{}
    heap.Init(h)
    return &PriorityQueue{h: h}
}

func (pq *PriorityQueue) Push(node *ScoredNode) {
    heap.Push(pq.h, node)
}

func (pq *PriorityQueue) Pop() *ScoredNode {
    return heap.Pop(pq.h).(*ScoredNode)
}

func (pq *PriorityQueue) Len() int { return pq.h.Len() }
```

---

## 6. Event-Driven Index Sync

```go
// adapter/event/subscriber.go

// Subscribe: ov.content.written → embed and upsert to VectorDB
func (s *Subscriber) HandleContentWritten(msg *nats.Msg) {
    var payload struct {
        URI         string `json:"uri"`
        AccountID   string `json:"account_id"`
        ContextType string `json:"context_type"`
        Level       int    `json:"level"`
    }
    json.Unmarshal(msg.Data, &payload)
    
    // Fetch content from FS service
    content, err := s.fsClient.Read(context.Background(), payload.URI, payload.Level)
    if err != nil {
        msg.Nak()  // Retry later
        return
    }
    
    // Embed content (isQuery=false → document embedding)
    embedResult, err := s.embedder.Embed(context.Background(), string(content), false)
    if err != nil {
        msg.Nak()
        return
    }
    
    // Upsert to VectorDB
    parentURI := filepath.Dir(payload.URI) + "/"
    err = s.vectorStore.UpsertContext(context.Background(), vectordb.ContextVector{
        URI:            payload.URI,
        ParentURI:      parentURI,
        ContextType:    parseContextType(payload.ContextType),
        Level:          payload.Level,
        OwnerAccountID: payload.AccountID,
        Abstract:       extractAbstract(content, payload.Level), // If L0, content IS the abstract
        DenseVector:    embedResult.DenseVector,
        SparseVector:   embedResult.SparseVector,
    })
    if err != nil {
        msg.Nak()
        return
    }
    
    msg.Ack()
}

// Subscribe: ov.content.deleted → remove from VectorDB
func (s *Subscriber) HandleContentDeleted(msg *nats.Msg) {
    var payload struct{ URI, AccountID string }
    json.Unmarshal(msg.Data, &payload)
    
    s.vectorStore.DeleteContext(context.Background(), payload.URI)
    msg.Ack()
}

// Subscribe: ov.session.committed → update hotness (active_count)
func (s *Subscriber) HandleSessionCommitted(msg *nats.Msg) {
    var payload struct {
        UsedURIs  []string `json:"used_uris"`
        AccountID string   `json:"account_id"`
    }
    json.Unmarshal(msg.Data, &payload)
    
    for _, uri := range payload.UsedURIs {
        s.vectorStore.UpdateActiveCount(context.Background(), uri, +1)
    }
    msg.Ack()
}
```

---

## 7. Search Caching (Redis)

```go
// adapter/cache/redis/search_cache.go

// Cache key: sha256({query}+{account_id}+{context_type}+{target_dirs})
// TTL: 120s (configurable)
// Invalidation: when ov.content.written → delete matching cache keys

// NOTE: Cache only for Find (stateless); Search (session-aware) always fresh

func (c *SearchCache) Get(ctx context.Context, req *domain.TypedQuery) (*domain.QueryResult, bool) {
    key := c.cacheKey(req)
    data, err := c.redis.Get(ctx, key).Bytes()
    if err != nil {
        return nil, false
    }
    var result domain.QueryResult
    json.Unmarshal(data, &result)
    return &result, true
}

func (c *SearchCache) Set(ctx context.Context, req *domain.TypedQuery, result *domain.QueryResult) {
    key := c.cacheKey(req)
    data, _ := json.Marshal(result)
    c.redis.Set(ctx, key, data, c.ttl)
}
```

---

## 8. Session-Aware Search

```go
// internal/usecase/search.go

type SearchUseCase struct {
    find       *FindUseCase
    reranker   port.RerankerClient
    sessionClient port.SessionClient
}

func (uc *SearchUseCase) Execute(ctx context.Context, req dto.SearchRequest) (*domain.QueryResult, error) {
    // 1. Get session context (WM v2 + used URIs)
    sessionCtx, err := uc.sessionClient.GetSessionContext(ctx, req.SessionID)
    if err != nil {
        // Graceful degradation: proceed without session context
        sessionCtx = nil
    }
    
    // 2. Enhance query with Working Memory context
    enrichedQuery := req.Query
    if sessionCtx != nil && sessionCtx.WorkingMemory != "" {
        // Prepend WM summary to help semantic search find relevant context
        enrichedQuery = fmt.Sprintf("Context: %s\n\nQuery: %s",
            sessionCtx.WorkingMemory[:min(500, len(sessionCtx.WorkingMemory))],
            req.Query)
    }
    
    // 3. Run stateless Find
    result, err := uc.find.Execute(ctx, dto.FindRequest{
        Query:       enrichedQuery,
        AccountID:   req.AccountID,
        ContextType: req.ContextType,
        Limit:       req.Limit * 2,  // Fetch 2x for post-session reranking
        SessionContext: sessionCtx,
    })
    if err != nil {
        return nil, err
    }
    
    // 4. Boost URIs recently used in current session
    if sessionCtx != nil {
        usedURISet := toSet(sessionCtx.UsedURIs)
        for i := range result.MatchedContexts {
            if usedURISet[result.MatchedContexts[i].URI] {
                result.MatchedContexts[i].Score *= 1.2  // 20% boost for recently used
            }
        }
        // Re-sort after boost
        sort.Slice(result.MatchedContexts, func(i, j int) bool {
            return result.MatchedContexts[i].Score > result.MatchedContexts[j].Score
        })
    }
    
    // 5. Trim to requested limit
    if len(result.MatchedContexts) > req.Limit {
        result.MatchedContexts = result.MatchedContexts[:req.Limit]
    }
    
    return result, nil
}
```

---

## 9. Debug/Replay

```go
// internal/usecase/replay_search.go

// SearchIO recording: enabled per-request via header X-OpenViking-Debug: true
type SearchIO struct {
    ID              string
    Query           *domain.TypedQuery
    EmbeddedVector  []float32
    GlobalHits      []vectordb.ScoredContext
    DirectoriesVisited []string
    CandidatesPerRound [][]string
    FinalResult     *domain.QueryResult
    RecordedAt      time.Time
}

// Stored in Redis with TTL=1h:
// Key: "ov_search_io:{account_id}:{search_id}"

func (uc *ReplaySearchUseCase) Execute(ctx context.Context, searchID, accountID string) (*domain.QueryResult, error) {
    // Load recorded IO
    io, err := uc.ioStore.Get(ctx, accountID, searchID)
    if err != nil {
        return nil, err
    }
    
    // Replay with pre-computed embedding (skip embed API call)
    return uc.retriever.RetrieveFromVector(ctx, io.Query, io.EmbeddedVector)
}
```

---

## 10. Configuration

```yaml
search:
  grpc:
    port: 9012
  health:
    port: 9092

  vectordb:
    provider: "qdrant"
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
    provider: "jina"                       # disabled | jina | cohere | openai | local
    model: "jina-reranker-v2-base-multilingual"
    threshold: 0.35
    enabled: false
    
  embedding:
    provider: "bifrost"
    model: "text-embedding-3-small"
    dense_dim: 1536
    sparse_enabled: false
    
  cache:
    redis_url: "redis://redis:6379/2"
    ttl: 120s
    enabled: true
    
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
    consumer_group: "openviking-search"
    subjects:
      subscribe_content_written: "ov.content.written"
      subscribe_content_deleted: "ov.content.deleted"
      subscribe_session_committed: "ov.session.committed"
      subscribe_resource_ingested: "ov.resource.ingested"
      
  clients:
    fs: "openviking-fs:9011"
    session: "openviking-session:9013"  # For session-aware search
    
  debug:
    io_record_enabled: false   # Enable via X-OpenViking-Debug header
    io_store_ttl: 3600s
```

---

## 11. Testing Strategy

### Unit Tests
- `TestHierarchicalRetriever_ContextTypeFilter` — MEMORY type → only user/ agent/ URIs searched
- `TestHierarchicalRetriever_ScorePropagation` — parent score 0.8 → child score boosted vs parent score 0.2
- `TestHierarchicalRetriever_ConvergenceDetection` — after 3 stable rounds → stops
- `TestHierarchicalRetriever_HotnessBlending` — active_count=100 → higher final score than count=1
- `TestPriorityQueue_MaxHeap` — pop order: highest score first
- `TestSessionAwareSearch_UsedURIBoost` — URI in used_uris → 20% score boost
- `TestIndexContent_EmbedAndUpsert` — ov.content.written → embed called → vectorStore.UpsertContext called
- `TestUpdateHotness_IncrementActiveCount` — active_count++ for each URI in list
- `TestSearchCache_HitSkipsRetriever` — cached result → retriever not called

### Integration Tests
- `TestQdrantSearchE2E` — upsert 100 vectors → search → top-K accurate
- `TestNATSIndexSyncE2E` — write file to FS → NATS event → search finds new content
- `TestHotnessUpdateE2E` — session committed → hotness updated → higher in subsequent search

### Performance Tests
- `BenchmarkHierarchicalRetriever_10kNodes` — must complete in < 500ms
- `BenchmarkGlobalSearch_Qdrant` — 100k vectors, topK=10, < 50ms

---

## 12. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| Qdrant không available → search down | Cao | Circuit breaker → fallback basic keyword search (không có vector) |
| Score propagation diverges nếu tree quá sâu | Thấp | Max convergence rounds = 3; max tree depth từ FS = 10 |
| Embedding API latency (>500ms) | Trung bình | Bulkhead + timeout; cache query embeddings (TTL=60s) |
| Session WM context tăng query length → cost | Thấp | Truncate WM summary to 500 chars trước khi prepend |
| NATS message flood khi large dir written | Trung bình | Batch indexing: debounce 100ms, nhóm events cùng account |
| Reranker (cross-encoder) là hot path | Trung bình | Reranker chỉ gọi khi `reranker_enabled=true` (default: false) |
