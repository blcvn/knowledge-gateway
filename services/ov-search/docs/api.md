---
id: DOC-S02
service: ov-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-search — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9052

## gRPC Service Definition

```protobuf
// api/proto/openviking/search/v1/service.proto
service OvSearchService {
  rpc HierarchicalSearch(SearchRequest) returns (SearchResponse);
  rpc RetrieveContext(ContextRequest) returns (ContextResponse);
  rpc GetHotness(HotnessRequest) returns (HotnessResponse);
  rpc UpsertEmbedding(UpsertRequest) returns (google.protobuf.Empty);
  rpc DeleteEmbedding(DeleteRequest) returns (google.protobuf.Empty);
}
```

## Endpoints

### HierarchicalSearch

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `query` | string | Yes | Natural language search query |
| `account_id` | string | Yes | Tenant account |
| `max_results` | int32 | No | Max results (default: 10) |
| `context_level` | ContextLevel | No | Requested context depth (L0/L1/L2) |
| `enable_hotness` | bool | No | Apply hotness scoring boost (default: true) |
| `rerank_strategy` | RerankStrategy | No | RRF / MMR / CROSS_ENCODER (default: RRF) |

**Response**: `SearchResponse { results: []SearchResult, metadata: SearchMetadata }`

### RetrieveContext

Retrieves context for a specific file with tiered loading.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | VikingFS file path |
| `context_level` | ContextLevel | Yes | L0, L1, or L2 |

### GetHotness

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `paths` | []string | Yes | File paths to query hotness |

**Response**: `HotnessResponse { scores: map<string, float64> }`

### UpsertEmbedding

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | File path |
| `content` | string | Yes | Text content to embed |
| `metadata` | map | No | Additional metadata |

### DeleteEmbedding

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | File path to remove from index |

## Hotness Scoring Algorithm

```
hotness(file) = base_score × decay(time_since_last_access) + session_boost
decay(t) = exp(-λ × t)    // configurable half-life (default: 24h)
session_boost = 0.3 × count(recent_session_references)
```

## Authentication

All requests require `x-tenant-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | File not found in index |
| `INVALID_ARGUMENT` | 400 | Invalid query or parameters |
| `INTERNAL` | 500 | Embedding or search engine error |
| `UNAVAILABLE` | 503 | Vector DB or Bifrost unavailable |
