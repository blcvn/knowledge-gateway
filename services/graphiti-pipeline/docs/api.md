---
id: DOC-S02
service: graphiti-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# graphiti-pipeline — API Reference

> **Protocol**: gRPC | **Port**: 9021 | **Health**: 9094

## Services Exposed

This binary exposes **two gRPC services** on the same port:

### 1. GraphitiIngestionService

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `IngestEpisode` | `IngestEpisodeRequest` | `IngestEpisodeResponse` | Single episode ingestion via saga pipeline |
| `BulkIngest` | `stream BulkIngestRequest` | `BulkIngestResponse` | Streaming batch ingestion with cross-episode dedup |
| `GetEpisodeStatus` | `GetStatusRequest` | `EpisodeStatus` | Query saga pipeline status for an episode |
| `ListEpisodes` | `ListEpisodesRequest` | `ListEpisodesResponse` | List episodes by group_id with pagination |
| `RemoveEpisode` | `RemoveEpisodeRequest` | `RemoveEpisodeResponse` | Remove episode with cascade cleanup |

### 2. GraphitiKnowledgeService

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `ExtractEntities` | `ExtractEntitiesRequest` | `ExtractEntitiesResponse` | LLM-based entity extraction from content |
| `ResolveEntities` | `ResolveEntitiesRequest` | `ResolveEntitiesResponse` | Entity deduplication via search + LLM |
| `ExtractEdges` | `ExtractEdgesRequest` | `ExtractEdgesResponse` | LLM-based fact triple extraction |
| `ResolveEdges` | `ResolveEdgesRequest` | `ResolveEdgesResponse` | Edge conflict detection + temporal invalidation |
| `GenerateEmbedding` | `GenerateEmbeddingRequest` | `GenerateEmbeddingResponse` | Vector embedding generation |
| `GenerateEmbeddingBulk` | `GenerateEmbeddingBulkRequest` | `GenerateEmbeddingBulkResponse` | Batch embedding generation |
| `Rerank` | `RerankRequest` | `RerankResponse` | Cross-encoder neural reranking |
| `UpdateCommunity` | `UpdateCommunityRequest` | `UpdateCommunityResponse` | Community detection + LLM summarization |
| `GetTokenUsage` | `GetTokenUsageRequest` | `GetTokenUsageResponse` | LLM token usage statistics |

---

## IngestEpisode — Detailed

### Request

```protobuf
message IngestEpisodeRequest {
  string name = 1;                              // Episode identifier
  string body = 2;                              // Episode content
  EpisodeSource source = 3;                     // message | text | json | fact_triple
  google.protobuf.Timestamp reference_time = 4; // When episode occurred
  string group_id = 5;                          // Tenant partition key
  map<string, string> entity_types = 6;         // Optional ontology constraints
  map<string, string> edge_types = 7;           // Optional relationship constraints
  optional string saga_id = 8;                  // Group episodes into saga
}
```

### Response

```protobuf
message IngestEpisodeResponse {
  string episode_uuid = 1;
  PipelineStats stats = 2;  // entities_extracted, edges_extracted, processing_time_ms
}
```

### Error Codes

| gRPC Status | Condition | Retry? |
|-------------|-----------|--------|
| `INVALID_ARGUMENT` | Missing required fields (name, body, group_id) | No |
| `ALREADY_EXISTS` | Duplicate episode (same name+group_id+valid_at) | No |
| `RESOURCE_EXHAUSTED` | Group queue full (backpressure) | Yes (with backoff) |
| `UNAVAILABLE` | Downstream service (graphiti-store) unavailable | Yes |
| `DEADLINE_EXCEEDED` | Pipeline timeout (default 300s) | Yes |
| `INTERNAL` | LLM error, DB error | Yes |

### Example (grpcurl)

```bash
grpcurl -plaintext -d '{
  "name": "user_message_001",
  "body": "Alice met Bob at the conference in Paris last Tuesday",
  "source": "EPISODE_SOURCE_MESSAGE",
  "reference_time": "2026-05-10T10:00:00Z",
  "group_id": "tenant_abc"
}' localhost:9021 graphiti.ingestion.v1.GraphitiIngestionService/IngestEpisode
```

---

## ExtractEntities — Detailed

### Request

```protobuf
message ExtractEntitiesRequest {
  repeated ContentChunk chunks = 1;               // Content to extract from
  repeated EpisodeContext previous_episodes = 2;   // Context from recent episodes
  map<string, string> entity_types = 3;            // Ontology constraints
  string group_id = 4;
}
```

### Response

```protobuf
message ExtractEntitiesResponse {
  repeated ExtractedEntity entities = 1;
  TokenUsage token_usage = 2;
}

message ExtractedEntity {
  string name = 1;
  string label = 2;
  string summary = 3;
}
```

---

## Health & Metrics

| Endpoint | Protocol | Port | Purpose |
|----------|----------|------|---------|
| `grpc.health.v1.Health/Check` | gRPC | 9021 | Readiness probe |
| `GET /healthz` | HTTP | 9094 | Liveness probe |
| `GET /readyz` | HTTP | 9094 | Readiness (includes downstream checks) |
| `GET /metrics` | HTTP | 9094 | Prometheus metrics |

## Authentication

All RPCs require gRPC metadata:
- `x-tenant-id`: **Required** — maps to Graphiti `group_id`
- `x-request-id`: **Optional** — correlation ID for tracing
- `authorization`: **Required** — Bearer token (validated by vnp-gateway)

## Rate Limiting

Rate limiting is enforced at the vnp-gateway level, not within this service.
