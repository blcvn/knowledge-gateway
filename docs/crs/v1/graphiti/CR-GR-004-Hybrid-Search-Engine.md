# Change Request: CR-GR-004 — Hybrid Search Engine (Multi-Strategy + Temporal Filtering)

**CR ID:** CR-GR-004  
**Component:** `services/graphiti-search` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** graphiti PRD §5.2, SRS §3.6, specs/services/03-search-service.md  
**Maps to Python:** `graphiti_core/search/` — `search.py`, `search_config.py`, `search_config_recipes.py`

---

## 1. Mô tả

Xây dựng **graphiti-search** service — hybrid search engine kết hợp 3 search streams:
1. **Cosine Similarity** — vector embedding search.
2. **BM25 Fulltext** — keyword-based search.
3. **BFS Graph Traversal** — breadth-first graph traversal.

Với 5 reranking strategies: **RRF**, **MMR**, **Cross-Encoder**, **Node Distance**, **Episode Mentions**.

Target performance: **< 1 second** end-to-end hybrid search latency.

---

## 2. Vấn đề hiện tại

`services/cognee-search` / `services/search-service` hiện tại:
- ✅ Có vector similarity search (qua Qdrant).
- ❌ Không có **BFS graph traversal** search.
- ❌ Không có **temporal filters** (valid_at, invalid_at, expired_at, created_at ranges).
- ❌ Không có **MMR** (Maximal Marginal Relevance) reranking.
- ❌ Không có **cross-encoder reranking** (neural scoring).
- ❌ Không có **node-distance reranking** (graph proximity boost).
- ❌ Không có **episode-mentions reranking** (mention frequency boost).
- ❌ Không có **search result caching** với Redis + NATS invalidation.
- ❌ Không có **pre-built search recipes** (pre-configured SearchConfig).
- ❌ Search theo cả 4 object types: edges, nodes, episodes, communities.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/graphiti-search/`

**Port:** `9002` (gRPC internal)

### 3.2. Search Pipeline (6 Steps)

```go
// internal/usecase/search_edges.go

func (uc *SearchEdgesUseCase) Execute(ctx, req SearchRequest) (*SearchResults, error) {
    // [1] Generate Query Embedding (→ Knowledge)
    embedding, _ := knowledgeClient.GenerateEmbedding(ctx, req.Query)

    // [2] Parallel Search Dispatch (→ Store)
    var (
        cosineResults []ScoredEdge
        bm25Results   []ScoredEdge
        bfsResults    []ScoredEdge
    )
    // fan-out via goroutines
    g, ctx := errgroup.WithContext(ctx)
    if config.HasMethod(CosineSimilarity) {
        g.Go(func() { cosineResults, _ = storeClient.EdgeSimilaritySearch(ctx, embedding, ...) })
    }
    if config.HasMethod(BM25) {
        g.Go(func() { bm25Results, _ = storeClient.EdgeFulltextSearch(ctx, req.Query, ...) })
    }
    if config.HasMethod(BFS) {
        g.Go(func() { bfsResults, _ = storeClient.EdgeBFSSearch(ctx, matchedNodeUUIDs, ...) })
    }
    g.Wait()

    // [3] Merge Results (deduplicate by UUID, preserve per-method scores)
    merged := mergeResults(cosineResults, bm25Results, bfsResults)

    // [4] Rerank
    switch config.EdgeConfig.Reranker {
    case RRF:         reranked = RRFRerank(resultSets, k=60)
    case MMR:         reranked = MMRRerank(embedding, candidates, lambda=0.5, limit)
    case CrossEncoder: reranked, _ = CrossEncoderRerank(ctx, knowledgeClient, req.Query, candidates)
    case NodeDistance: reranked, _ = NodeDistanceRerank(ctx, storeClient, candidates, req.CenterNodeUUID)
    case EpisodeMentions: reranked, _ = EpisodeMentionsRerank(ctx, storeClient, candidates)
    }

    // [5] Apply Temporal + Property Filters
    filtered = applyFilters(reranked, req.Filters)

    // [6] Return SearchResults (edges, nodes, episodes, communities)
    return &SearchResults{Edges: filtered}, nil
}
```

### 3.3. Search Configurations

```go
type SearchConfig struct {
    EdgeConfig      *EdgeSearchConfig
    NodeConfig      *NodeSearchConfig
    EpisodeConfig   *EpisodeSearchConfig
    CommunityConfig *CommunitySearchConfig
    Limit           int     // default: 10
    RerankerMinScore float64
}

type EdgeSearchConfig struct {
    Methods     []SearchMethod  // cosine_similarity | bm25 | bfs
    Reranker    Reranker       // rrf | mmr | cross_encoder | node_distance | episode_mentions
    SimMinScore float64        // default: 0.5
    MMRLambda   float64        // default: 0.5 (balance relevance/diversity)
    BFSMaxDepth int            // default: 2
}
```

### 3.4. Pre-Built Search Recipes

```go
// internal/domain/recipe.go (compile-time constants)

// EdgeHybridSearchRRF — Default: most common use case
var EdgeHybridSearchRRF = SearchConfig{
    EdgeConfig: &EdgeSearchConfig{
        Methods:     []SearchMethod{BM25, CosineSimilarity},
        Reranker:    RRF,
        SimMinScore: 0.5,
    },
    Limit: 10,
}

// EdgeHybridSearchMMR — Diverse results (avoid redundancy)
var EdgeHybridSearchMMR = SearchConfig{
    EdgeConfig: &EdgeSearchConfig{
        Methods:  []SearchMethod{BM25, CosineSimilarity},
        Reranker: MMR,
        MMRLambda: 0.5,
    },
    Limit: 10,
}

// EdgeHybridSearchCrossEncoder — High-accuracy with neural reranking
var EdgeHybridSearchCrossEncoder = SearchConfig{
    EdgeConfig: &EdgeSearchConfig{
        Methods:     []SearchMethod{BM25, CosineSimilarity, BFS},
        Reranker:    CrossEncoder,
        BFSMaxDepth: 2,
    },
    Limit: 10,
}

// NodeHybridSearchRRF — Node-focused search
var NodeHybridSearchRRF = SearchConfig{
    NodeConfig: &NodeSearchConfig{
        Methods:  []SearchMethod{BM25, CosineSimilarity},
        Reranker: RRF,
    },
    Limit: 10,
}

// CommunityHybridSearchRRF — Topic-level community search
var CommunityHybridSearchRRF = SearchConfig{
    CommunityConfig: &CommunitySearchConfig{
        Methods:  []SearchMethod{BM25, CosineSimilarity},
        Reranker: RRF,
    },
    Limit: 10,
}

// CombinedHybridSearchCrossEncoder — All 4 types simultaneously
var CombinedHybridSearchCrossEncoder = SearchConfig{
    EdgeConfig:      &EdgeSearchConfig{...},
    NodeConfig:      &NodeSearchConfig{...},
    EpisodeConfig:   &EpisodeSearchConfig{...},
    CommunityConfig: &CommunitySearchConfig{...},
    Limit: 10,
}
```

### 3.5. Reranking Implementations

```go
// RRF: Reciprocal Rank Fusion
// score(d) = Σ 1/(k + rank_i(d)) for each search method i
// k = 60 (standard constant)
func RRFRerank(resultSets [][]ScoredItem, k int) []ScoredItem

// MMR: Maximal Marginal Relevance
// score(d) = λ·sim(query,d) - (1-λ)·max(sim(d, selected_docs))
// Balances relevance with diversity
func MMRRerank(query []float32, candidates []ScoredItem, lambda float64, limit int) []ScoredItem

// Cross-Encoder: Neural scoring via Knowledge Service
// Delegates to knowledge.Rerank RPC
func CrossEncoderRerank(ctx, knowledgeClient, query string, candidates) ([]ScoredItem, error)

// Node Distance: Boost items closer (in graph hops) to center node
func NodeDistanceRerank(ctx, storeClient, candidates []ScoredItem, centerNodeUUID string) ([]ScoredItem, error)

// Episode Mentions: Boost items mentioned in more recent/frequent episodes
func EpisodeMentionsRerank(ctx, storeClient, candidates []ScoredItem) ([]ScoredItem, error)
```

### 3.6. Temporal Filters

```go
// internal/domain/filter.go

type SearchFilters struct {
    // Temporal window filters (EntityEdge.valid_at / invalid_at / created_at)
    CreatedAtStart *time.Time
    CreatedAtEnd   *time.Time
    ValidAt        *time.Time    // point-in-time query: fact valid at this time
    InvalidAt      *time.Time    // include invalidated edges up to this time

    // Identity filters
    NodeUUIDs      []string
    EdgeUUIDs      []string

    // Type filters
    EntityLabels   []string  // e.g. ["Person", "Company"]

    // Partition
    GroupIDs       []string
}

// Temporal query example:
// SearchFilters{ValidAt: &refTime} →
//   WHERE e.valid_at <= refTime AND (e.invalid_at IS NULL OR e.invalid_at > refTime)
```

### 3.7. Search Result Caching (Redis)

```go
// internal/adapter/cache/redis_cache.go

type CacheKey struct {
    Query    string
    GroupIDs []string
    Config   SearchConfig
    Filters  SearchFilters
}
// hash(CacheKey) → Redis key

// Cache TTL: 5 minutes (configurable)
// Invalidation events:
//   NATS: graphiti.episode.ingested → invalidate group_id caches
//   NATS: graphiti.entity.resolved  → invalidate affected entity caches
//   NATS: graphiti.community.rebuilt → invalidate community search caches
```

### 3.8. gRPC API

```protobuf
service SearchService {
    // Default hybrid edge search (RRF) — maps to Python graphiti.search()
    rpc Search(SearchRequest) returns (SearchResponse);

    // Configurable multi-type search — maps to Python graphiti.search_()
    rpc SearchAdvanced(SearchAdvancedRequest) returns (SearchAdvancedResponse);

    // Type-specific searches
    rpc SearchNodes(SearchNodesRequest) returns (SearchNodesResponse);
    rpc SearchEdges(SearchEdgesRequest) returns (SearchEdgesResponse);
    rpc SearchEpisodes(SearchEpisodesRequest) returns (SearchEpisodesResponse);
    rpc SearchCommunities(SearchCommunitiesRequest) returns (SearchCommunitiesResponse);
}
```

### 3.9. REST API (via Gateway)

```
POST /v1/search
  Body: {
    query: "who joined engineering",
    group_ids: ["project-alpha"],
    num_results: 10,
    center_node_uuid: "...",        // optional, for node_distance reranking
    search_config: { ... }           // optional custom config
  }
  Response: SearchResults{edges, nodes, episodes, communities}

GET /v1/entities/{uuid}             # retrieve EntityNode
GET /v1/edges/{uuid}                # retrieve EntityEdge
```

---

## 4. Configuration

```yaml
server:
  grpc_port: 9002
  max_concurrent_searches: 100

search:
  default_limit: 10
  max_limit: 100
  rrf_k: 60
  mmr_lambda: 0.5
  bfs_max_depth: 3
  sim_min_score: 0.5
  timeout: 30s

cache:
  enabled: true
  redis_url: "redis://redis:6379/1"
  default_ttl: 300s
  max_entries: 10000

services:
  knowledge: { address: "graphiti-knowledge:9003", timeout: 60s }
  store:     { address: "graphiti-store:9004", timeout: 15s }

events:
  nats_url: "nats://nats:4222"
  invalidation_subjects:
    - "graphiti.episode.ingested"
    - "graphiti.entity.resolved"
    - "graphiti.community.rebuilt"
```

---

## 5. Metrics

| Metric | Type | Labels |
|---|---|---|
| `search_queries_total` | Counter | config, status |
| `search_duration_seconds` | Histogram | config, method |
| `search_results_count` | Histogram | config, type |
| `search_cache_hits_total` | Counter | — |
| `search_rerank_duration_seconds` | Histogram | reranker |

---

## 6. Acceptance Criteria

- [ ] `POST /v1/search` query "who joined engineering" → trả về EntityEdge với fact "Alice joined engineering" (semantic match).
- [ ] Search latency < 1000ms (BM25 + cosine + RRF) với 10,000 nodes indexed.
- [ ] `SearchAdvanced` với `CombinedHybridSearchCrossEncoder` → trả về đồng thời `edges`, `nodes`, `episodes`, `communities`.
- [ ] Temporal filter `valid_at = "2026-03-01"` → chỉ trả về edges valid vào tháng 3, không trả về edges đã bị invalidated trước thời điểm đó.
- [ ] `SearchAdvanced` với `entity_labels: ["Person"]` → chỉ trả về EntityNodes có label "Person".
- [ ] Cache hit: cùng query gọi 2 lần → 2nd response time < 5ms (Redis cache).
- [ ] Cache invalidation: sau ingest episode mới → next search query bỏ qua cache (stale result bị invalidated).
- [ ] MMR reranking với `lambda=0.5` → kết quả diverse (không có 2 edges quá similar với nhau).
- [ ] Node Distance reranking với `center_node_uuid=Alice.uuid` → nodes gần Alice được boost lên đầu.
- [ ] `SearchNodes` với `NodeHybridSearchRRF` → trả về EntityNodes sorted by combined BM25+cosine score.
