---
id: DOC-S02
service: zep-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-search — API Reference

## gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.search.v1;

service SearchService {
  rpc GraphSearch(GraphSearchRequest) returns (SearchResponse);
  rpc SearchSessions(SessionSearchRequest) returns (SessionSearchResponse);
  rpc GetRelevantFacts(GetRelevantFactsRequest) returns (FactListResponse);
}
```

## Messages

### GraphSearchRequest
```protobuf
message GraphSearchRequest {
  string query = 1;
  optional string user_id = 2;
  repeated string group_ids = 3;
  string scope = 4;              // "edges" | "nodes" | "episodes" | "all"
  string reranker = 5;           // "rrf" | "mmr" | "cross_encoder" | "node_distance" | "episode_mentions"
  repeated string node_labels = 6;
  repeated string edge_types = 7;
  int32 limit = 8;
  optional double min_fact_rating = 9;
  optional double mmr_lambda = 10;      // 0.0=diversity, 1.0=relevance
  optional string center_node_uuid = 11; // for node_distance reranker
}
```

### SearchResponse
```protobuf
message SearchResponse {
  repeated SearchItem items = 1;
  int32 total = 2;
  string query = 3;
  string scope = 4;
  string reranker = 5;
  int64 latency_ms = 6;
}

message SearchItem {
  string uuid = 1;
  double score = 2;
  oneof result {
    FactResult fact = 3;
    NodeResult node = 4;
    EpisodeResult episode = 5;
  }
}
```

### GetRelevantFactsRequest
```protobuf
message GetRelevantFactsRequest {
  string group_id = 1;
  repeated string query_messages = 2;   // last 4 messages as context
  int32 max_facts = 3;                  // default: 5
}
```

## RPC Details

### GraphSearch

| Attribute | Value |
|-----------|-------|
| **Cache** | Redis with 30s TTL |
| **Search scopes** | edges (facts), nodes (entities), episodes, all |
| **Reranking** | 5 strategies with configurable parameters |

### GetRelevantFacts (Internal)

| Attribute | Value |
|-----------|-------|
| **Caller** | zep-memory (GetMemory context assembly) |
| **Input** | GroupID + last 4 messages as search context |
| **Output** | Max 5 relevant facts from knowledge graph |
