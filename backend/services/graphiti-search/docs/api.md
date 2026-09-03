---
id: DOC-S02
service: graphiti-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-search — API Reference

## gRPC Service Definition

```protobuf
service GraphitiSearchService {
  // Hybrid search combining vector + full-text + graph traversal
  rpc HybridSearch(SearchRequest) returns (SearchResponse);
  // Search entity nodes by name similarity
  rpc NodeSearch(NodeSearchRequest) returns (NodeSearchResponse);
  // Search entity edges/facts by content
  rpc EdgeSearch(EdgeSearchRequest) returns (EdgeSearchResponse);
  // Search community summaries
  rpc CommunitySearch(CommunitySearchRequest) returns (CommunitySearchResponse);
}
```

## RPC: HybridSearch

```protobuf
message SearchRequest {
  string query = 1;
  string group_id = 2;
  int32 limit = 3;                        // Default: 10
  SearchConfig config = 4;                // Search strategy configuration
  SearchFilters filters = 5;              // Temporal + property filters
}

message SearchConfig {
  EdgeSearchConfig edge_config = 1;
  NodeSearchConfig node_config = 2;
  EpisodeSearchConfig episode_config = 3;
  CommunitySearchConfig community_config = 4;
  float reranker_min_score = 5;           // Minimum score threshold
}

message SearchFilters {
  google.protobuf.Timestamp created_after = 1;
  google.protobuf.Timestamp created_before = 2;
  google.protobuf.Timestamp valid_after = 3;
  google.protobuf.Timestamp valid_before = 4;
  repeated string group_ids = 5;
  repeated string entity_types = 6;
  repeated string edge_types = 7;
}

message SearchResponse {
  repeated EntityEdge edges = 1;
  repeated float edge_scores = 2;
  repeated EntityNode nodes = 3;
  repeated float node_scores = 4;
  repeated EpisodicNode episodes = 5;
  repeated float episode_scores = 6;
  repeated CommunityNode communities = 7;
  repeated float community_scores = 8;
}
```

## Search Methods

| Method | Edge | Node | Episode | Community |
|--------|------|------|---------|-----------|
| `cosine_similarity` | ✅ | ✅ | ❌ | ✅ |
| `bm25` | ✅ | ✅ | ✅ | ✅ |
| `breadth_first_search` | ✅ | ✅ | ❌ | ❌ |

## Reranking Strategies

| Strategy | Enum | Use Case |
|----------|------|----------|
| Reciprocal Rank Fusion | `rrf` | Default, fast, multi-source merge |
| Maximal Marginal Relevance | `mmr` | Diversity-focused results |
| Cross-Encoder | `cross_encoder` | Highest quality (via graphiti-knowledge) |
| Node Distance | `node_distance` | Graph topology-aware ranking |
| Episode Mentions | `episode_mentions` | Frequency-weighted ranking |

## Pre-built Search Recipes

| Recipe | Description |
|--------|-------------|
| `EDGE_HYBRID_SEARCH_RRF` | Edge cosine + BM25 with RRF reranking |
| `EDGE_HYBRID_SEARCH_NODE_DISTANCE` | Edge search ranked by graph distance |
| `COMBINED_HYBRID_SEARCH_CROSS_ENCODER` | All entity types with cross-encoder reranking |

## NATS Events Subscribed

| Subject | Action |
|---------|--------|
| `graphiti.episode.ingested` | Invalidate search cache for group_id |
| `graphiti.entity.resolved` | Update node index |
| `graphiti.community.rebuilt` | Update community index |
