---
id: DOC-S02
service: graphiti-store
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-store — API Reference

## gRPC Service Definition

```protobuf
service GraphitiStoreService {
  // Node CRUD
  rpc SaveNode(SaveNodeRequest) returns (Node);
  rpc GetNode(GetNodeRequest) returns (Node);
  rpc DeleteNode(DeleteNodeRequest) returns (google.protobuf.Empty);

  // Edge CRUD
  rpc SaveEdge(SaveEdgeRequest) returns (Edge);
  rpc GetEdge(GetEdgeRequest) returns (Edge);
  rpc DeleteEdge(DeleteEdgeRequest) returns (google.protobuf.Empty);

  // Bulk operations
  rpc SaveBulk(SaveBulkRequest) returns (SaveBulkResponse);

  // Search primitives
  rpc CosineSimilaritySearch(VectorSearchRequest) returns (SearchResponse);
  rpc FulltextSearch(TextSearchRequest) returns (SearchResponse);
  rpc BFSSearch(BFSSearchRequest) returns (SearchResponse);

  // Group management
  rpc DeleteByGroupID(DeleteByGroupRequest) returns (google.protobuf.Empty);

  // Index management
  rpc BuildIndices(BuildIndicesRequest) returns (google.protobuf.Empty);
}
```

## Node Types (Graph Labels)

| Label | Description | Key Fields |
|-------|-------------|-----------|
| `Entity` | Named entity node | uuid, name, group_id, summary, name_embedding, labels, attributes |
| `Episodic` | Episode content node | uuid, name, group_id, content, source, valid_at, entity_edges |
| `Community` | Community summary node | uuid, name, group_id, summary, name_embedding |
| `Saga` | Episode group node | uuid, name, group_id, summary, first/last_episode_uuid |

## Edge Types (Relationships)

| Relationship | Source → Target | Key Fields |
|-------------|----------------|-----------|
| `RELATES_TO` | Entity → Entity | uuid, name, fact, fact_embedding, valid_at, invalid_at, expired_at |
| `MENTIONS` | Episodic → Entity | uuid, group_id, created_at |
| `HAS_MEMBER` | Community → Entity | uuid, group_id, created_at |
| `HAS_EPISODE` | Saga → Episodic | uuid, group_id, created_at |
| `NEXT_EPISODE` | Episodic → Episodic | uuid, group_id, created_at |

## Search Requests

### CosineSimilaritySearch
```protobuf
message VectorSearchRequest {
  repeated double embedding = 1;
  string group_id = 2;
  int32 limit = 3;
  string index_name = 4;  // "entities", "entity_edges", "communities"
}
```

### FulltextSearch (BM25)
```protobuf
message TextSearchRequest {
  string query = 1;
  string group_id = 2;
  int32 limit = 3;
  string index_name = 4;
}
```

### BFSSearch (Graph Traversal)
```protobuf
message BFSSearchRequest {
  string start_node_id = 1;
  int32 max_depth = 2;    // Default: 2
  string group_id = 3;
  int32 limit = 4;
}
```
