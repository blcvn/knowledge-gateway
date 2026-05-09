---
id: DOC-S02
service: graphiti-knowledge
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-knowledge — API Reference

## gRPC Service Definition

```protobuf
service GraphitiKnowledgeService {
  // Extract named entities from episode content using LLM
  rpc ExtractEntities(ExtractEntitiesRequest) returns (ExtractEntitiesResponse);
  // Resolve/deduplicate extracted entities against existing graph
  rpc ResolveEntities(ResolveEntitiesRequest) returns (ResolveEntitiesResponse);
  // Extract relationships/facts from episode with temporal markers
  rpc ExtractEdges(ExtractEdgesRequest) returns (ExtractEdgesResponse);
  // Resolve/deduplicate edges, invalidate contradicted facts
  rpc ResolveEdges(ResolveEdgesRequest) returns (ResolveEdgesResponse);
  // Generate vector embedding for text
  rpc GenerateEmbedding(EmbeddingRequest) returns (EmbeddingResponse);
  // Cross-encoder reranking for search results
  rpc Rerank(RerankRequest) returns (RerankResponse);
  // Update community structure after graph changes
  rpc UpdateCommunity(UpdateCommunityRequest) returns (UpdateCommunityResponse);
}
```

## RPC: ExtractEntities

```protobuf
message ExtractEntitiesRequest {
  string content = 1;                     // Episode content to extract from
  string group_id = 2;                    // Tenant partition
  repeated string previous_episodes = 3;  // Context from previous episodes
  map<string, string> entity_types = 4;   // Optional type constraints
  repeated string excluded_types = 5;     // Types to exclude
}

message ExtractEntitiesResponse {
  repeated EntityNode entities = 1;
  // Each entity: {uuid, name, type, summary, labels, group_id}
}
```

## RPC: ResolveEntities

```protobuf
message ResolveEntitiesRequest {
  repeated EntityNode extracted = 1;       // Newly extracted entities
  string group_id = 2;
  EpisodicNode episode = 3;               // Source episode context
  repeated EpisodicNode previous = 4;     // Previous episodes for context
}

message ResolveEntitiesResponse {
  repeated EntityNode resolved = 1;        // Resolved (deduped) entities
  map<string, string> uuid_map = 2;        // Old UUID → resolved UUID mapping
  repeated DuplicatePair duplicates = 3;   // Pairs identified as duplicates
}
```

## RPC: ExtractEdges

```protobuf
message ExtractEdgesRequest {
  EpisodicNode episode = 1;
  repeated EntityNode resolved_nodes = 2;
  repeated EpisodicNode previous = 3;
  string group_id = 4;
  map<string, string> edge_types = 5;     // Optional type constraints
}

message ExtractEdgesResponse {
  repeated EntityEdge edges = 1;
  // Each edge: {uuid, name, fact, source_uuid, target_uuid, valid_at, invalid_at, attributes}
}
```

## RPC: Rerank

```protobuf
message RerankRequest {
  string query = 1;
  repeated string documents = 2;          // Texts to rerank
  int32 top_k = 3;                        // Number of results to return
}

message RerankResponse {
  repeated float scores = 1;              // Relevance scores
  repeated int32 indices = 2;             // Original indices sorted by relevance
}
```

## NATS Events Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `graphiti.entity.resolved` | `{group_id, resolved_count, duplicate_count}` | After entity resolution |
| `graphiti.community.rebuilt` | `{group_id, community_id, member_count}` | After community update |
