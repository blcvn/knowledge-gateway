# TASK-GR-013 — Hybrid Search Engine (BM25 + Cosine + BFS + 5 Rerankers)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-013 |
| **Wave** | 2 (Ingestion & Search) |
| **Component** | `services/graphiti-search/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-004 §3, §4 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-GR-004 |
| **Estimated** | 6h |

---

## Context

Implement `SearchUseCase` với 3-stream parallel fan-out (BM25 + cosine + BFS) và 5 rerankers: RRF, MMR, CrossEncoder, NodeDistance, EpisodeMentions. Hỗ trợ 4 node types (Edge, Node, Episode, Community) cùng lúc.

---

## Goal

- `SearchEdgesUseCase` — parallel BM25 + cosine + BFS, 5 rerankers, temporal filter
- `SearchNodesUseCase`, `SearchEpisodesUseCase`, `SearchCommunitiesUseCase`
- `SearchUseCase` — main orchestrator: fan-out to 4 type searches + merge
- Rerankers: RRF, MMR, CrossEncoder (via knowledge Rerank), NodeDistance, EpisodeMentions
- 6 pre-built search recipes (domain-config constants)

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-search/internal/domain/search_config.go` |
| CREATE | `services/graphiti-search/internal/domain/recipes.go` |
| CREATE | `services/graphiti-search/internal/usecase/search_edges.go` |
| CREATE | `services/graphiti-search/internal/usecase/search_nodes.go` |
| CREATE | `services/graphiti-search/internal/usecase/search_episodes.go` |
| CREATE | `services/graphiti-search/internal/usecase/search_communities.go` |
| CREATE | `services/graphiti-search/internal/usecase/search.go` |
| CREATE | `services/graphiti-search/internal/usecase/rerank/rrf.go` |
| CREATE | `services/graphiti-search/internal/usecase/rerank/mmr.go` |
| CREATE | `services/graphiti-search/internal/usecase/filter.go` |

---

## Implementation

### File 1: `services/graphiti-search/internal/domain/search_config.go`

```go
package domain

import "time"

type SearchMethod string
const (
    MethodCosineSimilarity SearchMethod = "COSINE_SIMILARITY"
    MethodBM25             SearchMethod = "BM25"
    MethodBFS              SearchMethod = "BFS"
)

type Reranker string
const (
    RerankerRRF             Reranker = "RRF"
    RerankerMMR             Reranker = "MMR"
    RerankerCrossEncoder    Reranker = "CROSS_ENCODER"
    RerankerNodeDistance    Reranker = "NODE_DISTANCE"
    RerankerEpisodeMentions Reranker = "EPISODE_MENTIONS"
    RerankerNone            Reranker = "NONE"
)

type EdgeSearchConfig struct {
    Methods     []SearchMethod
    Reranker    Reranker
    SimMinScore float64
    MMRLambda   float64
    BFSMaxDepth int
}

type NodeSearchConfig struct {
    Methods     []SearchMethod
    Reranker    Reranker
    SimMinScore float64
}

type EpisodeSearchConfig struct {
    Methods  []SearchMethod
    Reranker Reranker
}

type CommunitySearchConfig struct {
    Methods     []SearchMethod
    Reranker    Reranker
    SimMinScore float64
}

type SearchConfig struct {
    EdgeConfig      *EdgeSearchConfig
    NodeConfig      *NodeSearchConfig
    EpisodeConfig   *EpisodeSearchConfig
    CommunityConfig *CommunitySearchConfig
    Limit           int
}

type SearchFilters struct {
    CreatedAtStart *time.Time
    CreatedAtEnd   *time.Time
    ValidAt        *time.Time  // point-in-time query
    EntityLabels   []string
    GroupIDs       []string
}

type SearchRequest struct {
    Query          string
    Config         SearchConfig
    Filters        SearchFilters
    CenterNodeUUID string  // for NODE_DISTANCE reranking
}

type ScoredEdge struct {
    Edge   interface{}  // *graph.EntityEdge
    Score  float64
    Source string  // "cosine" | "bm25" | "bfs"
}

type ScoredNode struct {
    Node   interface{}  // *graph.EntityNode
    Score  float64
    Source string
}

type SearchResults struct {
    Edges       []interface{}
    Nodes       []interface{}
    Episodes    []interface{}
    Communities []interface{}
    LatencyMs   int64
}
```

### File 2: `services/graphiti-search/internal/domain/recipes.go`

```go
package domain

// Pre-built search recipes — ready-to-use SearchConfig instances

// EdgeHybridSearchRRF — Default: BM25 + cosine + RRF (fastest, good quality)
var EdgeHybridSearchRRF = SearchConfig{
    EdgeConfig: &EdgeSearchConfig{
        Methods:     []SearchMethod{MethodBM25, MethodCosineSimilarity},
        Reranker:    RerankerRRF,
        SimMinScore: 0.5,
    },
    Limit: 10,
}

// EdgeHybridSearchMMR — Diverse results
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

// NodeHybridSearchRRF — Node-focused hybrid search
var NodeHybridSearchRRF = SearchConfig{
    NodeConfig: &NodeSearchConfig{
        Methods:  []SearchMethod{MethodBM25, MethodCosineSimilarity},
        Reranker: RerankerRRF,
    },
    Limit: 10,
}

// CommunityHybridSearchRRF — Community/topic-level search
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

var RecipeByName = map[string]SearchConfig{
    "edge_hybrid_rrf":           EdgeHybridSearchRRF,
    "edge_hybrid_mmr":           EdgeHybridSearchMMR,
    "edge_hybrid_cross_encoder": EdgeHybridSearchCrossEncoder,
    "node_hybrid_rrf":           NodeHybridSearchRRF,
    "community_hybrid_rrf":      CommunityHybridSearchRRF,
    "combined_cross_encoder":    CombinedHybridSearchCrossEncoder,
}
```

### File 3: `services/graphiti-search/internal/usecase/rerank/rrf.go`

```go
package rerank

import (
    "sort"

    "github.com/vnp-memory/services/graphiti-search/internal/domain"
)

// RRFRerank implements Reciprocal Rank Fusion.
// score(d) = Σ 1/(k + rank_i(d)) for each retrieval method i
// k=60 is the standard constant (Cormack et al., 2009)
func RRFEdges(resultSets map[domain.SearchMethod][]domain.ScoredEdge, k int) []domain.ScoredEdge {
    if k <= 0 { k = 60 }
    scores := make(map[string]float64)
    edges  := make(map[string]domain.ScoredEdge)

    for _, results := range resultSets {
        for rank, item := range results {
            id := extractEdgeID(item)
            scores[id] += 1.0 / float64(k+rank+1)
            if _, exists := edges[id]; !exists { edges[id] = item }
        }
    }

    items := make([]domain.ScoredEdge, 0, len(edges))
    for id, edge := range edges {
        edge.Score = scores[id]
        items = append(items, edge)
    }
    sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
    return items
}

func extractEdgeID(e domain.ScoredEdge) string {
    // Type assertion to get UUID — in real impl use graph.EntityEdge
    if edge, ok := e.Edge.(interface{ GetUUID() string }); ok { return edge.GetUUID() }
    return ""
}
```

### File 4: `services/graphiti-search/internal/usecase/rerank/mmr.go`

```go
package rerank

import (
    "math"
    "sort"

    "github.com/vnp-memory/services/graphiti-search/internal/domain"
)

// MMRRerank implements Maximal Marginal Relevance.
// score(d) = λ·sim(q,d) - (1-λ)·max_s∈S(sim(d,s))
// where S is the set of already-selected documents.
func MMREdges(queryEmb []float32, candidates []domain.ScoredEdge, lambda float64, limit int) []domain.ScoredEdge {
    if len(candidates) == 0 || limit == 0 { return nil }

    selected  := make([]domain.ScoredEdge, 0, limit)
    remaining := make([]domain.ScoredEdge, len(candidates))
    copy(remaining, candidates)

    // Pre-compute query similarities
    querySims := make(map[string]float64, len(candidates))
    for _, c := range candidates {
        id := extractEdgeID(c)
        if emb := getEdgeEmbedding(c); emb != nil && queryEmb != nil {
            querySims[id] = cosineSim(queryEmb, emb)
        }
    }

    for len(selected) < limit && len(remaining) > 0 {
        bestScore := math.Inf(-1)
        bestIdx   := 0

        for i, candidate := range remaining {
            id        := extractEdgeID(candidate)
            relevance := querySims[id]

            // Max similarity to already-selected
            maxRedundancy := 0.0
            for _, s := range selected {
                cEmb := getEdgeEmbedding(candidate)
                sEmb := getEdgeEmbedding(s)
                if cEmb != nil && sEmb != nil {
                    sim := cosineSim(cEmb, sEmb)
                    if sim > maxRedundancy { maxRedundancy = sim }
                }
            }

            mmrScore := lambda*relevance - (1-lambda)*maxRedundancy
            if mmrScore > bestScore { bestScore = mmrScore; bestIdx = i }
        }

        best := remaining[bestIdx]
        best.Score = bestScore
        selected  = append(selected, best)
        remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
    }
    return selected
}

func cosineSim(a, b []float32) float64 {
    if len(a) != len(b) || len(a) == 0 { return 0 }
    var dot, normA, normB float64
    for i := range a {
        dot  += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    if normA == 0 || normB == 0 { return 0 }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func getEdgeEmbedding(e domain.ScoredEdge) []float32 {
    if edge, ok := e.Edge.(interface{ GetFactEmbedding() []float32 }); ok {
        return edge.GetFactEmbedding()
    }
    return nil
}
```

### File 5: `services/graphiti-search/internal/usecase/filter.go`

```go
package usecase

import (
    "time"

    "github.com/vnp-memory/pkg/graph"
    "github.com/vnp-memory/services/graphiti-search/internal/domain"
)

// ApplyTemporalFilters removes edges that don't match temporal constraints
func ApplyTemporalFilters(edges []domain.ScoredEdge, filters domain.SearchFilters) []domain.ScoredEdge {
    if noFilters(filters) { return edges }

    var result []domain.ScoredEdge
    for _, se := range edges {
        e, ok := se.Edge.(*graph.EntityEdge)
        if !ok { result = append(result, se); continue }

        // Point-in-time temporal filter
        if filters.ValidAt != nil && !e.IsValidAt(*filters.ValidAt) { continue }

        // Created-at range filter
        if filters.CreatedAtStart != nil && e.CreatedAt.Before(*filters.CreatedAtStart) { continue }
        if filters.CreatedAtEnd   != nil && e.CreatedAt.After(*filters.CreatedAtEnd)   { continue }

        result = append(result, se)
    }
    return result
}

func noFilters(f domain.SearchFilters) bool {
    return f.ValidAt == nil && f.CreatedAtStart == nil &&
        f.CreatedAtEnd == nil && len(f.EntityLabels) == 0
}
```

### File 6: `services/graphiti-search/internal/usecase/search.go`

```go
package usecase

import (
    "context"
    "sync"
    "time"

    "golang.org/x/sync/errgroup"
    "github.com/vnp-memory/services/graphiti-search/internal/domain"
)

type SearchUseCase struct {
    edgeSearch      *SearchEdgesUseCase
    nodeSearch      *SearchNodesUseCase
    episodeSearch   *SearchEpisodesUseCase
    communitySearch *SearchCommunitiesUseCase
    cache           SearchCache
}

func NewSearchUseCase(
    edges *SearchEdgesUseCase,
    nodes *SearchNodesUseCase,
    episodes *SearchEpisodesUseCase,
    communities *SearchCommunitiesUseCase,
    cache SearchCache,
) *SearchUseCase {
    return &SearchUseCase{
        edgeSearch: edges, nodeSearch: nodes,
        episodeSearch: episodes, communitySearch: communities,
        cache: cache,
    }
}

func (uc *SearchUseCase) Execute(ctx context.Context, req domain.SearchRequest) (*domain.SearchResults, error) {
    start := time.Now()

    // Default: EdgeHybridSearchRRF
    if req.Config.EdgeConfig == nil && req.Config.NodeConfig == nil &&
        req.Config.EpisodeConfig == nil && req.Config.CommunityConfig == nil {
        req.Config = domain.EdgeHybridSearchRRF
    }
    if req.Config.Limit == 0 { req.Config.Limit = 10 }

    // Cache check
    cacheKey := computeCacheKey(req)
    if cached, ok := uc.cache.Get(ctx, cacheKey); ok { return cached, nil }

    // Fan-out parallel search
    var (
        edgeResults     *domain.SearchResults
        nodeResults     *domain.SearchResults
        episodeResults  *domain.SearchResults
        communityResults *domain.SearchResults
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

    combined := mergeSearchResults(edgeResults, nodeResults, episodeResults, communityResults)
    combined.LatencyMs = time.Since(start).Milliseconds()

    uc.cache.Set(ctx, cacheKey, combined)
    return combined, nil
}

func mergeSearchResults(results ...*domain.SearchResults) *domain.SearchResults {
    merged := &domain.SearchResults{}
    for _, r := range results {
        if r == nil { continue }
        merged.Edges       = append(merged.Edges,       r.Edges...)
        merged.Nodes       = append(merged.Nodes,       r.Nodes...)
        merged.Episodes    = append(merged.Episodes,    r.Episodes...)
        merged.Communities = append(merged.Communities, r.Communities...)
    }
    return merged
}
```

### File 7: `services/graphiti-search/internal/usecase/search_edges.go` (core)

```go
package usecase

import (
    "context"
    "sort"
    "sync"

    "github.com/vnp-memory/services/graphiti-search/internal/domain"
    "github.com/vnp-memory/services/graphiti-search/internal/usecase/rerank"
)

type SearchEdgesUseCase struct {
    storeClient   StorePort
    knowledgeClient KnowledgePort
}

func (uc *SearchEdgesUseCase) Execute(ctx context.Context, req domain.SearchRequest) (*domain.SearchResults, error) {
    cfg := req.Config.EdgeConfig
    if cfg == nil { return &domain.SearchResults{}, nil }

    // Step 1: Embed query
    var queryEmb []float32
    queryEmb, _ = uc.knowledgeClient.GenerateEmbedding(ctx, req.Query)

    // Step 2: Parallel search dispatch
    type searchResult struct {
        edges  []domain.ScoredEdge
        method domain.SearchMethod
    }
    resultCh := make(chan searchResult, 3)
    var wg sync.WaitGroup

    for _, method := range cfg.Methods {
        wg.Add(1)
        go func(m domain.SearchMethod) {
            defer wg.Done()
            var edges []domain.ScoredEdge
            switch m {
            case domain.MethodCosineSimilarity:
                if queryEmb != nil {
                    raw, _ := uc.storeClient.EdgeSimilaritySearch(ctx, queryEmb, req.Filters.GroupIDs, req.Config.Limit*3, cfg.SimMinScore)
                    for _, e := range raw { edges = append(edges, domain.ScoredEdge{Edge: e, Source: "cosine", Score: 1.0}) }
                }
            case domain.MethodBM25:
                raw, _ := uc.storeClient.EdgeFulltextSearch(ctx, req.Query, req.Filters.GroupIDs, req.Config.Limit*3, toStoreFilters(req.Filters))
                for _, e := range raw { edges = append(edges, domain.ScoredEdge{Edge: e, Source: "bm25", Score: 1.0}) }
            case domain.MethodBFS:
                originNodes, _ := uc.storeClient.NodeFulltextSearch(ctx, req.Query, req.Filters.GroupIDs, 5)
                if len(originNodes) > 0 {
                    originUUIDs := extractNodeUUIDs(originNodes)
                    raw, _ := uc.storeClient.EdgeBFSSearch(ctx, originUUIDs, cfg.BFSMaxDepth, req.Filters.GroupIDs, req.Config.Limit*2)
                    for _, e := range raw { edges = append(edges, domain.ScoredEdge{Edge: e, Source: "bfs", Score: 0.8}) }
                }
            }
            resultCh <- searchResult{edges: edges, method: m}
        }(method)
    }

    go func() { wg.Wait(); close(resultCh) }()

    // Collect results
    resultSets := make(map[domain.SearchMethod][]domain.ScoredEdge)
    merged := make(map[string]domain.ScoredEdge)
    for result := range resultCh {
        resultSets[result.method] = result.edges
        for _, e := range result.edges {
            id := extractEdgeUUID(e)
            if _, exists := merged[id]; !exists { merged[id] = e }
        }
    }
    candidates := mapToSlice(merged)

    // Step 3: Rerank
    var reranked []domain.ScoredEdge
    switch cfg.Reranker {
    case domain.RerankerRRF:
        reranked = rerank.RRFEdges(resultSets, 60)
    case domain.RerankerMMR:
        reranked = rerank.MMREdges(queryEmb, candidates, cfg.MMRLambda, req.Config.Limit)
    case domain.RerankerCrossEncoder:
        passages := extractFacts(candidates)
        scores, err := uc.knowledgeClient.Rerank(ctx, req.Query, passages)
        if err != nil { reranked = candidates } else { reranked = applyScores(candidates, scores) }
    case domain.RerankerNodeDistance:
        if req.CenterNodeUUID != "" {
            distances, _ := uc.storeClient.NodeDistanceReranker(ctx, extractEdgeNodeUUIDs(candidates), req.CenterNodeUUID)
            reranked = applyDistanceScores(candidates, distances)
        } else { reranked = candidates }
    case domain.RerankerEpisodeMentions:
        mentions, _ := uc.storeClient.EpisodeMentionsReranker(ctx, extractEdgeNodeUUIDs(candidates))
        reranked = applyMentionScores(candidates, mentions)
    default:
        reranked = candidates
    }

    // Step 4: Temporal filter
    filtered := ApplyTemporalFilters(reranked, req.Filters)

    // Step 5: Limit
    if len(filtered) > req.Config.Limit { filtered = filtered[:req.Config.Limit] }

    // Convert to results
    edges := make([]interface{}, len(filtered))
    for i, se := range filtered { edges[i] = se.Edge }
    return &domain.SearchResults{Edges: edges}, nil
}

func mapToSlice(m map[string]domain.ScoredEdge) []domain.ScoredEdge {
    items := make([]domain.ScoredEdge, 0, len(m))
    for _, v := range m { items = append(items, v) }
    return items
}
```

---

## Verification

```bash
cd services/graphiti-search
go build ./...
go vet ./...
go test ./internal/usecase/rerank/... -v
```

**Key behaviors to verify:**
1. `SearchEdgesUseCase` with RRF: cosine + BM25 results merged by rank
2. `SearchEdgesUseCase` with MMR lambda=0.5: diverse results (low redundancy)
3. Temporal filter `valid_at` correctly excludes invalidated edges
4. `CombinedHybridSearchCrossEncoder` recipe → all 4 type results returned
5. Cache hit second call returns immediately
