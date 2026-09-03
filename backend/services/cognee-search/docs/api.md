---
id: DOC-S02
service: cognee-search
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-search — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9013

## gRPC Service Definition

```protobuf
service CogneeSearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

## Endpoints

| RPC Method | Request | Response | Description |
|-----------|---------|----------|-------------|
| `Search` | `SearchRequest` | `SearchResponse` | Multi-strategy search over KG + vectors |
| `RAGComplete` | `RAGRequest` | `RAGResponse` | RAG completion with LLM |
| `GetChunks` | `GetChunksRequest` | `ChunksResponse` | Direct chunk retrieval by IDs |

## REST Routes (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/cognee/search` | Search knowledge |
| POST | `/v1/cognee/rag` | RAG completion |

## Request Schemas

### SearchRequest

```protobuf
message SearchRequest {
  string query = 1;
  SearchType search_type = 2;     // GRAPH_COMPLETION (default), SIMILARITY, etc.
  repeated string dataset_ids = 3; // Filter by datasets (empty = all)
  int32 top_k = 4;                // Max results (default: 10)
  string system_prompt = 5;       // Custom system prompt for LLM types
  bool only_context = 6;          // Return context only, no LLM completion
  SearchConfig config = 7;        // Strategy-specific config
}

message SearchConfig {
  int32 neighborhood_depth = 1;    // Graph traversal depth
  int32 neighborhood_seed_top_k = 2;
  float triplet_distance_penalty = 3;  // Default: 6.5
  int32 wide_search_top_k = 4;    // Default: 100
}
```

## Authentication

All requests require `x-tenant-id` gRPC metadata.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Dataset not found |
| `INVALID_ARGUMENT` | 400 | Invalid search type or parameters |
| `FAILED_PRECONDITION` | 412 | No cognified data available |
| `INTERNAL` | 500 | Search execution error |

## NATS Events

| Direction | Subject | Peer |
|-----------|---------|------|
| Subscribe | `cognee.pipeline.completed` | cognee-cognify |
