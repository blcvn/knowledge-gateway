# TASK-OV-010 — `services/openviking-search` HierarchicalRetriever & Find

**Wave:** 4 (Search)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-001, TASK-OV-003 (vectordb, embedder, reranker), TASK-OV-009 (fs grpc client)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-OV-003 §3, §4, §5](../solutions/SOL-OV-003-Search-Service.md)  
**Port gRPC:** 9012

---

## Mục tiêu

Implement `services/openviking-search/internal/` — Core innovation của OpenViking: **HierarchicalRetriever** 6-step algorithm với Max-Heap priority queue, score propagation, hotness blending, convergence detection, và session-aware search.

---

## Cấu trúc thư mục

```
services/openviking-search/
├── cmd/server/main.go
├── api/proto/search/v1/search.proto
├── internal/
│   ├── domain/
│   │   ├── query.go           # TypedQuery, QueryResult, MatchedContext
│   │   ├── retriever_config.go # RetrieverConfig với tuning params
│   │   ├── search_io.go       # SearchIO for debug recording
│   │   └── errors.go
│   ├── usecase/
│   │   ├── find.go            # Stateless HierarchicalRetriever
│   │   ├── search.go          # Session-aware search (extends Find)
│   │   ├── index_content.go   # Embed + upsert to VectorDB
│   │   ├── remove_content.go  # Delete from VectorDB
│   │   ├── update_hotness.go  # Increment active_count
│   │   ├── retriever/
│   │   │   ├── hierarchical.go # Core 6-step algorithm
│   │   │   ├── priority_queue.go # max-heap
│   │   │   └── convergence.go  # Convergence detection
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go      # VectorStore, EmbedderClient, RerankerClient, FSClient
```

---

## 1. Domain Models

**File: `internal/domain/query.go`**

```go
type TypedQuery struct {
    Query              string
    AccountID          string
    UserID             string
    ContextType        *viking.ContextType  // nil = all types
    TargetDirectories  []string             // Optional: restrict scope
    Limit              int                  // default: 10
    Threshold          float64              // minimum score (default: 0.0)
    RerankerEnabled    bool
    SessionContext     *SessionContext
}

type SessionContext struct {
    WorkingMemory string   // WM v2 content (max 500 chars used)
    UsedURIs      []string // URIs accessed in current session
}

type QueryResult struct {
    MatchedContexts     []MatchedContext
    SearchedDirectories []string
    LatencyMs           int64
    SearchIO            *SearchIO  // Non-nil if debug mode
}

type MatchedContext struct {
    URI           string
    ParentURI     string
    ContextType   viking.ContextType
    Level         int
    Abstract      string
    Score         float64   // Final blended score
    SemanticScore float64   // Raw vector similarity
    HotnessScore  float64   // log(active_count+1)/log(max+1)
    ActiveCount   int64
    Raw           *vectordb.ScoredContext  // unexported field for internal use
}
```

**File: `internal/domain/retriever_config.go`**

```go
type RetrieverConfig struct {
    GlobalSearchTopK        int     // default: 10
    MaxConvergenceRounds    int     // default: 3
    ScorePropagationAlpha   float64 // default: 0.7 (parent weight in propagation)
    HotnessAlpha            float64 // default: 0.1 (hotness blend weight)
    Threshold               float64 // default: 0.0
    RerankerEnabled         bool
    RerankerThreshold       float64 // default: 0.35
}

func DefaultConfig() RetrieverConfig {
    return RetrieverConfig{
        GlobalSearchTopK:      10,
        MaxConvergenceRounds:  3,
        ScorePropagationAlpha: 0.7,
        HotnessAlpha:          0.1,
    }
}
```

---

## 2. Priority Queue

**File: `internal/usecase/retriever/priority_queue.go`**

```go
type ScoredNode struct {
    URI         string
    Score       float64
    IsDirectory bool
    Raw         *vectordb.ScoredContext
}

// MaxHeap implements heap.Interface — pop returns highest score
type MaxHeap []*ScoredNode

func (h MaxHeap) Len() int            { return len(h) }
func (h MaxHeap) Less(i, j int) bool  { return h[i].Score > h[j].Score }  // MAX heap
func (h MaxHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *MaxHeap) Push(x interface{}) { *h = append(*h, x.(*ScoredNode)) }
func (h *MaxHeap) Pop() interface{} {
    old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x
}

type PriorityQueue struct {
    h *MaxHeap
}

func NewPriorityQueue() *PriorityQueue
func (pq *PriorityQueue) Push(node *ScoredNode)
func (pq *PriorityQueue) Pop() *ScoredNode
func (pq *PriorityQueue) Len() int
```

---

## 3. Convergence Detector

**File: `internal/usecase/retriever/convergence.go`**

```go
type ConvergenceDetector struct {
    maxRounds   int     // default: 3
    topK        int     // compare top-K URIs
    roundsStable int    // consecutive stable rounds
    snapshot    []string
}

func NewConvergenceDetector(maxRounds, topK int) *ConvergenceDetector

// Update: given current candidates, check if top-K URIs have changed
// Returns true if converged (stop iteration)
func (cd *ConvergenceDetector) Update(candidates map[string]*domain.MatchedContext) bool {
    newTopK := topKURIs(candidates, cd.topK)
    if slicesEqual(newTopK, cd.snapshot) {
        cd.roundsStable++
    } else {
        cd.roundsStable = 0
        cd.snapshot = newTopK
    }
    return cd.roundsStable >= cd.maxRounds
}
```

---

## 4. HierarchicalRetriever — Core Algorithm

**File: `internal/usecase/retriever/hierarchical.go`**

```go
type HierarchicalRetriever struct {
    vectorStore  port.VectorStore
    embedder     port.EmbedderClient
    reranker     port.RerankerClient
    config       *domain.RetrieverConfig
}

func (r *HierarchicalRetriever) Retrieve(ctx context.Context, query *domain.TypedQuery) (*domain.QueryResult, error) {

    // ── STEP 1: Determine starting directories ──────────────────
    startingDirs := query.TargetDirectories
    if len(startingDirs) == 0 {
        if query.ContextType != nil {
            startingDirs = query.ContextType.RootURIs()
        } else {
            startingDirs = []string{
                "viking://user/" + query.AccountID + "/",
                "viking://resources/",
                "viking://agent/" + query.AccountID + "/",
            }
        }
    }

    // ── STEP 2: Embed query ──────────────────────────────────────
    embedResult, err := r.embedder.Embed(ctx, query.Query, true)
    // Handle error

    // ── STEP 3: Global search (L0/L1 only) ─────────────────────
    globalHits, _ := r.vectorStore.SearchGlobalRoots(ctx, embedResult.DenseVector,
        embedResult.SparseVector, query.AccountID, r.config.GlobalSearchTopK)

    // ── STEP 4: Merge into priority queue ───────────────────────
    pq := NewPriorityQueue()
    for _, dir := range startingDirs {
        pq.Push(&ScoredNode{URI: dir, Score: 0.5, IsDirectory: true})
    }
    for _, hit := range globalHits {
        if hit.Level <= 1 {
            pq.Push(&ScoredNode{URI: hit.URI, Score: hit.Score, IsDirectory: true, Raw: &hit})
        }
    }

    // ── STEP 5: Recursive search with convergence ────────────────
    candidates := make(map[string]*domain.MatchedContext)
    convergence := NewConvergenceDetector(r.config.MaxConvergenceRounds, query.Limit)
    searchedDirs := []string{}

    for pq.Len() > 0 {
        node := pq.Pop()
        searchedDirs = append(searchedDirs, node.URI)

        children, _ := r.vectorStore.SearchChildren(ctx, node.URI,
            embedResult.DenseVector, embedResult.SparseVector, query.AccountID)

        for _, child := range children {
            if child.Level == 2 {
                // Score propagation
                propagated := r.config.ScorePropagationAlpha*child.Score +
                    (1-r.config.ScorePropagationAlpha)*node.Score

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
                    ActiveCount:   child.ActiveCount,
                    Raw:           &child,
                }
            } else if child.IsDirectory || child.Level <= 1 {
                pq.Push(&ScoredNode{URI: child.URI, Score: child.Score, IsDirectory: true})
            }
        }

        if convergence.Update(candidates) {
            break
        }
    }

    // ── STEP 6: Post-processing ──────────────────────────────────
    maxActiveCount := findMaxActiveCount(candidates)
    for uri, mc := range candidates {
        hotnessScore := 0.0
        if maxActiveCount > 0 && mc.ActiveCount > 0 {
            hotnessScore = math.Log(float64(mc.ActiveCount)+1) / math.Log(float64(maxActiveCount)+1)
        }
        mc.HotnessScore = hotnessScore
        mc.Score = (1-r.config.HotnessAlpha)*mc.SemanticScore + r.config.HotnessAlpha*hotnessScore
        candidates[uri] = mc
    }

    // Sort by Score (desc), filter by threshold, limit
    results := toSortedSlice(candidates)
    results = filterByThreshold(results, r.config.Threshold)
    if len(results) > query.Limit { results = results[:query.Limit] }

    return &domain.QueryResult{
        MatchedContexts:     results,
        SearchedDirectories: searchedDirs,
        LatencyMs:           time.Since(start).Milliseconds(),
    }, nil
}
```

---

## 5. FindUseCase & SearchUseCase

**File: `internal/usecase/find.go`**

```go
type FindUseCase struct {
    retriever *retriever.HierarchicalRetriever
    cache     SearchCache  // Redis-backed, TTL=120s
}

func (uc *FindUseCase) Execute(ctx context.Context, query *domain.TypedQuery) (*domain.QueryResult, error) {
    // 1. Check cache (skip for session-aware queries)
    if query.SessionContext == nil {
        if cached, ok := uc.cache.Get(ctx, query); ok { return cached, nil }
    }
    // 2. Run retriever
    result, err := uc.retriever.Retrieve(ctx, query)
    // 3. Cache result (skip for session-aware)
    if err == nil && query.SessionContext == nil {
        uc.cache.Set(ctx, query, result)
    }
    return result, err
}
```

**File: `internal/usecase/search.go`**

```go
type SearchUseCase struct {
    find          *FindUseCase
    sessionClient port.SessionClient
}

func (uc *SearchUseCase) Execute(ctx context.Context, req SearchRequest) (*domain.QueryResult, error) {
    // 1. Get session context
    sessionCtx, _ := uc.sessionClient.GetSessionContext(ctx, req.SessionID)

    // 2. Enrich query with WM context (max 500 chars)
    enrichedQuery := req.Query
    if sessionCtx != nil && sessionCtx.WorkingMemory != "" {
        wmSummary := sessionCtx.WorkingMemory
        if len(wmSummary) > 500 { wmSummary = wmSummary[:500] }
        enrichedQuery = fmt.Sprintf("Context: %s\n\nQuery: %s", wmSummary, req.Query)
    }

    // 3. Find with 2x limit (for post-session reranking)
    result, err := uc.find.Execute(ctx, &domain.TypedQuery{
        Query:         enrichedQuery,
        AccountID:     req.AccountID,
        Limit:         req.Limit * 2,
        SessionContext: sessionCtx,
    })

    // 4. Boost recently-used URIs (+20%)
    if sessionCtx != nil {
        usedSet := toSet(sessionCtx.UsedURIs)
        for i := range result.MatchedContexts {
            if usedSet[result.MatchedContexts[i].URI] {
                result.MatchedContexts[i].Score *= 1.2
            }
        }
        sort.Slice(result.MatchedContexts, func(i, j int) bool {
            return result.MatchedContexts[i].Score > result.MatchedContexts[j].Score
        })
    }

    // 5. Trim to requested limit
    if len(result.MatchedContexts) > req.Limit {
        result.MatchedContexts = result.MatchedContexts[:req.Limit]
    }
    return result, err
}
```

---

## 6. Protobuf Definition

**File: `api/proto/search/v1/search.proto`**

```protobuf
syntax = "proto3";
package openviking.search.v1;

service SearchService {
  rpc Find(FindRequest)           returns (FindResponse);      // Stateless
  rpc Search(SearchRequest)       returns (SearchResponse);    // Session-aware
  rpc IndexContent(IndexRequest)  returns (IndexResponse);     // Embed + upsert
  rpc RemoveContent(RemoveRequest) returns (RemoveResponse);
  rpc UpdateHotness(UpdateHotnessRequest) returns (UpdateHotnessResponse);
  rpc GetSessionContext(GetSessionContextRequest) returns (GetSessionContextResponse);
}

message FindRequest {
  string query        = 1;
  string account_id   = 2;
  string user_id      = 3;
  int32  context_type = 4; // -1=all, 0=MEMORY, 1=RESOURCE, 2=SKILL, 3=SESSION
  repeated string target_dirs = 5;
  int32  limit        = 6;
  double threshold    = 7;
  bool   reranker_enabled = 8;
}

message FindResponse {
  repeated MatchedContextProto contexts = 1;
  repeated string searched_dirs = 2;
  int64 latency_ms = 3;
}

message MatchedContextProto {
  string uri          = 1;
  string parent_uri   = 2;
  int32  context_type = 3;
  int32  level        = 4;
  string abstract     = 5;
  double score        = 6;
  double semantic_score = 7;
  double hotness_score  = 8;
}

message SearchRequest {
  string query      = 1;
  string account_id = 2;
  string session_id = 3;
  int32  limit      = 4;
}

message IndexRequest {
  string uri         = 1;
  string account_id  = 2;
  string context_type = 3;
  int32  level       = 4;
}

message UpdateHotnessRequest {
  repeated string uris = 1;
}
```

---

## Unit Tests

```
TestHierarchicalRetriever_EmptyVectorDB    → empty DB → empty results
TestHierarchicalRetriever_FindsLeafNodes   → insert L2 node → Find → returned
TestHierarchicalRetriever_ScorePropagation → parent score 0.8 → child boosted vs parent 0.2
TestHierarchicalRetriever_ContextTypeFilter → MEMORY type → only user/ and agent/ searched
TestHierarchicalRetriever_ConvergenceStops → stable top-K 3 rounds → stops
TestHierarchicalRetriever_HotnessBlending  → high active_count → higher final score
TestHierarchicalRetriever_Threshold        → threshold=0.5 → low-score results excluded
TestHierarchicalRetriever_Limit            → 100 candidates → limit=10 → 10 returned
TestPriorityQueue_MaxOrder                 → push 0.1, 0.9, 0.5 → pop order: 0.9, 0.5, 0.1
TestPriorityQueue_Empty                    → pop from empty → nil (no panic)
TestConvergenceDetector_ConvergesAfter3    → stable 3 rounds → returns true
TestConvergenceDetector_ResetsOnChange     → stable 2, change, stable 3 → 5 total
TestSessionAwareSearch_BoostsUsedURIs      → URI in used_uris → 20% score boost
TestSessionAwareSearch_GracefulWithoutSession → no session → fallback to stateless Find
TestFindUseCase_CacheHit                   → 2nd call → retriever not called
TestFindUseCase_SessionAware_SkipsCache    → session search → no caching
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

buf generate services/openviking-search/
go build ./services/openviking-search/...
go test ./services/openviking-search/internal/... -v -count=1
```

---

## Ghi chú triển khai

- `vectordb.memory.InMemoryVectorStore` dùng cho unit tests (không cần Qdrant)
- `embedder.disabled.DisabledEmbedder` trả về zero vectors cho unit tests
- Math: `math.Log(x+1)/math.Log(max+1)` — tránh log(0) với +1 offset
- `sort.Slice` stable để tránh kết quả không deterministic khi score bằng nhau
- Cache key: `sha256(query+account_id+context_type+sorted(target_dirs))`
