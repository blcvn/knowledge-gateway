---
id: DOC-S02
service: graphiti-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-ingestion — API Reference

> **Proto**: `api/proto/graphiti/ingestion/v1/service.proto`

## gRPC Service Definition

```protobuf
service GraphitiIngestionService {
  // Ingest a single episode into the temporal knowledge graph
  rpc IngestEpisode(IngestEpisodeRequest) returns (IngestEpisodeResponse);

  // Stream bulk episode ingestion for batch processing
  rpc BulkIngest(stream BulkIngestRequest) returns (BulkIngestResponse);

  // Query the processing status of an episode
  rpc GetEpisodeStatus(GetStatusRequest) returns (EpisodeStatus);
}
```

---

## RPC: IngestEpisode

Ingest a single episode into the Graphiti temporal knowledge graph. Triggers the full saga pipeline: entity extraction → entity resolution → edge extraction → edge resolution → bulk save → community update.

### Request

```protobuf
message IngestEpisodeRequest {
  string name = 1;                    // Episode name/identifier
  string group_id = 2;               // Tenant/partition isolation key
  string content = 3;                // Raw episode data (message, JSON, text, fact_triple)
  EpisodeType source = 4;            // Content type classification
  string source_description = 5;     // Description of the data source
  google.protobuf.Timestamp valid_at = 6;  // When the original event occurred
  map<string, string> metadata = 7;  // Customer-defined metadata for filtering

  // Optional: entity/edge type constraints
  repeated string entity_types = 8;  // Restrict extraction to these entity types
  repeated string edge_types = 9;    // Restrict extraction to these edge types

  // Optional: saga association
  string saga_name = 10;             // Group episodes into a named saga
}

enum EpisodeType {
  EPISODE_TYPE_UNSPECIFIED = 0;
  EPISODE_TYPE_MESSAGE = 1;          // Format: "actor: content"
  EPISODE_TYPE_JSON = 2;             // Structured JSON data
  EPISODE_TYPE_TEXT = 3;             // Plain text
  EPISODE_TYPE_FACT_TRIPLE = 4;      // Subject-predicate-object triple
}
```

### Response

```protobuf
message IngestEpisodeResponse {
  string episode_id = 1;             // UUID of the created episode
  string status = 2;                 // QUEUED | PROCESSING | COMPLETED | FAILED
  int32 nodes_created = 3;           // Number of entity nodes created/resolved
  int32 edges_created = 4;           // Number of entity edges created/resolved
  int32 communities_updated = 5;     // Number of communities affected
}
```

### Error Codes

| gRPC Status | Condition |
|------------|-----------|
| `INVALID_ARGUMENT` | Missing required fields (content, group_id) |
| `ALREADY_EXISTS` | Duplicate episode (same name + group_id + valid_at) |
| `RESOURCE_EXHAUSTED` | Rate limit exceeded for tenant |
| `INTERNAL` | Pipeline saga failure (check episode status for details) |

### Example (grpcurl)

```bash
grpcurl -plaintext -d '{
  "name": "conversation-001",
  "group_id": "tenant-abc",
  "content": "user: What is the capital of France?\nassistant: The capital of France is Paris.",
  "source": "EPISODE_TYPE_MESSAGE",
  "source_description": "Chat conversation",
  "valid_at": "2026-05-09T10:00:00Z"
}' localhost:9021 graphiti.ingestion.v1.GraphitiIngestionService/IngestEpisode
```

---

## RPC: BulkIngest

Stream multiple episodes for batch processing. Optimized for bulk data loading with deduplication across the batch.

### Request (streaming)

```protobuf
message BulkIngestRequest {
  string name = 1;
  string group_id = 2;
  string content = 3;
  EpisodeType source = 4;
  string source_description = 5;
  google.protobuf.Timestamp valid_at = 6;
  map<string, string> metadata = 7;
}
```

### Response

```protobuf
message BulkIngestResponse {
  int32 episodes_processed = 1;
  int32 total_nodes = 2;
  int32 total_edges = 3;
  int32 total_communities = 4;
  repeated string failed_episodes = 5;  // Episode names that failed
}
```

---

## RPC: GetEpisodeStatus

Query the processing status of a previously ingested episode.

### Request

```protobuf
message GetStatusRequest {
  string episode_id = 1;
}
```

### Response

```protobuf
message EpisodeStatus {
  string episode_id = 1;
  string status = 2;                 // QUEUED | PROCESSING | COMPLETED | FAILED
  string current_step = 3;           // Current saga step (e.g., "ExtractEntities")
  string error_message = 4;          // Error details if FAILED
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}
```

---

## NATS Events

### Published Events

| Subject | Payload | Trigger |
|---------|---------|---------|
| `graphiti.episode.ingested` | `{episode_id, group_id, nodes_count, edges_count}` | After successful saga completion |

### Subscribed Events

None — this service is the ingestion entry point.

---

## Authentication

All RPCs require valid tenant context via gRPC metadata:
- `x-tenant-id`: Tenant identifier (mapped to `group_id`)
- `authorization`: Bearer JWT or API key

## Rate Limiting

- Default: 100 episodes/min per tenant
- Bulk: 1000 episodes/min per tenant
- Configurable via `vnp-admin` tenant config
