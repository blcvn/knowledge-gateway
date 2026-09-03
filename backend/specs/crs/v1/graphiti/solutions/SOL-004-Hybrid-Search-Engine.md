# Solution: SOL-004 — Hybrid Search Engine (Multi-Strategy + Temporal Filtering)

**CR ID:** CR-GR-004  
**Solution ID:** SOL-004  
**Priority:** Critical (Wave 2)  
**Architecture:** REBUILD `services/graphiti-search/` — Hybrid retrieval với 3 streams + 5 rerankers

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- `graphiti-search` đã trong monolith (service #5).
- **Redis** đã configured — dùng cho search result caching.
- **NATS** embedded — subscribe `graphiti.episode.ingested` để invalidate cache.
- `graphiti-store` cung cấp `NodeBFSSearch`, `EdgeSimilaritySearch`, `EdgeFulltextSearch`.
- `graphiti-knowledge` cung cấp `GenerateEmbedding`, `Rerank`.

---

## 2. Search Domain — `internal/domain/`

### 2.1. Search Config

```go
// services/graphiti-search/internal/domain/search_config.go

type SearchMethod string
const (
    MethodCosineSimilarity SearchMethod = "COSINE_SIMILARITY"
    MethodBM25             SearchMethod = "BM25"
    MethodBFS              SearchMethod = "BFS"
)

type Reranker string
const (
    RerankerRRF           Reranker = "RRF"
    RerankerMMR           Reranker = "MMR"
    RerankerCrossEncoder  Reranker = "CROSS_ENCODER"
    RerankerNodeDistance  Reranker = "NODE_DISTANCE"
    RerankerEpisodeMentions Reranker = "EPISODE_MENTIONS"
    RerankerNone          Reranker = "NONE"
)

type SearchConfig struct {
    EdgeConfig      *EdgeSearchConfig
    NodeConfig      *NodeSearchConfig
    EpisodeConfig   *EpisodeSearchConfig
    CommunityConfig *CommunitySearchConfig
    Limit           int
    RerankerMinScore float64
}

type EdgeSearchConfig struct {
    Methods     []SearchMethod
    Reranker    Reranker
    SimMinScore float64   // default: 0.5
    MMRLambda   float64   // default: 0.5
    BFSMaxDepth int       // default: 2
}

type NodeSearchConfig struct {
    Methods     []SearchMethod
    Reranker    Reranker
    SimMinScore float64
}

type EpisodeSearchConfig struct {
    Methods     []SearchMethod
    Reranker    Reranker
}

type CommunitySearchConfig struct {
    Methods     []SearchMethod
    Reranker    Reranker
    SimMinScore float64
}

// SearchFilters — temporal + identity filters
type SearchFilters struct {
    CreatedAtStart *time.Time
    CreatedAtEnd   *time.Time
    ValidAt        *time.Time   // point-in-time: fact valid at this time
    InvalidAt      *time.Time   // include invalidated edges up to this time
    NodeUUIDs      []string
    EdgeUUIDs      []string
    EntityLabels   []string
    GroupIDs       []string
}
```

### 2.2. Pre-Built Search Recipes

```go
// services/graphiti-search/internal/domain/recipes.go

// EdgeHybridSearchRRF — Default: BM25 + cosine + RRF (fastest, good quality)
var EdgeHybridSearchRRF = SearchConfig{
    EdgeConfig: &EdgeSearchConfig{
        Methods:     []SearchMethod{MethodBM25, MethodCosineSimilarity},
        Reranker:    RerankerRRF,
        SimMinScore: 0.5,
    },
    Limit: 10,
}

// EdgeHybridSearchMMR — Diverse results, avoids redundancy
var EdgeHybridSearchMMR = SearchConfig{
    EdgeConfig: &EdgeSearchConfig{
        Methods:   []SearchMethod{MethodBM25, MethodCosineSimilarity},
        Reranker:  RerankerMMR,
        MMRLambda: 0.5,
    },
    Limit: 10,
}

// EdgeHybridSearchCrossEncoder — High accuracy with neural reranking
var EdgeHybridSearchCrossEncoder = SearchConfig{
    EdgeConfig: &EdgeSearchConfig{
        Methods:     []SearchMethod{MethodBM25, MethodCosineSimilarity, MethodBFS},
        Reranker:    RerankerCrossEncoder,
        BFSMaxDepth: 2,
        SimMinScore: 0.4,
    },
    Limit: 10,
}

// NodeHybridSearchRRF — Node-focused
var NodeHybridSearchRRF = SearchConfig{
    NodeConfig: &NodeSearchConfig{
        Methods:  []SearchMethod{MethodBM25, MethodCosineSimilarity},
        Reranker: RerankerRRF,
    },
    Limit: 10,
}

// CommunityHybridSearchRRF — Topic-level search
var CommunityHybridSearchRRF = SearchConfig{
    CommunityConfig: &CommunitySearchConfig{
        Methods:  []SearchMethod{MethodBM25, MethodCosineSimilarity},
        Reranker: RerankerRRF,
    },
    Limit: 10,
}

// CombinedHybridSearchCrossEncoder — All 4 types simultaneously
var CombinedHybridSearchCrossEncoder = SearchConfig{
    EdgeConfig:      &EdgeSearchConfig{Methods: []SearchMethod{MethodBM25, MethodCosineSimilarity}, Reranker: RerankerCrossEncoder},
    NodeConfig:      &NodeSearchConfig{Methods: []SearchMethod{MethodCosineSimilarity}, Reranker: RerankerRRF},
    EpisodeConfig:   &EpisodeSearchConfig{Methods: []SearchMethod{MethodBM25}, Reranker: RerankerNone},
    CommunityConfig: &CommunitySearchConfig{Methods: []SearchMethod{MethodCosineSimilarity}, Reranker: RerankerRRF},
    Limit: 10,
}

// RecipeByName maps string name to SearchConfig (for REST/gRPC API)
var RecipeByName = map[string]SearchConfig{
    "edge_hybrid_rrf":             EdgeHybridSearchRRF,
    "edge_hybrid_mmr":             EdgeHybridSearchMMR,
    "edge_hybrid_cross_encoder":   EdgeHybridSearchCrossEncoder,
    "node_hybrid_rrf":             NodeHybridSearchRRF,
    "community_hybrid_rrf":        CommunityHybridSearchRRF,
    "combined_cross_encoder":      CombinedHybridSearchCrossEncoder,
}
```

---

## 3. 6-Step Search Pipeline — `internal/usecase/search_edges.go`

```go
// services/graphiti-search/internal/usecase/search_edges.go

type SearchEdgesUseCase struct {
    storeClient     port.StorePort
    knowledgeClient port.KnowledgePort
    cache           port.SearchCache
    centerNodeUUID  string  // for node_distance reranking (from request)
}

type ScoredEdge struct {
    Edge   graph.EntityEdge
    Score  float64
    Source string  // "cosine" | "bm25" | "bfs"
}

func (uc *SearchEdgesUseCase) Execute(ctx context.Context, req SearchRequest) (*SearchResults, error) {
    cfg := req.Config.EdgeConfig
    if cfg == nil { return &SearchResults{}, nil }

    // ─── Step 1: Generate Query Embedding ──────────────────────────────────
    queryEmb, err := uc.knowledgeClient.GenerateEmbedding(ctx, req.Query)
    if err != nil { queryEmb = nil }  // graceful: disable cosine if embedding fails

    // ─── Step 2: Parallel Search Dispatch ──────────────────────────────────
    type searchResult struct {
        edges []ScoredEdge
        method SearchMethod
    }
    resultCh := make(chan searchResult, 3)
    var wg sync.WaitGroup

    for _, method := range cfg.Methods {
        wg.Add(1)
        go func(m SearchMethod) {
            defer wg.Done()
            var edges []ScoredEdge
            switch m {
            case MethodCosineSimilarity:
                if queryEmb != nil {
                    raw, _ := uc.storeClient.EdgeSimilaritySearch(ctx, port.EdgeSimilarityReq{
                        Vector:   queryEmb,
                        GroupIDs: req.Filters.GroupIDs,
                        Limit:    req.Config.Limit * 3,
                        MinScore: cfg.SimMinScore,
                        Filters:  mapToStoreFilters(req.Filters),
                    })
                    for _, e := range raw {
                        edges = append(edges, ScoredEdge{Edge: *e, Source: "cosine", Score: 1.0})
                    }
                }
            case MethodBM25:
                raw, _ := uc.storeClient.EdgeFulltextSearch(ctx, port.EdgeFulltextReq{
                    Query:    req.Query,
                    GroupIDs: req.Filters.GroupIDs,
                    Limit:    req.Config.Limit * 3,
                    Filters:  mapToStoreFilters(req.Filters),
                })
                for _, e := range raw {
                    edges = append(edges, ScoredEdge{Edge: *e, Source: "bm25", Score: 1.0})
                }
            case MethodBFS:
                // First: find matching nodes by name/similarity, then BFS from them
                originNodes, _ := uc.storeClient.NodeFulltextSearch(ctx, req.Query, req.Filters.GroupIDs, 5, nil)
                if len(originNodes) > 0 {
                    originUUIDs := extractNodeUUIDs(originNodes)
                    raw, _ := uc.storeClient.EdgeBFSSearch(ctx, originUUIDs, cfg.BFSMaxDepth, req.Filters.GroupIDs, req.Config.Limit*2)
                    for _, e := range raw {
                        edges = append(edges, ScoredEdge{Edge: *e, Source: "bfs", Score: 0.8})
                    }
                }
            }
            resultCh <- searchResult{edges: edges, method: m}
        }(method)
    }

    go func() { wg.Wait(); close(resultCh) }()

    // Collect and merge (dedup by UUID)
    resultSets := make(map[SearchMethod][]ScoredEdge)
    merged := make(map[string]ScoredEdge)
    for result := range resultCh {
        resultSets[result.method] = result.edges
        for _, e := range result.edges {
            if _, exists := merged[e.Edge.UUID]; !exists {
                merged[e.Edge.UUID] = e
            }
        }
    }
    candidates := mapToSlice(merged)

    // ─── Step 3: Reranking ─────────────────────────────────────────────────
    var reranked []ScoredEdge
    switch cfg.Reranker {
    case RerankerRRF:
        reranked = RRFRerank(resultSets, 60)
    case RerankerMMR:
        reranked = MMRRerank(queryEmb, candidates, cfg.MMRLambda, req.Config.Limit)
    case RerankerCrossEncoder:
        passages := extractFacts(candidates)
        scores, err := uc.knowledgeClient.Rerank(ctx, req.Query, passages)
        if err != nil { reranked = candidates } else { reranked = applyScores(candidates, scores) }
    case RerankerNodeDistance:
        if req.CenterNodeUUID != "" {
            distances, _ := uc.storeClient.NodeDistanceReranker(ctx, extractEdgeNodeUUIDs(candidates), req.CenterNodeUUID)
            reranked = applyDistanceScores(candidates, distances)
        } else { reranked = candidates }
    case RerankerEpisodeMentions:
        mentions, _ := uc.storeClient.EpisodeMentionsReranker(ctx, extractEdgeNodeUUIDs(candidates))
        reranked = applyMentionScores(candidates, mentions)
    default:
        reranked = candidates
    }

    // ─── Step 4: Apply Temporal + Property Filters ─────────────────────────
    filtered := applyTemporalFilters(reranked, req.Filters)

    // ─── Step 5: Limit ─────────────────────────────────────────────────────
    if len(filtered) > req.Config.Limit { filtered = filtered[:req.Config.Limit] }

    return &SearchResults{Edges: toEdgeSlice(filtered)}, nil
}
```

---

## 4. Reranking Implementations — `internal/usecase/rerank/`

### 4.1. RRF (Reciprocal Rank Fusion)

```go
// internal/usecase/rerank/rrf.go

// RRFRerank implements Reciprocal Rank Fusion
// score(d) = Σ 1/(k + rank_i(d)) for each method i
// k=60 is the standard constant (Cormack et al., 2009)
func RRFRerank(resultSets map[SearchMethod][]ScoredEdge, k int) []ScoredEdge {
    scores := make(map[string]float64)
    edges := make(map[string]ScoredEdge)

    for _, results := range resultSets {
        for rank, item := range results {
            scores[item.Edge.UUID] += 1.0 / float64(k+rank+1)
            if _, exists := edges[item.Edge.UUID]; !exists {
                edges[item.Edge.UUID] = item
            }
        }
    }

    // Build sorted result
    items := make([]ScoredEdge, 0, len(edges))
    for uuid, edge := range edges {
        edge.Score = scores[uuid]
        items = append(items, edge)
    }
    sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
    return items
}
```

### 4.2. MMR (Maximal Marginal Relevance)

```go
// internal/usecase/rerank/mmr.go

// MMRRerank implements Maximal Marginal Relevance
// Balances relevance with diversity: λ·sim(q,d) - (1-λ)·max(sim(d, selected))
func MMRRerank(queryEmb []float32, candidates []ScoredEdge, lambda float64, limit int) []ScoredEdge {
    if len(candidates) == 0 { return nil }
    selected := make([]ScoredEdge, 0, limit)
    remaining := make([]ScoredEdge, len(candidates))
    copy(remaining, candidates)

    // Compute cosine similarity between query and candidates
    querySims := make(map[string]float64)
    for _, c := range candidates {
        if queryEmb != nil && len(c.Edge.FactEmbedding) > 0 {
            querySims[c.Edge.UUID] = cosineSimilarity(queryEmb, c.Edge.FactEmbedding)
        }
    }

    for len(selected) < limit && len(remaining) > 0 {
        bestScore := math.Inf(-1)
        bestIdx := 0

        for i, candidate := range remaining {
            relevance := querySims[candidate.Edge.UUID]

            // Max similarity to already-selected items
            maxRedundancy := 0.0
            for _, s := range selected {
                if len(candidate.Edge.FactEmbedding) > 0 && len(s.Edge.FactEmbedding) > 0 {
                    sim := cosineSimilarity(candidate.Edge.FactEmbedding, s.Edge.FactEmbedding)
                    if sim > maxRedundancy { maxRedundancy = sim }
                }
            }

            mmrScore := lambda*relevance - (1-lambda)*maxRedundancy
            if mmrScore > bestScore {
                bestScore = mmrScore
                bestIdx = i
            }
        }

        best := remaining[bestIdx]
        best.Score = bestScore
        selected = append(selected, best)
        remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
    }
    return selected
}
```

### 4.3. Temporal Filter

```go
// internal/usecase/filter.go

// applyTemporalFilters removes edges that don't match temporal constraints
func applyTemporalFilters(edges []ScoredEdge, filters SearchFilters) []ScoredEdge {
    if filters.ValidAt == nil && filters.InvalidAt == nil &&
       filters.CreatedAtStart == nil && filters.CreatedAtEnd == nil &&
       len(filters.EntityLabels) == 0 { return edges }

    var result []ScoredEdge
    for _, se := range edges {
        e := se.Edge

        // Point-in-time temporal filter: valid_at <= refTime AND (invalid_at IS NULL OR invalid_at > refTime)
        if filters.ValidAt != nil {
            if e.ValidAt != nil && e.ValidAt.After(*filters.ValidAt) { continue }
            if e.InvalidAt != nil && !e.InvalidAt.After(*filters.ValidAt) { continue }
        }

        // Created at range filter
        if filters.CreatedAtStart != nil && e.CreatedAt.Before(*filters.CreatedAtStart) { continue }
        if filters.CreatedAtEnd != nil && e.CreatedAt.After(*filters.CreatedAtEnd) { continue }

        result = append(result, se)
    }
    return result
}
```

---

## 5. Search Result Cache — `internal/adapter/cache/`

```go
// services/graphiti-search/internal/adapter/cache/redis_cache.go

type RedisSearchCache struct {
    client redis.Client
    ttl    time.Duration  // default: 5 minutes
}

func (c *RedisSearchCache) Get(ctx context.Context, key string) (*SearchResults, bool) {
    val, err := c.client.Get(ctx, "graphiti:search:"+key).Bytes()
    if err != nil { return nil, false }
    var results SearchResults
    if err := json.Unmarshal(val, &results); err != nil { return nil, false }
    return &results, true
}

func (c *RedisSearchCache) Set(ctx context.Context, key string, results *SearchResults) {
    data, _ := json.Marshal(results)
    c.client.Set(ctx, "graphiti:search:"+key, data, c.ttl)
}

// InvalidateGroup removes all cached search results for a group
// Called when NATS receives graphiti.episode.ingested
func (c *RedisSearchCache) InvalidateGroup(ctx context.Context, groupID string) {
    // Scan and delete all keys matching group pattern
    pattern := fmt.Sprintf("graphiti:search:*%s*", groupID)
    iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
    var keys []string
    for iter.Next(ctx) { keys = append(keys, iter.Val()) }
    if len(keys) > 0 { c.client.Del(ctx, keys...) }
}

// CacheKey hash from query + config + filters
func computeCacheKey(query string, groupIDs []string, config SearchConfig, filters SearchFilters) string {
    h := sha256.New()
    fmt.Fprintf(h, "%s|%v|", query, groupIDs)
    json.NewEncoder(h).Encode(config)
    json.NewEncoder(h).Encode(filters)
    return hex.EncodeToString(h.Sum(nil))[:16]
}
```

---

## 6. NATS Cache Invalidation Subscriber

```go
// services/graphiti-search/internal/adapter/nats/cache_invalidator.go

type CacheInvalidator struct {
    cache    port.SearchCache
    natsConn *nats.Conn
}

func (ci *CacheInvalidator) Start(ctx context.Context) {
    subjects := []string{
        "graphiti.episode.ingested",
        "graphiti.entity.resolved",
        "graphiti.community.rebuilt",
    }
    for _, subj := range subjects {
        ci.natsConn.Subscribe(subj, func(msg *nats.Msg) {
            var payload struct {
                GroupID string `json:"group_id"`
            }
            json.Unmarshal(msg.Data, &payload)
            if payload.GroupID != "" {
                ci.cache.InvalidateGroup(context.Background(), payload.GroupID)
            }
        })
    }
}
```

---

## 7. Main Search Use Case (with caching)

```go
// services/graphiti-search/internal/usecase/search.go

type SearchUseCase struct {
    edgeSearch     *SearchEdgesUseCase
    nodeSearch     *SearchNodesUseCase
    episodeSearch  *SearchEpisodesUseCase
    communitySearch *SearchCommunitiesUseCase
    cache          port.SearchCache
}

func (uc *SearchUseCase) Execute(ctx context.Context, req SearchRequest) (*SearchResults, error) {
    start := time.Now()

    // Default: EdgeHybridSearchRRF
    if req.Config.EdgeConfig == nil && req.Config.NodeConfig == nil &&
       req.Config.EpisodeConfig == nil && req.Config.CommunityConfig == nil {
        req.Config = domain.EdgeHybridSearchRRF
    }

    // Check cache
    cacheKey := computeCacheKey(req.Query, req.Filters.GroupIDs, req.Config, req.Filters)
    if cached, ok := uc.cache.Get(ctx, cacheKey); ok {
        return cached, nil
    }

    // Fan-out parallel search across configured type configs
    var (
        edgeResults     *SearchResults
        nodeResults     *SearchResults
        episodeResults  *SearchResults
        communityResults *SearchResults
    )
    g, gctx := errgroup.WithContext(ctx)

    if req.Config.EdgeConfig != nil {
        g.Go(func() error { var err error; edgeResults, err = uc.edgeSearch.Execute(gctx, req); return err })
    }
    if req.Config.NodeConfig != nil {
        g.Go(func() error { var err error; nodeResults, err = uc.nodeSearch.Execute(gctx, req); return err })
    }
    if req.Config.EpisodeConfig != nil {
        g.Go(func() error { var err error; episodeResults, err = uc.episodeSearch.Execute(gctx, req); return err })
    }
    if req.Config.CommunityConfig != nil {
        g.Go(func() error { var err error; communityResults, err = uc.communitySearch.Execute(gctx, req); return err })
    }

    if err := g.Wait(); err != nil { return nil, err }

    // Merge results
    combined := mergeSearchResults(edgeResults, nodeResults, episodeResults, communityResults)
    combined.LatencyMs = time.Since(start).Milliseconds()

    // Cache result
    uc.cache.Set(ctx, cacheKey, combined)
    return combined, nil
}
```

---

## 8. gRPC Handler

```protobuf
// api/proto/graphiti/search/v1/search.proto

service SearchService {
    // Default hybrid edge search (RRF) — maps to Python graphiti.search()
    rpc Search(SearchRequest) returns (SearchResponse);

    // Configurable multi-type search — maps to Python graphiti.search_()
    rpc SearchAdvanced(SearchAdvancedRequest) returns (SearchAdvancedResponse);

    // Type-specific
    rpc SearchNodes(SearchNodesRequest) returns (SearchNodesResponse);
    rpc SearchEdges(SearchEdgesRequest) returns (SearchEdgesResponse);
    rpc SearchEpisodes(SearchEpisodesRequest) returns (SearchEpisodesResponse);
    rpc SearchCommunities(SearchCommunitiesRequest) returns (SearchCommunitiesResponse);
}

message SearchRequest {
    string query            = 1;
    repeated string group_ids = 2;
    int32 num_results       = 3;
    string center_node_uuid = 4;   // for NODE_DISTANCE reranking
    SearchConfigProto search_config = 5;
    SearchFiltersProto filters = 6;
}

message SearchResponse {
    repeated EntityEdgeProto edges       = 1;
    repeated EntityNodeProto nodes       = 2;
    repeated EpisodicNodeProto episodes  = 3;
    repeated CommunityNodeProto communities = 4;
    int64 latency_ms = 5;
}
```

---

## 9. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/graphiti-search/internal/domain/search_config.go` | SearchConfig, SearchFilters |
| `services/graphiti-search/internal/domain/recipes.go` | 6 pre-built search recipes |
| `services/graphiti-search/internal/usecase/search.go` | Main search use case + caching |
| `services/graphiti-search/internal/usecase/search_edges.go` | 6-step edge search pipeline |
| `services/graphiti-search/internal/usecase/search_nodes.go` | Node search |
| `services/graphiti-search/internal/usecase/search_episodes.go` | Episode search |
| `services/graphiti-search/internal/usecase/search_communities.go` | Community search |
| `services/graphiti-search/internal/usecase/rerank/rrf.go` | RRF algorithm |
| `services/graphiti-search/internal/usecase/rerank/mmr.go` | MMR algorithm |
| `services/graphiti-search/internal/usecase/filter.go` | Temporal + label filters |
| `services/graphiti-search/internal/adapter/cache/redis_cache.go` | Redis result cache |
| `services/graphiti-search/internal/adapter/nats/cache_invalidator.go` | NATS-based invalidation |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/graphiti-search/internal/adapter/grpc/handler.go` | Implement all 6 RPCs |
| `api/proto/graphiti/search/v1/search.proto` | Full gRPC contract |
| `apps/memory/internal/bootstrap/graphiti.go` | Init SearchUseCase + cache + NATS subscriber |

---

## 10. Acceptance Criteria Mapping

| AC từ CR-GR-004 | Covered by |
|----------------|-----------|
| Search "who joined engineering" → EntityEdge with fact | searchEdgesUC BM25+cosine+RRF |
| Latency < 1000ms with 10K nodes | parallel fanout + Redis cache |
| SearchAdvanced CombinedCrossEncoder → edges+nodes+episodes+communities | mergeSearchResults() |
| Temporal filter valid_at → only valid edges | applyTemporalFilters() |
| entity_labels filter ["Person"] → only Person nodes | applyTemporalFilters() labels |
| Cache hit 2nd call < 5ms | RedisSearchCache.Get() |
| Cache invalidation after ingest → stale result cleared | NATS CacheInvalidator |
| MMR lambda=0.5 → diverse results | MMRRerank() |
| Node Distance reranking center_uuid → closer nodes boosted | NodeDistanceReranker via Store |
| SearchNodes NodeHybridRRF → sorted by BM25+cosine | searchNodesUC |
