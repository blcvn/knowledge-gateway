# Solution: SOL-ZEP-004 — Semantic Graph Search & 5 Reranking Strategies

**CR ID:** CR-ZEP-004  
**Solution ID:** SOL-ZEP-004  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp `services/search-service/` (zep domain, port gRPC 9045) để hỗ trợ **Graph-Aware Search** đa scope (edges/nodes/episodes/all) với 5 reranking strategies. Tách biệt hoàn toàn với Supermemory search service (SOL-SM-003). Redis cache TTL 30s với NATS-driven invalidation sau mỗi graph extraction.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `zep-search` service | `apps/memory/internal/bootstrap/` | Có: basic search, chưa có graph-aware |
| `search-service/` | `services/search-service/` | Có: multi-strategy search cơ bản |
| NATS events `zep.graph.*` | Architecture | Sẵn có để subscribe |
| Redis | Infrastructure | Có: dùng cho cache |

### Gap phân tích

- Chưa có graph-aware search (chỉ vector search)
- Chưa có 5 reranking strategies (chỉ có 1)
- Thiếu NodeLabel và EdgeType filtering
- Thiếu MinFactRating filter
- Chưa có Redis cache với NATS invalidation
- Thiếu `GetRelevantFacts` internal endpoint (dùng bởi Memory Service)

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Module Mới (trong search-service)

```
services/search-service/internal/
├── domain/
│   ├── graph_search.go        # GraphSearchQuery, GraphSearchResult
│   ├── reranker.go            # 5 reranker interfaces + registry
│   └── repository.go          # GraphSearchRepository port
├── usecase/
│   ├── graph_search.go        # Multi-scope search orchestrator
│   ├── get_relevant_facts.go  # Internal: cho Memory Service
│   └── session_search.go      # Search across sessions
├── adapter/
│   ├── grpc/
│   │   └── search_server.go
│   └── subscriber/
│       └── graph_events.go    # NATS: cache invalidation
└── infra/
    ├── redis/
    │   └── search_cache.go    # TTL 30s cache
    ├── neo4j/
    │   └── graph_searcher.go  # Cypher-based search
    └── reranker/
        ├── rrf.go             # Reciprocal Rank Fusion
        ├── mmr.go             # Maximal Marginal Relevance
        ├── cross_encoder.go   # Neural cross-encoder
        ├── node_distance.go   # Graph shortest path
        └── episode_mentions.go # Episode co-occurrence
```

### 3.2. Domain Model

```go
// services/search-service/internal/domain/graph_search.go

package domain

import "time"

type SearchScope string

const (
    SearchScopeEdges    SearchScope = "edges"    // temporal facts (TemporalEdge)
    SearchScopeNodes    SearchScope = "nodes"    // entity nodes (EntityNode)
    SearchScopeEpisodes SearchScope = "episodes" // source events (Episode)
    SearchScopeAll      SearchScope = "all"      // expensive: search everything
)

type RerankerType string

const (
    RerankerRRF             RerankerType = "rrf"             // Reciprocal Rank Fusion
    RerankerMMR             RerankerType = "mmr"             // Maximal Marginal Relevance
    RerankerCrossEncoder    RerankerType = "cross_encoder"   // Neural cross-encoder
    RerankerNodeDistance    RerankerType = "node_distance"   // Graph shortest path
    RerankerEpisodeMentions RerankerType = "episode_mentions" // Episode co-occurrence
)

type GraphSearchQuery struct {
    Query          string
    UserID         *string      // scope to user's graph
    GroupIDs       []string     // scope to specific groups (sessions)
    Scope          SearchScope
    Reranker       RerankerType
    NodeLabels     []string     // filter: only return these node types
    EdgeTypes      []string     // filter: only return these edge types
    Limit          int          // default: 10
    MinFactRating  *float64     // filter by quality rating (0.0-1.0)
    MmrLambda      *float64     // MMR: 0.0=diversity, 1.0=relevance (default 0.5)
    CenterNodeUUID *string      // for node_distance: reference node
}

type GraphSearchResult struct {
    UUID    string
    Score   float64
    Type    SearchScope  // "edges" | "nodes" | "episodes"

    // For edges:
    Fact    *TemporalEdgeSummary
    // For nodes:
    Node    *EntityNodeSummary
    // For episodes:
    Episode *EpisodeSummary
}

type TemporalEdgeSummary struct {
    Name      string
    Fact      string
    ValidAt   *time.Time
    InvalidAt *time.Time
    ExpiredAt *time.Time
}

type EntityNodeSummary struct {
    Name     string
    NodeType string
    Summary  string
}

type EpisodeSummary struct {
    Content   string
    Source    string
    CreatedAt time.Time
}

type GraphSearchResponse struct {
    Items     []GraphSearchResult
    Total     int
    Query     string
    Scope     SearchScope
    Reranker  RerankerType
    LatencyMs int64
}
```

### 3.3. 5 Reranker Implementations

```go
// services/search-service/internal/infra/reranker/rrf.go

// Reranker interface
type Reranker interface {
    Rerank(ctx context.Context, query string, results []GraphSearchResult) ([]GraphSearchResult, error)
}

// 1. Reciprocal Rank Fusion (RRF) — general purpose, no latency overhead
type RRFReranker struct {
    K int // fusion constant, default 60
}

func (r *RRFReranker) Rerank(ctx context.Context, query string, results []GraphSearchResult) ([]GraphSearchResult, error) {
    // RRF score = Σ 1 / (k + rank_i) for each ranking list
    // Fuses: vector similarity rank + BM25 text rank + fact_rating rank
    scores := make(map[string]float64)
    for _, list := range buildRankingLists(results) {
        for rank, item := range list {
            scores[item.UUID] += 1.0 / float64(r.K + rank + 1)
        }
    }
    return sortByScore(results, scores), nil
}

// 2. Maximal Marginal Relevance (MMR) — diverse results
type MMRReranker struct {
    DefaultLambda float64 // 0.5: balanced diversity vs relevance
}

func (r *MMRReranker) Rerank(ctx context.Context, query string, results []GraphSearchResult, lambda ...float64) ([]GraphSearchResult, error) {
    l := r.DefaultLambda
    if len(lambda) > 0 { l = lambda[0] }
    // MMR = argmax[λ*sim(d,q) - (1-λ)*max_sim(d,S)]
    // Where S = already selected results set
    selected := make([]GraphSearchResult, 0, len(results))
    remaining := make([]GraphSearchResult, len(results))
    copy(remaining, results)

    for len(selected) < len(results) && len(remaining) > 0 {
        best := selectMMRBest(remaining, selected, query, l)
        selected = append(selected, best)
        remaining = removeItem(remaining, best.UUID)
    }
    return selected, nil
}

// 3. Cross-Encoder — best accuracy, highest latency (~200ms)
type CrossEncoderReranker struct {
    ModelURL  string  // external cross-encoder service URL
    BatchSize int     // default 32
}

func (r *CrossEncoderReranker) Rerank(ctx context.Context, query string, results []GraphSearchResult) ([]GraphSearchResult, error) {
    // Send (query, document) pairs to cross-encoder model
    // Returns relevance scores from neural model
    pairs := makePairs(query, results)
    scores, err := r.callCrossEncoderAPI(ctx, pairs)
    if err != nil {
        // Fallback to RRF on cross-encoder failure
        return (&RRFReranker{K: 60}).Rerank(ctx, query, results)
    }
    return applyScores(results, scores), nil
}

// 4. Node Distance — graph-aware, traverses Neo4j
type NodeDistanceReranker struct {
    MaxDepth int  // max graph traversal depth (default 3)
    neo4j    Neo4jClient
}

func (r *NodeDistanceReranker) Rerank(ctx context.Context, query string, results []GraphSearchResult) ([]GraphSearchResult, error) {
    if r.centerNodeUUID == "" { return results, nil }
    // Compute shortest path distance from centerNode to each result node
    // Closer nodes → higher score
    distances := r.neo4j.ShortestPaths(ctx, r.centerNodeUUID, extractNodeUUIDs(results), r.MaxDepth)
    return scoreByDistance(results, distances), nil
}

// 5. Episode Mentions — recency-aware
type EpisodeMentionsReranker struct {
    TimeDecay float64 // exponential decay factor (default 0.95)
}

func (r *EpisodeMentionsReranker) Rerank(ctx context.Context, query string, results []GraphSearchResult) ([]GraphSearchResult, error) {
    // Score = Σ decay^days_ago for each episode mentioning this fact
    // Recent episodes → higher score
    for i := range results {
        results[i].Score = computeEpisodeScore(results[i], r.TimeDecay)
    }
    return sortByScore(results, nil), nil
}
```

### 3.4. Multi-Scope Search Orchestrator

```go
// services/search-service/internal/usecase/graph_search.go

type GraphSearchUseCase struct {
    graphSearcher  GraphSearcher    // Neo4j-backed
    embedder       EmbedderPort     // vector embedding
    rerankers      map[RerankerType]Reranker
    cache          SearchCache      // Redis TTL 30s
}

func (uc *GraphSearchUseCase) Execute(ctx context.Context, query GraphSearchQuery) (*GraphSearchResponse, error) {
    start := time.Now()

    // 1. Check Redis cache
    cacheKey := buildCacheKey(query)
    if cached, err := uc.cache.Get(ctx, cacheKey); err == nil {
        return cached, nil
    }

    // 2. Generate query embedding
    embedding, err := uc.embedder.Embed(ctx, query.Query)
    if err != nil { return nil, err }

    // 3. Execute scoped search in Neo4j
    var rawResults []GraphSearchResult

    switch query.Scope {
    case SearchScopeEdges:
        rawResults, err = uc.graphSearcher.SearchEdges(ctx, EdgeSearchParams{
            GroupIDs:      resolveGroupIDs(query),
            Embedding:     embedding,
            NodeLabels:    query.NodeLabels,
            EdgeTypes:     query.EdgeTypes,
            MinFactRating: query.MinFactRating,
            Limit:         query.Limit * 3, // over-fetch for reranking
        })
    case SearchScopeNodes:
        rawResults, err = uc.graphSearcher.SearchNodes(ctx, NodeSearchParams{
            GroupIDs:   resolveGroupIDs(query),
            Embedding:  embedding,
            NodeLabels: query.NodeLabels,
            Limit:      query.Limit * 3,
        })
    case SearchScopeEpisodes:
        rawResults, err = uc.graphSearcher.SearchEpisodes(ctx, EpisodeSearchParams{
            GroupIDs:  resolveGroupIDs(query),
            Embedding: embedding,
            Limit:     query.Limit * 3,
        })
    case SearchScopeAll:
        // Expensive: parallel search across all scopes
        rawResults, err = uc.searchAll(ctx, query, embedding)
    }

    if err != nil { return nil, err }

    // 4. Apply reranking strategy
    reranker := uc.rerankers[query.Reranker]
    if reranker == nil { reranker = uc.rerankers[RerankerRRF] } // default
    ranked, err := reranker.Rerank(ctx, query.Query, rawResults)
    if err != nil { return nil, err }

    // 5. Apply limit
    if len(ranked) > query.Limit { ranked = ranked[:query.Limit] }

    resp := &GraphSearchResponse{
        Items:     ranked,
        Total:     len(ranked),
        Query:     query.Query,
        Scope:     query.Scope,
        Reranker:  query.Reranker,
        LatencyMs: time.Since(start).Milliseconds(),
    }

    // 6. Cache result (TTL 30s)
    uc.cache.Set(ctx, cacheKey, resp, 30*time.Second)

    return resp, nil
}
```

### 3.5. Redis Cache with NATS Invalidation

```go
// services/search-service/internal/adapter/subscriber/graph_events.go

type GraphEventSubscriber struct {
    nats  NATSClient
    cache SearchCache  // Redis
}

func (s *GraphEventSubscriber) Start(ctx context.Context) {
    // Invalidate cache when graph extraction completes
    s.nats.Subscribe(ctx, "zep.graph.extraction.completed", func(event GraphExtractionCompletedEvent) {
        // Invalidate all cache entries for this group
        s.cache.DeleteByPattern(ctx, fmt.Sprintf("search:*:%s:*", event.GroupID))
    })

    // Invalidate specific fact from cache on invalidation
    s.nats.Subscribe(ctx, "zep.graph.fact.invalidated", func(event FactInvalidatedEvent) {
        s.cache.DeleteByPattern(ctx, fmt.Sprintf("search:*:%s:*", event.GroupID))
    })
}

// Cache key construction
func buildCacheKey(q GraphSearchQuery) string {
    // Deterministic key from query + scope + reranker + filters
    h := sha256.New()
    fmt.Fprintf(h, "%s:%s:%s:%v:%v:%v",
        q.Query, q.Scope, q.Reranker,
        q.NodeLabels, q.EdgeTypes, q.MinFactRating,
    )
    return fmt.Sprintf("search:%x", h.Sum(nil)[:16])
}
```

### 3.6. GetRelevantFacts (Internal — cho Memory Service)

```go
// services/search-service/internal/usecase/get_relevant_facts.go

// GetRelevantFacts is an internal gRPC-only endpoint used by Memory Service
// to assemble context in GetMemory (sub-100ms target)
type GetRelevantFactsUseCase struct {
    graphSearcher GraphSearcher
    embedder      EmbedderPort
    cache         SearchCache  // Redis
}

func (uc *GetRelevantFactsUseCase) Execute(ctx context.Context, req GetRelevantFactsRequest) ([]Fact, error) {
    // Build query from last N messages content
    queryText := buildQueryFromMessages(req.Messages)
    embedding, _ := uc.embedder.Embed(ctx, queryText)

    // Search edges only (facts), scoped to groupID
    rawFacts, err := uc.graphSearcher.SearchEdges(ctx, EdgeSearchParams{
        GroupIDs:  []string{req.GroupID},
        Embedding: embedding,
        Limit:     req.MaxFacts * 2,
    })
    if err != nil {
        return []Fact{}, nil // graceful degradation (empty, not error)
    }

    // Apply RRF reranking for GetRelevantFacts (always fast)
    reranked, _ := (&RRFReranker{K: 60}).Rerank(ctx, queryText, rawFacts)

    // Return top MaxFacts
    if len(reranked) > req.MaxFacts { reranked = reranked[:req.MaxFacts] }
    return convertToFacts(reranked), nil
}
```

---

## 4. Neo4j Vector Search (pgvector Alternative)

```cypher
-- Neo4j 5.22+ has native vector index (replaces pgvector for graph data)
CREATE VECTOR INDEX edge_embedding_idx IF NOT EXISTS
    FOR ()-[r:*]-() ON (r.embedding)
    OPTIONS {indexConfig: {`vector.dimensions`: 1536, `vector.similarity_function`: 'cosine'}};

CREATE VECTOR INDEX node_embedding_idx IF NOT EXISTS
    FOR (n:Entity) ON (n.embedding)
    OPTIONS {indexConfig: {`vector.dimensions`: 1536, `vector.similarity_function`: 'cosine'}};

-- Semantic search on edges:
CALL db.index.vector.queryRelationships('edge_embedding_idx', $limit * 3, $embedding)
YIELD relationship AS r, score
WHERE r.group_id IN $group_ids
  AND r.invalid_at IS NULL    -- exclude invalidated facts
  AND (r.fact_rating IS NULL OR r.fact_rating >= $min_fact_rating)
  AND ($edge_types IS NULL OR type(r) IN $edge_types)
RETURN r, score
ORDER BY score DESC
LIMIT $limit
```

---

## 5. Reranker Configuration

```yaml
# apps/memory/configs/config.yaml — thêm section
zep:
  search:
    reranker:
      default: "rrf"
      rrf:
        k: 60
      mmr:
        default_lambda: 0.5
      cross_encoder:
        model_url: "http://cross-encoder-service:8200"
        batch_size: 32
        timeout_ms: 500
      node_distance:
        max_depth: 3
      episode_mentions:
        time_decay: 0.95
    cache:
      ttl_seconds: 30
```

---

## 6. API Endpoints

```
POST   /api/v2/graph/search      → GraphSearch (multi-scope, 5 rerankers)
POST   /api/v2/sessions/search   → SessionSearch (search across sessions)

# Internal gRPC (not REST):
# GetRelevantFacts — used by Memory Service for context assembly
```

---

## 7. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model (SearchScope, RerankerType, GraphSearchQuery) | 1 ngày |
| **P2** | RRF + MMR rerankers (pure Go, no external deps) | 1 ngày |
| **P3** | Neo4j graph searcher (Cypher vector + filter queries) | 2 ngày |
| **P4** | Cross-Encoder reranker (HTTP external service) | 1 ngày |
| **P5** | Node Distance reranker (Neo4j shortest path) | 1 ngày |
| **P6** | Episode Mentions reranker (time decay) | 1 ngày |
| **P7** | GraphSearch orchestrator (multi-scope) | 1.5 ngày |
| **P8** | Redis cache + NATS invalidation subscriber | 1 ngày |
| **P9** | GetRelevantFacts internal endpoint | 1 ngày |
| **P10** | Gateway integration + tests | 1.5 ngày |

**Tổng:** ~12 ngày (Wave 4 — song song với SOL-ZEP-003)

---

## 8. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| scope=edges → chỉ facts, không có nodes | SearchScopeEdges → SearchEdges() only |
| reranker=mmr + 20 results → diverse hơn rrf | MMR với lambda=0.5, cosine similarity dedup |
| node_labels=["Organization"] → chỉ Organization | NodeLabels filter trong Cypher WHERE |
| min_fact_rating=0.8 → high quality only | `WHERE r.fact_rating >= $min_fact_rating` |
| Cache hit → latency < 10ms | Redis GET ~1-5ms |
| Sau graph extraction → cache invalidate | NATS subscriber + DeleteByPattern |
| Session search → facts grouped by session_id | GroupIDs = [sessionIDs], result includes session_id |
