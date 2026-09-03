# Change Request: CR-AM-003 — Hybrid Search Engine (BM25 + Vector + Graph + RRF)

**CR ID:** CR-AM-003  
**Component:** `services/search-service` [NEW SERVICE] | `services/cognee-search` [EXTEND]  
**Priority:** Critical  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.2, SRS FR-SEARCH-001..005, FR-GRAPH-001..004, FR-CTX-001..002  
**Spec:** `references/agentmemory/specs/services/search-service/spec.md`

---

## 1. Mô tả

Xây dựng **agentmemory Search Service** với Hybrid RRF engine kết hợp 3 signals: BM25 (keyword), Vector (semantic, local embedding), và Graph traversal — tất cả trong memory, không cần gọi Qdrant bên ngoài. Đạt **p50 latency ≤ 14ms**, **R@5 ≥ 95.2%** trên LongMemEval-S.

Đây là service độc lập, khác với `cognee-search` (vốn dựa vào Qdrant external service). Search service này tối ưu cho **coding agent memory** với local embedding (`all-MiniLM-L6-v2`, 384 dimensions) — zero API cost.

---

## 2. Vấn đề hiện tại

`services/cognee-search` trong VNP Memory:
- Dựa vào **Qdrant** external service — phức tạp hơn cần thiết cho local agent memory.
- Không có **BM25 in-memory index**.
- Không có **local embedding** (all-MiniLM-L6-v2) — luôn cần API key.
- Không có **query expansion** (LLM-based synonym expansion).
- Không có **context builder** với token budget management.
- Không có **index persistence** (BM25/vector survive restart).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/observe-search/` (search service cho agentmemory)

**Port:** `8082`  
**Binary:** `cmd/search/main.go`

**Cấu trúc:**
```
services/observe-search/
├── cmd/search/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # SearchResult, ContextBlock, QueryExpansion
│   │   └── value_object.go     # SearchStrategy, ScoreWeights
│   ├── search/
│   │   ├── smart_search.go     # Hybrid RRF engine
│   │   ├── context_builder.go  # Token-budget context assembly
│   │   ├── query_expand.go     # LLM query expansion
│   │   └── rebuild.go          # Index rebuild from KV
│   ├── usecase/
│   │   └── port/output.go      # KVStore, BM25Index, VectorIndex, GraphClient
│   └── adapter/
│       ├── http/handler.go
│       └── repository/sqlite/
└── pkg/search/ (shared)
    ├── bm25.go                 # BM25 in-memory index
    ├── vector_index.go         # Vector index (384-dim)
    ├── rrf.go                  # Reciprocal Rank Fusion
    ├── tokenizer.go            # BM25 tokenizer + CJK bigrams
    └── persistence.go          # 30s debounce disk save
```

### 3.2. BM25 Index (in-memory, persisted to disk)

```go
// pkg/search/bm25.go
// BM25 implementation with:
// - Inverted index: term → [(obsId, sessionId, tf), ...]
// - IDF: log((N-df+0.5)/(df+0.5)+1)
// - TF normalization: (tf*(1.25+1)) / (tf + 1.25*(1-0.75+0.75*dl/avgdl))
// - k1=1.25, b=0.75 (standard BM25 params)
// - CJK bigram segmentation
// - Porter stemmer for English

type BM25Index struct {
    mu          sync.RWMutex
    invertedIdx map[string][]Posting   // term → postings
    docLengths  map[string]int         // obsId → term count
    docMeta     map[string]string      // obsId → sessionId
    totalDocs   int
    totalLength int
}

type Posting struct {
    ObsID     string
    SessionID string
    TF        int
}

func (b *BM25Index) Add(obsID, sessionID string, terms []string) { ... }
func (b *BM25Index) Remove(obsID string) { ... }
func (b *BM25Index) Search(query string, limit int) []BM25Result { ... }
func (b *BM25Index) Serialize() ([]byte, error) { ... }    // gob encoding
func (b *BM25Index) Deserialize(data []byte) error { ... }
```

### 3.3. Vector Index (in-memory, 384-dim)

```go
// pkg/search/vector_index.go
// Dense vector index using cosine similarity
// - 384-dimension vectors (all-MiniLM-L6-v2)
// - Brute-force cosine for <50k docs (sufficient for personal memory)
// - Dimension validation (prevent corruption on provider change)

type VectorIndex struct {
    mu       sync.RWMutex
    vectors  map[string][]float32  // obsId → embedding
    sessions map[string]string     // obsId → sessionId
    dims     int                   // 384
}

func (v *VectorIndex) Add(obsID, sessionID string, vec []float32) error { ... }
func (v *VectorIndex) Remove(obsID string) { ... }
func (v *VectorIndex) Search(query []float32, limit int) []VectorResult { ... }
// cosineSimilarity: dot(a,b) / (|a|*|b|)
```

### 3.4. RRF Fusion (Reciprocal Rank Fusion)

```go
// pkg/search/rrf.go
// RRF formula: score(d) = Σ 1/(k + rank_i(d))
// where k=60 (standard), rank_i = position in each ranked list

type HybridSearch struct {
    BM25          *BM25Index
    Vector        *VectorIndex
    EmbedProvider provider.EmbeddingProvider
    BM25Weight    float64  // default 0.4
    VectorWeight  float64  // default 0.6
    GraphWeight   float64  // default 0.3
    GraphURL      string   // graph service URL
}

func (h *HybridSearch) Search(ctx context.Context, query string, limit int) ([]HybridResult, error) {
    // 1. BM25 search
    bm25Results := h.BM25.Search(query, limit*3)
    
    // 2. Vector search (if embedding provider available)
    var vectorResults []VectorResult
    if h.EmbedProvider != nil {
        vec, err := h.EmbedProvider.Embed(ctx, query)
        if err == nil {
            vectorResults = h.Vector.Search(vec, limit*3)
        }
    }
    
    // 3. Graph search (optional, HTTP call to graph service)
    var graphResults []GraphResult
    if h.GraphURL != "" {
        graphResults, _ = h.searchGraph(ctx, query, limit)
    }
    
    // 4. RRF fusion
    scores := map[string]*HybridResult{}
    const k = 60
    
    for rank, r := range bm25Results {
        id := r.ObsID
        if scores[id] == nil { scores[id] = &HybridResult{ObsID: id, SessionID: r.SessionID} }
        scores[id].CombinedScore += h.BM25Weight * (1.0 / float64(k+rank+1))
        scores[id].BM25Rank = rank + 1
        scores[id].BM25Score = r.Score
    }
    for rank, r := range vectorResults {
        id := r.ObsID
        if scores[id] == nil { scores[id] = &HybridResult{ObsID: id, SessionID: r.SessionID} }
        scores[id].CombinedScore += h.VectorWeight * (1.0 / float64(k+rank+1))
        scores[id].VectorRank = rank + 1
        scores[id].VectorScore = r.Score
    }
    // Same pattern for graph results...
    
    // Sort, return top-limit
    ...
}
```

### 3.5. Local Embedding (zero API cost)

```go
// pkg/provider/local_embed.go
// Uses ONNX runtime to run all-MiniLM-L6-v2 locally
// Model download on first run → cache in AGENTMEMORY_DATA_DIR/models/
// 384-dimension vectors
// Zero API cost, no internet required after first download

type LocalEmbeddingProvider struct {
    modelPath string
    session   *onnxruntime.Session
}

func (p *LocalEmbeddingProvider) Name() string { return "local" }
func (p *LocalEmbeddingProvider) Dimensions() int { return 384 }
func (p *LocalEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) { ... }
```

### 3.6. Context Builder (token budget)

```go
// API: POST /search/context
// Assembles context for agent injection within TOKEN_BUDGET tokens
// Priority order:
// 1. Recent high-strength memories (from memory-service)
// 2. Recent session summaries (last 3 sessions)
// 3. Relevant observations (via smart search if query provided)

type ContextBlock struct {
    Type    string  // "memory" | "summary" | "observation"
    Content string
    Tokens  int     // estimated as len(content)/4
    Recency float64 // exp(-days/7)
    Source  string  // ID reference
}
```

### 3.7. API Endpoints

```
POST /search/smart             # Hybrid BM25+Vector+Graph with RRF
POST /search/bm25              # Pure BM25 keyword search
POST /search/vector            # Pure vector similarity search
POST /search/context           # Build context block for agent (token budget)
POST /search/query-expand      # LLM-based query expansion
POST /search/rebuild-index     # Admin: full rebuild from KV
POST /index/add                # Internal: add observation to indexes
POST /index/remove             # Internal: remove observation from indexes
GET  /health                   # Health + index stats
```

### 3.8. Smart Search Request/Response

```json
// Request
{
  "query": "how did we fix the N+1 database query",
  "limit": 10,
  "project": "my-app",
  "agent_id": "claude-code",
  "bm25_weight": 0.4,
  "vector_weight": 0.6,
  "graph_weight": 0.3
}

// Response
{
  "results": [
    {
      "obs_id": "obs_abc123",
      "session_id": "sess_xyz",
      "score": 0.892,
      "bm25_score": 0.751,
      "vector_score": 0.834,
      "bm25_rank": 1,
      "vector_rank": 2,
      "observation": {
        "title": "Fixed N+1 query in UserRepository.findAll()",
        "facts": ["Used eager loading with JOIN", "Performance improved 40x"],
        "files": ["/src/repositories/user.ts"]
      }
    }
  ],
  "expansions": ["database optimization", "ORM query fix"],
  "took_ms": 11
}
```

### 3.9. Index Persistence

```go
// Index saved to disk every 30s (debounced) on write
// Loaded from disk on startup (async, server available immediately)
// Files: AGENTMEMORY_DATA_DIR/bm25.gob + vector.gob
// Dimension validation: if vector index dims != provider dims → drop and rebuild
```

### 3.10. Gateway Integration

```
POST /v1/observe/search/smart    → observe-search:8082 /search/smart
POST /v1/observe/search/context  → observe-search:8082 /search/context
POST /v1/observe/search/bm25     → observe-search:8082 /search/bm25
POST /v1/observe/search/vector   → observe-search:8082 /search/vector
```

---

## 4. Environment Variables

| Variable | Default | Mô tả |
|---|---|---|
| `SEARCH_PORT` | `8082` | Service port |
| `AGENTMEMORY_DATA_DIR` | `~/.agentmemory` | Data + index dir |
| `BM25_WEIGHT` | `0.4` | BM25 weight in RRF |
| `VECTOR_WEIGHT` | `0.6` | Vector weight in RRF |
| `GRAPH_WEIGHT` | `0.3` | Graph weight in RRF |
| `EMBEDDING_PROVIDER` | `local` | `local`, `openai`, `gemini`, `none` |
| `TOKEN_BUDGET` | `2000` | Default context token budget |
| `AGENTMEMORY_DROP_STALE_INDEX` | `false` | Drop on dimension mismatch |

---

## 5. Acceptance Criteria

- [x] `POST /search/smart` query "database performance" returns observations about "N+1 query fix" (semantic match, no keyword overlap).
- [x] p50 search latency ≤ 14ms (measured with 10,000 indexed observations).
- [x] BM25 index survives service restart (loaded from `bm25.gob`).
- [x] Vector index survives service restart (loaded from `vector.gob`).
- [x] Local embedding mode works with zero API key configured.
- [x] `POST /search/context` với `token_budget: 1000` trả về blocks tổng ≤ 1000 tokens.
- [x] Query expansion trả về 2-3 reformulations và temporal concretizations.
- [x] `POST /index/add` được gọi bởi observe-service sau mỗi observation → searchable trong < 1s.
- [x] `GET /health` trả về index stats: `{bm25.documents, vector.documents, status: "healthy"}`.
