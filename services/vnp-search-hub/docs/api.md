---
id: DOC-S02
service: vnp-search-hub
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-search-hub — API Reference

## gRPC Service Definition

```protobuf
service VNPSearchHubService {
  rpc Recall(RecallRequest) returns (RecallResponse);
  rpc SearchEngine(EngineSearchRequest) returns (EngineSearchResponse);
}
```

## RPC: Recall

Cross-engine unified search — fan-out to all 6 engines, merge, rerank.

```protobuf
message RecallRequest {
  string query = 1;
  string group_id = 2;
  int32 limit = 3;                     // Default: 20
  repeated string engines = 4;         // Empty = all engines
  string reranker = 5;                 // "rrf" (default) or "cross_encoder"
}

message RecallResponse {
  repeated UnifiedResult results = 1;
  map<string, EngineStatus> engine_status = 2;
}

message UnifiedResult {
  string id = 1;
  string source_engine = 2;
  string content = 3;
  float score = 4;
  string result_type = 5;             // "node", "edge", "episode", "document"
  map<string, string> metadata = 6;
}

message EngineStatus {
  bool available = 1;
  int32 result_count = 2;
  int64 latency_ms = 3;
}
```

## RPC: SearchEngine

Search a specific engine directly.

```protobuf
message EngineSearchRequest {
  string engine = 1;                   // "graphiti", "cognee", etc.
  string query = 2;
  string group_id = 3;
  int32 limit = 4;
}
```
