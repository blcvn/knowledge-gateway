# Solution: SOL-003 — Hybrid Search Engine (BM25 + Vector + RRF)

**CR ID:** CR-AM-003  
**Solution ID:** SOL-003  
**Priority:** Critical (Wave 1)  
**Architecture:** NEW `services/observe-search/` + `pkg/search/` shared package

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §5.1`:
- `services/search-service/` đã có domain: `search/`, `connector/`, `mcp/`.
- `services/cognee-search` dùng Qdrant (external vector DB).
- VNP Memory monolith dùng `vnp-search-hub` cho cross-engine search.

**Chiến lược:** Tạo **service mới `observe-search`** — hoàn toàn in-memory search cho agentmemory, không dùng Qdrant. Đây là service thứ 37 trong monolith. Dùng `pkg/search/` (shared package) để tái sử dụng giữa các services.

**Điểm quan trọng:** Service này dùng **SQLite** (không phải PostgreSQL) cho local KV store — giữ observations và index riêng, tối ưu cho cá nhân developer.

> **Đơn giản hóa local embedding:** Thay vì ONNX runtime, dùng **Bifrost** (LLM gateway đã có) với provider `local` hoặc `openai`. Cấu hình `EMBEDDING_PROVIDER=none` → chỉ dùng BM25 (zero cost). Provider `openai/gemini` → dùng API.

---

## 2. Giải pháp

### 2.1. Shared Package `pkg/search/`

```
pkg/search/
├── bm25.go           # BM25 in-memory inverted index
├── vector_index.go   # Dense vector cosine index (384-dim or configurable)
├── rrf.go            # Reciprocal Rank Fusion
├── tokenizer.go      # BM25 tokenizer + CJK bigrams + Porter stemmer
├── persistence.go    # Gob serialize/deserialize + 30s debounce save
└── types.go          # BM25Result, VectorResult, HybridResult, ScoreWeights
```

#### BM25 Index (in-memory + persistent)

```go
// pkg/search/bm25.go

const (
    k1 = 1.25
    b  = 0.75
)

type BM25Index struct {
    mu          sync.RWMutex
    invertedIdx map[string][]Posting  // term → postings
    docLengths  map[string]int        // docID → term count
    docMeta     map[string]DocMeta    // docID → {sessionID, agentID}
    totalDocs   int
    totalLength int
    dirty       bool                  // needs persist
}

type Posting struct {
    DocID     string
    SessionID string
    TF        int
}

type DocMeta struct {
    SessionID string
    AgentID   string
}

func NewBM25Index() *BM25Index { ... }

func (b *BM25Index) Add(docID, sessionID, agentID string, text string) {
    terms := Tokenize(text)
    b.mu.Lock()
    defer b.mu.Unlock()
    // Build posting list + update docLengths + totalDocs + totalLength
    b.dirty = true
}

func (b *BM25Index) Remove(docID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    // Remove all postings for docID
    b.dirty = true
}

func (b *BM25Index) Search(query string, limit int) []BM25Result {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    terms := Tokenize(query)
    scores := map[string]float64{}
    avgdl := float64(b.totalLength) / math.Max(float64(b.totalDocs), 1)
    
    for _, term := range terms {
        postings := b.invertedIdx[term]
        df := float64(len(postings))
        idf := math.Log((float64(b.totalDocs)-df+0.5)/(df+0.5) + 1)
        
        for _, p := range postings {
            dl := float64(b.docLengths[p.DocID])
            tf := float64(p.TF)
            tfNorm := (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgdl))
            scores[p.DocID] += idf * tfNorm
        }
    }
    
    // Sort descending, top-limit
    return sortAndTrim(scores, b.docMeta, limit)
}

// Serialization: gob encoding
func (b *BM25Index) Save(path string) error {
    b.mu.RLock()
    defer b.mu.RUnlock()
    // encode invertedIdx + docLengths + docMeta to gob file
}

func (b *BM25Index) Load(path string) error { ... }
```

#### Tokenizer

```go
// pkg/search/tokenizer.go

func Tokenize(text string) []string {
    text = strings.ToLower(text)
    var tokens []string
    
    // ASCII word splitting
    for _, word := range strings.FieldsFunc(text, func(r rune) bool {
        return !unicode.IsLetter(r) && !unicode.IsDigit(r)
    }) {
        stemmed := porterStem(word)
        if len(stemmed) >= 2 && !isStopword(stemmed) {
            tokens = append(tokens, stemmed)
        }
    }
    
    // CJK bigram tokenization
    tokens = append(tokens, cjkBigrams(text)...)
    return tokens
}
```

#### Vector Index (cosine similarity, brute-force for <50K docs)

```go
// pkg/search/vector_index.go

type VectorIndex struct {
    mu       sync.RWMutex
    vectors  map[string][]float32   // docID → embedding
    sessions map[string]string      // docID → sessionID
    dims     int                    // e.g. 384 for all-MiniLM, 1536 for OpenAI
}

func (v *VectorIndex) Add(docID, sessionID string, vec []float32) error {
    if len(vec) != v.dims { return ErrDimensionMismatch }
    v.mu.Lock()
    v.vectors[docID] = vec
    v.sessions[docID] = sessionID
    v.mu.Unlock()
    return nil
}

func (v *VectorIndex) Search(query []float32, limit int) []VectorResult {
    v.mu.RLock()
    defer v.mu.RUnlock()
    
    scored := make([]VectorResult, 0, len(v.vectors))
    for docID, vec := range v.vectors {
        score := cosineSimilarity(query, vec)
        scored = append(scored, VectorResult{DocID: docID, SessionID: v.sessions[docID], Score: score})
    }
    sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
    if limit > len(scored) { limit = len(scored) }
    return scored[:limit]
}

func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    if normA == 0 || normB == 0 { return 0 }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

#### RRF Fusion

```go
// pkg/search/rrf.go

const rrfK = 60

type HybridResult struct {
    DocID         string
    SessionID     string
    CombinedScore float64
    BM25Score     float64
    VectorScore   float64
    GraphScore    float64
    BM25Rank      int
    VectorRank    int
}

func RRFFuse(bm25Results []BM25Result, vectorResults []VectorResult, graphResults []GraphResult,
    weights ScoreWeights, limit int) []HybridResult {
    
    scores := map[string]*HybridResult{}
    
    for rank, r := range bm25Results {
        if scores[r.DocID] == nil { scores[r.DocID] = &HybridResult{DocID: r.DocID, SessionID: r.SessionID} }
        scores[r.DocID].CombinedScore += weights.BM25 * (1.0 / float64(rrfK+rank+1))
        scores[r.DocID].BM25Rank = rank + 1
        scores[r.DocID].BM25Score = r.Score
    }
    for rank, r := range vectorResults {
        if scores[r.DocID] == nil { scores[r.DocID] = &HybridResult{DocID: r.DocID, SessionID: r.SessionID} }
        scores[r.DocID].CombinedScore += weights.Vector * (1.0 / float64(rrfK+rank+1))
        scores[r.DocID].VectorRank = rank + 1
        scores[r.DocID].VectorScore = r.Score
    }
    for rank, r := range graphResults {
        if scores[r.DocID] == nil { scores[r.DocID] = &HybridResult{DocID: r.DocID} }
        scores[r.DocID].CombinedScore += weights.Graph * (1.0 / float64(rrfK+rank+1))
        scores[r.DocID].GraphScore = r.Score
    }
    
    results := make([]HybridResult, 0, len(scores))
    for _, v := range scores { results = append(results, *v) }
    sort.Slice(results, func(i, j int) bool { return results[i].CombinedScore > results[j].CombinedScore })
    if limit > len(results) { limit = len(results) }
    return results[:limit]
}
```

#### Index Persistence (debounced)

```go
// pkg/search/persistence.go

type IndexPersister struct {
    bm25   *BM25Index
    vector *VectorIndex
    dir    string
    timer  *time.Timer
    mu     sync.Mutex
}

// Debounced save: calls to Schedule() reset the 30s timer
// After 30s of no writes → save both indexes to disk
func (p *IndexPersister) Schedule() {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.timer != nil { p.timer.Stop() }
    p.timer = time.AfterFunc(30*time.Second, func() {
        p.bm25.Save(filepath.Join(p.dir, "bm25.gob"))
        p.vector.Save(filepath.Join(p.dir, "vector.gob"))
    })
}

// On startup: load indexes async (server available immediately)
func (p *IndexPersister) LoadAsync() {
    go func() {
        p.bm25.Load(filepath.Join(p.dir, "bm25.gob"))
        p.vector.Load(filepath.Join(p.dir, "vector.gob"))
    }()
}
```

### 2.2. `services/observe-search/` — New Service

```
services/observe-search/
├── cmd/search/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go          # SearchResult, ContextBlock, QueryExpansion
│   │   └── value_object.go    # SearchStrategy, ScoreWeights
│   ├── search/
│   │   ├── smart_search.go    # SmartSearch: orchestrate BM25+Vector+Graph+RRF
│   │   ├── context_builder.go # Token-budget context assembly
│   │   └── query_expand.go    # LLM query expansion (optional)
│   ├── index/
│   │   ├── manager.go         # Index lifecycle: add, remove, rebuild
│   │   └── rebuilder.go       # Full index rebuild from observation KV
│   ├── usecase/
│   │   ├── smart_search.go
│   │   ├── bm25_search.go
│   │   ├── vector_search.go
│   │   ├── build_context.go
│   │   ├── index_add.go
│   │   ├── index_remove.go
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go      # IObservationStore, IEmbedder
│   └── adapter/
│       ├── grpc/handler.go
│       ├── http/handler.go    # REST fallback
│       ├── repository/
│       │   └── postgres/
│       │       └── obs_store.go   # Read compressed observations
│       └── bifrost/
│           └── embedder.go    # Bifrost embedding client
└── api/proto/search/v1/observe_search.proto
```

### 2.3. Smart Search Use Case

```go
// services/observe-search/internal/usecase/smart_search.go

type SmartSearchUseCase struct {
    bm25    *search.BM25Index
    vector  *search.VectorIndex
    embedder port.IEmbedder       // Bifrost or local
    weights  search.ScoreWeights
}

func (uc *SmartSearchUseCase) Execute(ctx context.Context, req SmartSearchRequest) (*SmartSearchResponse, error) {
    start := time.Now()
    
    // 1. BM25 search (always)
    bm25Results := uc.bm25.Search(req.Query, req.Limit*3)
    
    // 2. Vector search (if embedder configured)
    var vectorResults []search.VectorResult
    if uc.embedder != nil {
        vec, err := uc.embedder.Embed(ctx, req.Query)
        if err == nil {
            vectorResults = uc.vector.Search(vec, req.Limit*3)
        }
        // Silent fail: vector unavailable → BM25 only
    }
    
    // 3. RRF fusion
    weights := req.Weights
    if weights.BM25 == 0 { weights = uc.weights } // use default
    hybridResults := search.RRFFuse(bm25Results, vectorResults, nil, weights, req.Limit)
    
    // 4. Enrich with observation data
    results, _ := uc.enrichResults(ctx, hybridResults, req)
    
    return &SmartSearchResponse{
        Results: results,
        TookMs:  time.Since(start).Milliseconds(),
    }, nil
}
```

### 2.4. Context Builder (Token Budget)

```go
// services/observe-search/internal/search/context_builder.go

type ContextBuilder struct {
    obsStore  port.IObservationStore
    memClient port.IAgentMemoryClient  // gRPC to memory-service
    smartSearch *SmartSearchUseCase
}

func (cb *ContextBuilder) Build(ctx context.Context, req ContextRequest) (*ContextResponse, error) {
    var blocks []ContextBlock
    budget := req.TokenBudget
    
    // P1: Recent high-strength memories from memory-service (gRPC call)
    memories, _ := cb.memClient.ListLatest(ctx, req.TenantID, req.Project, 30) // last 30 days, strength > 0.5
    for _, m := range memories {
        tokens := len(m.Content) / 4
        if budget-tokens < 0 { break }
        blocks = append(blocks, ContextBlock{
            Type: "memory", Content: formatMemory(m), Tokens: tokens,
            Recency: math.Exp(-float64(time.Since(m.UpdatedAt).Hours()/24)/7),
            Source: m.ID,
        })
        budget -= tokens
    }
    
    // P2: Last 3 session summaries
    summaries, _ := cb.obsStore.GetRecentSummaries(ctx, req.TenantID, req.Project, 3)
    for _, s := range summaries {
        tokens := len(s.Narrative) / 4
        if budget-tokens < 0 { break }
        blocks = append(blocks, ContextBlock{Type: "summary", Content: s.Narrative, Tokens: tokens})
        budget -= tokens
    }
    
    // P3: Relevant observations (if query provided)
    if req.Query != "" && budget > 100 {
        searchResults, _ := cb.smartSearch.Execute(ctx, SmartSearchRequest{Query: req.Query, Limit: 5})
        for _, r := range searchResults.Results {
            tokens := len(r.Narrative) / 4
            if budget-tokens < 0 { break }
            blocks = append(blocks, ContextBlock{Type: "observation", Content: r.Narrative, Tokens: tokens})
            budget -= tokens
        }
    }
    
    formatted := formatContextBlocks(blocks)
    return &ContextResponse{Blocks: blocks, TotalTokens: req.TokenBudget - budget, Formatted: formatted}, nil
}
```

### 2.5. Embedding via Bifrost

```go
// services/observe-search/internal/adapter/bifrost/embedder.go

type BifrostEmbedder struct {
    client *http.Client
    url    string     // Bifrost URL
    model  string     // e.g. "text-embedding-3-small"
    dims   int
}

func (b *BifrostEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // POST /v1/embeddings → Bifrost → provider
    // If EMBEDDING_PROVIDER=none → return nil (search falls back to BM25-only)
}

// NullEmbedder: for EMBEDDING_PROVIDER=none
type NullEmbedder struct{}
func (n *NullEmbedder) Embed(_ context.Context, _ string) ([]float32, error) { return nil, nil }
```

### 2.6. Index Add/Remove (called by observe-service)

```go
// services/observe-search/internal/usecase/index_add.go
// Called via gRPC from observe-service after each observation

func (uc *IndexAddUseCase) Execute(ctx context.Context, req IndexAddRequest) error {
    // Build text for BM25 from CompressedObservation
    text := req.Title + " " + strings.Join(req.Facts, " ") + " " + strings.Join(req.Concepts, " ")
    
    // Add to BM25
    uc.bm25.Add(req.ObsID, req.SessionID, req.AgentID, text)
    
    // Add to Vector (if embedder available)
    if uc.embedder != nil {
        vec, err := uc.embedder.Embed(ctx, text)
        if err == nil {
            uc.vector.Add(req.ObsID, req.SessionID, vec)
        }
    }
    
    // Schedule persist (debounced 30s)
    uc.persister.Schedule()
    
    return nil
}
```

### 2.7. gRPC Proto

```protobuf
// api/proto/search/v1/observe_search.proto

service ObserveSearchService {
  rpc SmartSearch(SmartSearchRequest) returns (SmartSearchResponse);
  rpc BM25Search(BM25SearchRequest) returns (BM25SearchResponse);
  rpc VectorSearch(VectorSearchRequest) returns (VectorSearchResponse);
  rpc BuildContext(ContextRequest) returns (ContextResponse);
  rpc IndexAdd(IndexAddRequest) returns (IndexAddResponse);
  rpc IndexRemove(IndexRemoveRequest) returns (IndexRemoveResponse);
  rpc RebuildIndex(RebuildIndexRequest) returns (RebuildIndexResponse);
  rpc GetIndexStats(GetIndexStatsRequest) returns (GetIndexStatsResponse);
}
```

### 2.8. Bootstrap Integration

```go
// apps/memory/internal/bootstrap/observe_search.go — NEW

func InitObserveSearch(reg *bus.InProcessRegistry, db *sql.DB, bifrost *bifrost.Client, cfg *config.Config) {
    // Init indexes
    bm25 := search.NewBM25Index()
    vector := search.NewVectorIndex(cfg.Search.EmbedDims)
    
    // Init embedder (null if EMBEDDING_PROVIDER=none)
    var embedder port.IEmbedder = &bifrost.NullEmbedder{}
    if cfg.Search.EmbeddingProvider != "none" {
        embedder = bifrostadapter.NewEmbedder(bifrost, cfg.Search.EmbeddingModel)
    }
    
    // Index persistence
    persister := search.NewIndexPersister(bm25, vector, cfg.Search.DataDir)
    persister.LoadAsync() // Load from disk on startup
    
    // Obs store (read compressed observations from PostgreSQL)
    obsStore := postgres.NewObservationStore(db)
    
    // Build use cases
    smartSearchUC := usecase.NewSmartSearch(bm25, vector, embedder, cfg.Search.DefaultWeights)
    contextBuilderUC := usecase.NewBuildContext(obsStore, memClient, smartSearchUC)
    indexAddUC := usecase.NewIndexAdd(bm25, vector, embedder, persister)
    
    // Register gRPC
    grpcServer := grpc.NewServer()
    searchpb.RegisterObserveSearchServiceServer(grpcServer, grpchandler.NewHandler(
        smartSearchUC, contextBuilderUC, indexAddUC, ...
    ))
    reg.Register("am-search", grpcServer)
}
```

### 2.9. Gateway Routes

```go
r.Post("/v1/observe/search/smart",   h.ForwardTo("am-search", "ObserveSearchService/SmartSearch"))
r.Post("/v1/observe/search/bm25",    h.ForwardTo("am-search", "ObserveSearchService/BM25Search"))
r.Post("/v1/observe/search/vector",  h.ForwardTo("am-search", "ObserveSearchService/VectorSearch"))
r.Post("/v1/observe/search/context", h.ForwardTo("am-search", "ObserveSearchService/BuildContext"))
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `pkg/search/bm25.go` | BM25 in-memory index |
| `pkg/search/vector_index.go` | Cosine vector index |
| `pkg/search/rrf.go` | RRF fusion algorithm |
| `pkg/search/tokenizer.go` | BM25 tokenizer + CJK bigrams |
| `pkg/search/persistence.go` | Gob debounced persistence |
| `services/observe-search/` | Full service |
| `apps/memory/internal/bootstrap/observe_search.go` | Bootstrap |
| `api/proto/search/v1/observe_search.proto` | Proto |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `apps/memory/internal/bootstrap/bootstrap.go` | Gọi InitObserveSearch() |
| `gateway/internal/adapter/handler/router.go` | Routes `/v1/observe/search/*` |
| `services/observe-service/internal/adapter/event/publisher.go` | Gọi IndexAdd sau observe |
| `apps/memory/configs/config.yaml` | Thêm `search:` section |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-AM-003 | Covered by |
|-----------------|------------|
| SmartSearch semantic match (no keyword overlap) | Vector index + RRF |
| p50 latency ≤ 14ms với 10K docs | Brute-force cosine O(n·d) |
| BM25 index survive restart | bm25.gob load on startup |
| Vector index survive restart | vector.gob load on startup |
| Local embedding (zero API key) | EMBEDDING_PROVIDER=none → NullEmbedder |
| Context token budget ≤ 1000 | context_builder.go budget tracking |
| Index add → searchable in < 1s | IndexAdd + BM25.Add (in-memory, instant) |
| GET health → index stats | GetIndexStats gRPC |
