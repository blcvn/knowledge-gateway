---
id: DOC-S02
service: zep-graph
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-graph — API Reference

## gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.graph.v1;

service GraphService {
  rpc GetFact(GetFactRequest) returns (FactResponse);
  rpc DeleteFact(DeleteFactRequest) returns (google.protobuf.Empty);
  rpc AddGraphData(AddGraphDataRequest) returns (google.protobuf.Empty);
  rpc SetOntology(SetOntologyRequest) returns (google.protobuf.Empty);
  rpc DeleteGroup(DeleteGroupRequest) returns (google.protobuf.Empty);
  rpc GetUserNodes(GetUserNodesRequest) returns (NodeListResponse);
  rpc GetUserEdges(GetUserEdgesRequest) returns (EdgeListResponse);
  rpc GetEpisodes(GetEpisodesRequest) returns (EpisodeListResponse);
  rpc GetNode(GetNodeRequest) returns (NodeResponse);
  rpc GetEdge(GetEdgeRequest) returns (EdgeResponse);
  rpc GetEpisode(GetEpisodeRequest) returns (EpisodeResponse);
  rpc GetNodeEdges(GetNodeEdgesRequest) returns (EdgeListResponse);
  rpc GetEpisodeMentions(GetEpisodeMentionsRequest) returns (EpisodeMentionsResponse);
}
```

## Messages

### NodeResponse
```protobuf
message NodeResponse {
  string uuid = 1;
  string name = 2;
  string node_type = 3;    // User|Assistant|Preference|Organization|Event|Location|Document|Topic|Object
  string group_id = 4;
  string summary = 5;
  repeated string labels = 6;
  google.protobuf.Struct properties = 7;
  google.protobuf.Timestamp created_at = 8;
}
```

### EdgeResponse (Fact)
```protobuf
message EdgeResponse {
  string uuid = 1;
  string name = 2;          // relationship label
  string fact = 3;           // human-readable fact statement
  string source_id = 4;
  string target_id = 5;
  string edge_type = 6;     // LOCATED_AT, OCCURRED_AT, etc.
  string group_id = 7;
  optional google.protobuf.Timestamp valid_at = 8;
  optional google.protobuf.Timestamp invalid_at = 9;
  optional google.protobuf.Timestamp expired_at = 10;
  google.protobuf.Timestamp created_at = 11;
}
```

### EpisodeResponse
```protobuf
message EpisodeResponse {
  string uuid = 1;
  string name = 2;
  string content = 3;       // original message content
  string group_id = 4;
  string source_id = 5;     // message UUID (potentially prefixed)
  google.protobuf.Timestamp created_at = 6;
}
```

### AddGraphDataRequest
```protobuf
message AddGraphDataRequest {
  string user_id = 1;
  string graph_id = 2;
  string data = 3;          // text or JSON content
  string type = 4;           // "text" | "json"
}
```

## Node Ontology Hierarchy

| Priority | Node Type | Description |
|----------|-----------|-------------|
| 1 (Highest) | User | Singleton user node |
| 1 | Assistant | Singleton assistant node |
| 2 | Preference | LOW extraction threshold |
| 3 | Organization | Companies, teams |
| 3 | Event | Temporal events |
| 4 | Location | Places, addresses |
| 4 | Document | Files, articles |
| 5 | Topic | Abstract topics |
| 6 (Lowest) | Object | Last resort classification |

## Temporal Annotations

Facts use temporal annotations to track validity:
- `valid_at`: When the fact became true
- `invalid_at`: When the fact ceased to be true
- `expired_at`: When superseded by a newer fact

Example: "Alice worked at Acme" → valid_at: 2020-01-01, invalid_at: 2023-06-30
