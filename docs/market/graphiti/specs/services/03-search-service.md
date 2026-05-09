# graphiti-search — Search Service

**Version:** 2.0 | **Date:** 2026-05-09  
**Origin:** Python L4 (Search & Retrieval Layer)  
**Architecture:** Clean Architecture | **Protocol:** gRPC

---

## 1. Service Overview

Search Service thực hiện **hybrid search** kết hợp nhiều phương pháp tìm kiếm, reranking đa chiến lược, và temporal/property filtering. Nó orchestrate giữa embedding generation (Knowledge) và graph queries (Store).

### Responsibilities

| Concern | Description |
|---------|-------------|
| **Hybrid Search** | Combine cosine similarity + BM25 + BFS graph traversal |
| **Reranking** | RRF, MMR, cross-encoder, node-distance, episode-mentions |
| **Filtering** | Temporal windows, entity labels, UUID restrictions, group partitioning |
| **Search Config** | Pre-built recipes + custom configurations |
| **Result Merging** | Deduplicate and merge results from multiple search methods |
| **Caching** | Redis-backed search result caching with invalidation |

---

## 2. Clean Architecture Layout

```
services/graphiti-search/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/                         # Layer 1: Entities
│   │   ├── search_config.go            #   SearchConfig, SearchMethod enum
│   │   ├── search_result.go            #   SearchResults, ScoredItem
│   │   ├── reranker.go                 #   Reranker enum, RerankStrategy
│   │   ├── filter.go                   #   SearchFilters, TemporalFilter
│   │   ├── recipe.go                   #   Pre-built search recipes
│   │   └── errors.go
│   ├── usecase/                        # Layer 2: Use Cases
│   │   ├── search_edges.go             #   Edge-focused hybrid search
│   │   ├── search_nodes.go             #   Node-focused search
│   │   ├── search_episodes.go          #   Episode search
│   │   ├── search_communities.go       #   Community-level search
│   │   ├── search_advanced.go          #   Configurable multi-type search
│   │   ├── rerank.go                   #   Reranking strategies
│   │   ├── merge_results.go            #   Result merging + dedup
│   │   ├── port/
│   │   │   ├── input.go               #   SearchUseCase interfaces
│   │   │   └── output.go             #   EmbeddingPort, GraphQueryPort, RerankerPort
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── grpc/
│   │   │   ├── handler.go             #   gRPC service implementation
│   │   │   └── mapper.go
│   │   ├── client/
│   │   │   ├── knowledge_client.go    #   For embedding + cross-encoder
│   │   │   └── store_client.go        #   For graph queries
│   │   ├── cache/
│   │   │   └── redis_cache.go         #   Search result cache
│   │   └── event/
│   │       └── subscriber.go          #   Cache invalidation on events
│   └── infra/
│       ├── config/
│       │   └── config.go
│       ├── server/
│       │   └── grpc.go
│       ├── telemetry/
│       └── wire/
├── api/
│   └── proto/
│       └── search/v1/
│           └── search.proto
├── Dockerfile
└── Makefile
```

---

## 3. gRPC API (Protobuf)

```protobuf
syntax = "proto3";
package graphiti.search.v1;

import "google/protobuf/timestamp.proto";
import "common/pagination.proto";

service SearchService {
  // Basic hybrid edge search (RRF) — maps to Python graphiti.search()
  rpc Search(SearchRequest) returns (SearchResponse);
  
  // Advanced configurable search — maps to Python graphiti.search_()
  rpc SearchAdvanced(SearchAdvancedRequest) returns (SearchAdvancedResponse);
  
  // Type-specific searches
  rpc SearchNodes(SearchNodesRequest) returns (SearchNodesResponse);
  rpc SearchEdges(SearchEdgesRequest) returns (SearchEdgesResponse);
  rpc SearchEpisodes(SearchEpisodesRequest) returns (SearchEpisodesResponse);
  rpc SearchCommunities(SearchCommunitiesRequest) returns (SearchCommunitiesResponse);
}

message SearchRequest {
  string query = 1;
  repeated string group_ids = 2;
  int32 limit = 3;                     // default: 10
  SearchFilters filters = 4;
  string center_node_uuid = 5;         // for node-distance reranking
}

message SearchAdvancedRequest {
  string query = 1;
  repeated string group_ids = 2;
  SearchConfig config = 3;
  SearchFilters filters = 4;
}

message SearchConfig {
  optional EdgeSearchConfig edge_config = 1;
  optional NodeSearchConfig node_config = 2;
  optional EpisodeSearchConfig episode_config = 3;
  optional CommunitySearchConfig community_config = 4;
  int32 limit = 5;
  double reranker_min_score = 6;
}

message EdgeSearchConfig {
  repeated SearchMethod search_methods = 1;
  Reranker reranker = 2;
  double sim_min_score = 3;            // default: 0.5
  double mmr_lambda = 4;               // default: 0.5
  int32 bfs_max_depth = 5;             // default: 2
}

// Similar for Node, Episode, Community configs

message SearchFilters {
  // Temporal
  optional google.protobuf.Timestamp created_at_start = 1;
  optional google.protobuf.Timestamp created_at_end = 2;
  optional google.protobuf.Timestamp valid_at = 3;
  optional google.protobuf.Timestamp invalid_at = 4;
  
  // Identity
  repeated string node_uuids = 5;
  repeated string edge_uuids = 6;
  
  // Type
  repeated string entity_labels = 7;
  
  // Partition
  repeated string group_ids = 8;
}

enum SearchMethod {
  SEARCH_METHOD_UNSPECIFIED = 0;
  SEARCH_METHOD_COSINE_SIMILARITY = 1;
  SEARCH_METHOD_BM25 = 2;
  SEARCH_METHOD_BFS = 3;
}

enum Reranker {
  RERANKER_UNSPECIFIED = 0;
  RERANKER_RRF = 1;
  RERANKER_MMR = 2;
  RERANKER_CROSS_ENCODER = 3;
  RERANKER_NODE_DISTANCE = 4;
  RERANKER_EPISODE_MENTIONS = 5;
}

message SearchResponse {
  repeated EntityEdgeResult edges = 1;
  SearchMeta meta = 2;
}

message SearchAdvancedResponse {
  repeated EntityEdgeResult edges = 1;
  repeated EntityNodeResult nodes = 2;
  repeated EpisodicNodeResult episodes = 3;
  repeated CommunityNodeResult communities = 4;
  SearchMeta meta = 5;
}

message SearchMeta {
  int64 total_results = 1;
  int64 search_time_ms = 2;
  string search_config_used = 3;
}
```

---

## 4. Search Pipeline

### 4.1 Core Search Flow

```
SearchRequest
  │
  ├─1─► Generate Query Embedding (→ Knowledge)
  │     knowledge.GenerateEmbedding(query)
  │     Returns: []float32 (embedding vector)
  │
  ├─2─► Parallel Search Dispatch (→ Store)
  │     ├─── store.CosineSimilaritySearch(embedding, group_ids, limit)
  │     │    Returns: []ScoredEdge (ranked by cosine distance)
  │     │
  │     ├─── store.FulltextSearch(query, group_ids, limit)
  │     │    Returns: []ScoredEdge (ranked by BM25 score)
  │     │
  │     └─── store.BFSSearch(origin_uuids, max_depth, group_ids)
  │          Returns: []ScoredEdge (ranked by graph distance)
  │
  ├─3─► Merge Results
  │     Deduplicate by UUID, preserve per-method scores
  │
  ├─4─► Rerank
  │     ├── RRF:  score = Σ 1/(k + rank_i) for each method
  │     ├── MMR:  λ·sim(q,d) - (1-λ)·max(sim(d,d')) 
  │     ├── CrossEncoder: knowledge.Rerank(query, passages)
  │     ├── NodeDistance: store.GetNodeDistances(uuids, center)
  │     └── EpisodeMentions: store.GetMentionCounts(uuids)
  │
  ├─5─► Apply Filters
  │     Filter by temporal window, labels, UUID restrictions
  │
  └─6─► Return SearchResults
```

### 4.2 Pre-built Recipes

```go
// internal/domain/recipe.go

var (
    // Default search() — most common use case
    EdgeHybridSearchRRF = SearchConfig{
        EdgeConfig: &EdgeSearchConfig{
            Methods:     []SearchMethod{BM25, CosineSimilarity},
            Reranker:    RRF,
            SimMinScore: 0.5,
        },
        Limit: 10,
    }
    
    // High-accuracy with neural reranking
    EdgeHybridSearchCrossEncoder = SearchConfig{
        EdgeConfig: &EdgeSearchConfig{
            Methods:     []SearchMethod{BM25, CosineSimilarity, BFS},
            Reranker:    CrossEncoder,
            BFSMaxDepth: 2,
        },
        Limit: 10,
    }
    
    // Comprehensive multi-type search
    CombinedHybridSearchCrossEncoder = SearchConfig{
        EdgeConfig:      &EdgeSearchConfig{...},
        NodeConfig:      &NodeSearchConfig{...},
        EpisodeConfig:   &EpisodeSearchConfig{...},
        CommunityConfig: &CommunitySearchConfig{...},
        Limit: 10,
    }
    
    // Node-focused
    NodeHybridSearchRRF = SearchConfig{
        NodeConfig: &NodeSearchConfig{
            Methods:  []SearchMethod{BM25, CosineSimilarity},
            Reranker: RRF,
        },
        Limit: 10,
    }
    
    // Community-level topic retrieval
    CommunityHybridSearchRRF = SearchConfig{
        CommunityConfig: &CommunitySearchConfig{
            Methods:  []SearchMethod{BM25, CosineSimilarity},
            Reranker: RRF,
        },
        Limit: 10,
    }
)
```

---

## 5. Reranking Strategies

### 5.1 Reciprocal Rank Fusion (RRF)

```go
// internal/usecase/rerank.go

// RRF merges results from multiple search methods
// score(d) = Σ 1/(k + rank_i(d)) for each method i
// k = 60 (standard constant)
func RRFRerank(resultSets [][]ScoredItem, k int) []ScoredItem {
    scores := make(map[string]float64)
    for _, results := range resultSets {
        for rank, item := range results {
            scores[item.UUID] += 1.0 / float64(k + rank + 1)
        }
    }
    // Sort by aggregated score, return top-N
}
```

### 5.2 Maximal Marginal Relevance (MMR)

```go
// MMR balances relevance and diversity
// score(d) = λ·sim(q,d) - (1-λ)·max(sim(d,d_selected))
func MMRRerank(query []float32, candidates []ScoredItem, lambda float64, limit int) []ScoredItem
```

### 5.3 Cross-Encoder (Neural Reranking)

```go
// Delegates to Knowledge Service for neural scoring
func CrossEncoderRerank(ctx context.Context, knowledgeClient KnowledgePort,
    query string, candidates []ScoredItem) ([]ScoredItem, error) {
    
    passages := extractPassages(candidates)
    scores, err := knowledgeClient.Rerank(ctx, query, passages)
    // Apply scores, sort, return
}
```

### 5.4 Node Distance

```go
// Boost items closer (in graph hops) to a center node
func NodeDistanceRerank(ctx context.Context, storeClient GraphQueryPort,
    candidates []ScoredItem, centerNodeUUID string) ([]ScoredItem, error)
```

### 5.5 Episode Mentions

```go
// Boost items mentioned in more recent/frequent episodes
func EpisodeMentionsRerank(ctx context.Context, storeClient GraphQueryPort,
    candidates []ScoredItem) ([]ScoredItem, error)
```

---

## 6. Search Result Caching

```go
// internal/adapter/cache/redis_cache.go

type SearchCache interface {
    Get(ctx context.Context, key CacheKey) (*SearchResults, error)
    Set(ctx context.Context, key CacheKey, results *SearchResults, ttl time.Duration) error
    Invalidate(ctx context.Context, patterns ...string) error
}

// Cache key = hash(query + group_ids + config + filters)
type CacheKey struct {
    Query    string
    GroupIDs []string
    Config   SearchConfig
    Filters  SearchFilters
}

// Cache invalidation events:
// - episode.ingested → invalidate group_id caches
// - entity.resolved → invalidate affected entity caches
// - community.rebuilt → invalidate community search caches
```

---

## 7. Configuration

```yaml
# config/search.yaml
server:
  grpc_port: 9002
  max_concurrent_searches: 100

search:
  default_limit: 10
  max_limit: 100
  rrf_k: 60                           # RRF constant
  mmr_lambda: 0.5                     # MMR diversity parameter
  bfs_max_depth: 3                    # Max BFS traversal depth
  sim_min_score: 0.5                  # Minimum similarity threshold
  timeout: 30s                        # Per-search timeout

cache:
  enabled: true
  redis_url: "redis://redis:6379/1"
  default_ttl: 300s                   # 5 minutes
  max_entries: 10000

services:
  knowledge:
    address: "graphiti-knowledge:9003"
    timeout: 60s
  store:
    address: "graphiti-store:9004"
    timeout: 15s

events:
  nats_url: "nats://nats:4222"
  subjects:
    - "episode.ingested"               # trigger cache invalidation
    - "entity.resolved"
    - "community.rebuilt"

telemetry:
  otel_endpoint: "otel-collector:4317"
  service_name: "graphiti-search"
```

---

## 8. Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `search_queries_total` | Counter | config, status | Total search queries |
| `search_duration_seconds` | Histogram | config, method | Search latency |
| `search_results_count` | Histogram | config, type | Results per query |
| `search_cache_hits_total` | Counter | — | Cache hit count |
| `search_cache_misses_total` | Counter | — | Cache miss count |
| `search_rerank_duration_seconds` | Histogram | reranker | Reranking latency |
| `search_embedding_duration_seconds` | Histogram | — | Query embedding time |

---

## 9. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Search as separate service** | Independent scaling — search load differs from ingestion load |
| **Embedding via Knowledge service** | Single source of truth for AI model access |
| **Redis caching with event invalidation** | Low-latency repeated queries; events prevent stale results |
| **Pre-built recipes as domain constants** | Go constants are compile-time safe; no runtime config parsing errors |
| **Parallel search dispatch** | Fan-out to Store for cosine/BM25/BFS simultaneously via goroutines |
| **Reranker in Search, not Knowledge** | RRF/MMR/NodeDistance are algorithmic, not AI — only cross-encoder needs Knowledge |
