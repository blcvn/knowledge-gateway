# Graphiti Services

---

# Graphiti Ingestion Service

> **Service**: `graphiti-ingestion` | **gRPC Port**: 9021

## 1. Responsibility

Episode lifecycle management, pipeline orchestration via saga pattern. Serializes per-group ingestion for consistency.

## 2. gRPC API

```protobuf
service GraphitiIngestionService {
  rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);
  rpc BulkIngest(stream BulkIngestRequest) returns (BulkIngestResponse);
  rpc GetEpisodeStatus(GetStatusRequest) returns (EpisodeStatus);
}
```

## 3. Pipeline (Saga)

```
1. Validate + enqueue (per group_id serialization)
2. → knowledge.ExtractEntities(content)
3. → knowledge.ResolveEntities(extracted, group_id)
4. → knowledge.ExtractEdges(episode, resolved_nodes)
5. → knowledge.ResolveEdges(extracted_edges, group_id)
6. → store.SaveBulk(nodes, edges, episode)
7. → knowledge.UpdateCommunity(affected_entities)
8. Emit: graphiti.episode.ingested
```

## 4. Compensating Actions

| Step Failed | Compensation |
|-------------|-------------|
| ExtractEntities | Mark episode FAILED, retry |
| SaveBulk | Rollback partial writes |
| UpdateCommunity | Queue for async retry |

---

# Graphiti Search Service

> **Service**: `graphiti-search` | **gRPC Port**: 9022

## 1. Responsibility

Hybrid search over temporal knowledge graph: vector similarity + full-text + BFS graph traversal + reranking.

## 2. gRPC API

```protobuf
service GraphitiSearchService {
  rpc HybridSearch(SearchRequest) returns (SearchResponse);
  rpc NodeSearch(NodeSearchRequest) returns (NodeSearchResponse);
  rpc EdgeSearch(EdgeSearchRequest) returns (EdgeSearchResponse);
  rpc CommunitySearch(CommunitySearchRequest) returns (CommunitySearchResponse);
}
```

## 3. Search Pipeline

```
1. Generate embedding (→ knowledge.GenerateEmbedding)
2. Parallel:
   ├── store.CosineSimilaritySearch(embedding)
   ├── store.FulltextSearch(query)
   └── store.BFSSearch(matched_nodes)
3. Merge + RRF/MMR/CrossEncoder rerank
4. Apply temporal + property filters
5. Return ranked results
```

---

# Graphiti Knowledge Service

> **Service**: `graphiti-knowledge` | **gRPC Port**: 9023

## 1. Responsibility

LLM-intensive processing: entity extraction, edge extraction, entity/edge resolution, community detection, embedding generation.

## 2. gRPC API

```protobuf
service GraphitiKnowledgeService {
  rpc ExtractEntities(ExtractEntitiesRequest) returns (ExtractEntitiesResponse);
  rpc ResolveEntities(ResolveEntitiesRequest) returns (ResolveEntitiesResponse);
  rpc ExtractEdges(ExtractEdgesRequest) returns (ExtractEdgesResponse);
  rpc ResolveEdges(ResolveEdgesRequest) returns (ResolveEdgesResponse);
  rpc GenerateEmbedding(EmbeddingRequest) returns (EmbeddingResponse);
  rpc Rerank(RerankRequest) returns (RerankResponse);
  rpc UpdateCommunity(UpdateCommunityRequest) returns (UpdateCommunityResponse);
}
```

## 3. LLM Integration

| Operation | Model | Structured Output |
|-----------|-------|-------------------|
| ExtractEntities | GPT-4o | JSON Schema: `{name, type, summary}[]` |
| ResolveEntities | GPT-4o-mini | JSON Schema: `{source_id, target_id, is_duplicate}[]` |
| ExtractEdges | GPT-4o | JSON Schema: `{source, target, type, fact, temporal}[]` |
| Rerank | Cross-encoder | Relevance scores |

---

# Graphiti Store Service

> **Service**: `graphiti-store` | **gRPC Port**: 9024

## 1. Responsibility

Graph database abstraction layer. Supports pluggable backends (Neo4j, FalkorDB, Kuzu, Neptune). All graph CRUD, transactions, index management.

## 2. gRPC API

```protobuf
service GraphitiStoreService {
  rpc SaveNode(SaveNodeRequest) returns (Node);
  rpc SaveEdge(SaveEdgeRequest) returns (Edge);
  rpc SaveBulk(SaveBulkRequest) returns (SaveBulkResponse);
  rpc GetNode(GetNodeRequest) returns (Node);
  rpc GetEdge(GetEdgeRequest) returns (Edge);
  rpc DeleteNode(DeleteNodeRequest) returns (Empty);
  rpc CosineSimilaritySearch(VectorSearchRequest) returns (SearchResponse);
  rpc FulltextSearch(TextSearchRequest) returns (SearchResponse);
  rpc BFSSearch(BFSSearchRequest) returns (SearchResponse);
}
```

## 3. Driver Interface

```go
type GraphDriver interface {
    SaveNode(ctx context.Context, node *graph.EntityNode) error
    SaveEdge(ctx context.Context, edge *graph.EntityEdge) error
    GetNode(ctx context.Context, id string, groupID string) (*graph.EntityNode, error)
    CosineSimilarity(ctx context.Context, embedding []float64, groupID string, limit int) ([]*graph.EntityNode, error)
    FulltextSearch(ctx context.Context, query string, groupID string, limit int) ([]*graph.EntityNode, error)
    BFSTraversal(ctx context.Context, startNodeID string, depth int) ([]*graph.EntityNode, error)
}
```

## 4. Implementations

| Backend | Status | Isolation |
|---------|--------|-----------|
| Neo4j 5.x | Primary | Property filter `group_id` |
| FalkorDB | Pluggable | Separate graph per `group_id` |
| Kuzu | Pluggable | Separate database per `group_id` |
