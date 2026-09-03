# TASK-ZEP-013 — services/search-service: Graph Search & 5 Rerankers

**Task ID:** TASK-ZEP-013  
**Wave:** 4 (Graph Intelligence)  
**Solution:** [SOL-ZEP-004](../solutions/SOL-ZEP-004-Semantic-Graph-Search-Reranking.md)  
**Depends on:** TASK-ZEP-011 (graph domain), TASK-ZEP-010 (Neo4j infra)  
**Ước tính:** 5h  
**Priority:** High

---

## Mục tiêu

Nâng cấp `services/search-service/` với graph-aware search và 5 reranking strategies:
1. **GraphSearchUseCase** — multi-scope: edges/nodes/episodes/all
2. **5 Rerankers**: RRF, MMR, Cross-Encoder, Node Distance, Episode Mentions
3. **Redis cache** TTL 30s + NATS invalidation
4. **GetRelevantFacts** — internal endpoint cho Memory Service (GetMemory)

---

## Công việc cụ thể

### 1. Tạo Domain Types

**`services/search-service/internal/domain/graph_search.go`**

```go
type SearchScope string
const (
    SearchScopeEdges    SearchScope = "edges"    // temporal facts only
    SearchScopeNodes    SearchScope = "nodes"    // entity nodes only
    SearchScopeEpisodes SearchScope = "episodes" // source events only
    SearchScopeAll      SearchScope = "all"      // expensive: all types
)

type RerankerType string
const (
    RerankerRRF             RerankerType = "rrf"
    RerankerMMR             RerankerType = "mmr"
    RerankerCrossEncoder    RerankerType = "cross_encoder"
    RerankerNodeDistance    RerankerType = "node_distance"
    RerankerEpisodeMentions RerankerType = "episode_mentions"
)

type GraphSearchQuery struct {
    Query          string
    UserID         *string
    GroupIDs       []string
    Scope          SearchScope
    Reranker       RerankerType
    NodeLabels     []string     // filter by NodeType
    EdgeTypes      []string     // filter by relationship type
    Limit          int          // default 10
    MinFactRating  *float64
    MmrLambda      *float64     // 0.0=diversity, 1.0=relevance
    CenterNodeUUID *string      // for node_distance
}
```

### 2. Implement 5 Rerankers

**`services/search-service/internal/infra/reranker/`**

#### `rrf.go` — Reciprocal Rank Fusion
```go
// RRFReranker fuses vector score + BM25 rank + fact_rating rank
// Score = Σ 1/(k + rank_i) for each ranking list
// k default: 60
// No external deps — pure Go
type RRFReranker struct{ K int }
func (r *RRFReranker) Rerank(ctx context.Context, query string, results []GraphSearchResult) ([]GraphSearchResult, error)
```

#### `mmr.go` — Maximal Marginal Relevance
```go
// MMRReranker balances relevance vs diversity
// MMR = argmax[λ*sim(d,q) - (1-λ)*max_sim(d,S)]
// S = already selected results
// lambda: 0.0=max diversity, 1.0=max relevance (default 0.5)
type MMRReranker struct{ DefaultLambda float64 }
func (r *MMRReranker) Rerank(...) — iteratively select most diverse + relevant
```

#### `cross_encoder.go` — Neural Cross-Encoder
```go
// CrossEncoderReranker calls external cross-encoder service (HTTP)
// Fallback: RRF on error
// Batch size: 32
type CrossEncoderReranker struct {
    ModelURL  string
    BatchSize int
    fallback  *RRFReranker
}
```

#### `node_distance.go` — Graph Shortest Path
```go
// NodeDistanceReranker ranks by graph distance from centerNode
// Needs Neo4j shortest path query
// MaxDepth: 3
type NodeDistanceReranker struct {
    MaxDepth int
    neo4j    Neo4jClient
}
```

#### `episode_mentions.go` — Episode Co-occurrence
```go
// EpisodeMentionsReranker scores by recency of episode mentions
// score = Σ decay^days_ago for each episode mentioning fact
// TimeDecay default: 0.95
type EpisodeMentionsReranker struct{ TimeDecay float64 }
```

### 3. Implement Neo4j Graph Searcher

**`services/search-service/internal/infra/neo4j/graph_searcher.go`**

```cypher
-- SearchEdges (vector + temporal filter):
CALL db.index.vector.queryRelationships('edge_embedding_idx', $over_fetch_limit, $embedding)
YIELD relationship AS r, score
WHERE r.group_id IN $group_ids
  AND r.invalid_at IS NULL
  AND ($min_fact_rating IS NULL OR r.fact_rating >= $min_fact_rating)
  AND ($edge_types IS NULL OR type(r) IN $edge_types)
RETURN r, score ORDER BY score DESC

-- SearchNodes (vector):
CALL db.index.vector.queryNodes('entity_embedding_idx', $over_fetch_limit, $embedding)
YIELD node AS n, score
WHERE n.group_id IN $group_ids
  AND ($node_labels IS NULL OR n.node_type IN $node_labels)
RETURN n, score ORDER BY score DESC
```

### 4. Implement Redis Cache + NATS Invalidation

**`services/search-service/internal/infra/redis/search_cache.go`**
```go
// TTL: 30 seconds
// Key: SHA-256(query + scope + reranker + filters)[:16]
func (c *SearchCache) Get(ctx context.Context, key string) (*GraphSearchResponse, error)
func (c *SearchCache) Set(ctx context.Context, key string, val *GraphSearchResponse, ttl time.Duration)
func (c *SearchCache) DeleteByPattern(ctx context.Context, pattern string)  // for group invalidation
```

**`services/search-service/internal/adapter/subscriber/graph_events.go`**
```go
// Subscribe to NATS and invalidate cache:
// "zep.graph.extraction.completed" → DeleteByPattern("search:*:{groupID}:*")
// "zep.graph.fact.invalidated" → DeleteByPattern("search:*:{groupID}:*")
```

### 5. Implement `GetRelevantFacts` (Internal)

```go
// GetRelevantFacts: internal gRPC endpoint chỉ dùng bởi Memory Service
// Timeout target: 100ms (để GetMemory không bị chậm)
// Luôn trả về slice (kể cả rỗng) — không return error cho graceful degradation
func (uc *GetRelevantFactsUseCase) Execute(ctx context.Context, req GetRelevantFactsRequest) ([]Fact, error)
```

### 6. Tests

- `TestRRFReranker_FusesMultipleLists`: 3 ranking lists → RRF scores correct
- `TestMMRReranker_Lambda1_SameAsRelevance`: lambda=1 → sorted by relevance only
- `TestMMRReranker_Lambda0_MaxDiversity`: lambda=0 → no duplicate similar results
- `TestSearchCache_InvalidationOnExtraction`: publish NATS event → cache cleared
- `TestGraphSearch_ScopeEdges_NoNodes`: scope=edges → result type all "edges"
- `TestGraphSearch_MinFactRating_Filters`: rating=0.8 → no results below threshold
- `TestGetRelevantFacts_SearchDown_ReturnsEmpty`: searcher error → empty slice (no error)

---

## Acceptance Criteria

- [ ] `go build ./services/search-service/...` không có lỗi
- [ ] scope=edges → results chỉ là TemporalEdge (không có nodes/episodes)
- [ ] reranker=mmr,lambda=0 → results đa dạng (không có duplicate similar)
- [ ] min_fact_rating=0.8 → kết quả có fact_rating >= 0.8
- [ ] Cache hit → response trong < 10ms
- [ ] NATS `extraction.completed` → cache cleared ngay
- [ ] GetRelevantFacts error → trả về `[]Fact{}` (không crash Memory Service)
- [ ] `go test ./services/search-service/...` pass

---

## Files tạo ra

```
services/search-service/
├── internal/
│   ├── domain/
│   │   └── graph_search.go
│   ├── usecase/
│   │   ├── graph_search.go
│   │   ├── graph_search_test.go
│   │   └── get_relevant_facts.go
│   ├── infra/
│   │   ├── reranker/
│   │   │   ├── rrf.go
│   │   │   ├── mmr.go
│   │   │   ├── cross_encoder.go
│   │   │   ├── node_distance.go
│   │   │   ├── episode_mentions.go
│   │   │   └── reranker_test.go
│   │   ├── neo4j/
│   │   │   └── graph_searcher.go
│   │   └── redis/
│   │       └── search_cache.go
│   └── adapter/
│       └── subscriber/
│           └── graph_events.go
```

## Sau khi hoàn thành

Chạy: `go build ./services/search-service/... && go test ./services/search-service/...`
