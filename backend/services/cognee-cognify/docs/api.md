---
id: DOC-S02
service: cognee-cognify
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-cognify — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9012
> **Proto Path**: `api/proto/cognee/cognify/v1/service.proto`

## gRPC Service Definition

```protobuf
service CogneeCognifyService {
  rpc TriggerCognify(TriggerCognifyRequest) returns (CognifyJob);
  rpc GetJobStatus(GetJobStatusRequest) returns (CognifyJob);
  rpc CancelJob(CancelJobRequest) returns (google.protobuf.Empty);
}
```

## Endpoints

| RPC Method | Request | Response | Description |
|-----------|---------|----------|-------------|
| `TriggerCognify` | `TriggerCognifyRequest` | `CognifyJob` | Trigger KG construction pipeline for a dataset |
| `GetJobStatus` | `GetJobStatusRequest` | `CognifyJob` | Query current pipeline job status and progress |
| `CancelJob` | `CancelJobRequest` | `Empty` | Cancel a running cognify job |

## REST Routes (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/cognee/datasets/{id}/cognify` | Trigger KG pipeline |
| GET | `/v1/cognee/jobs/{id}` | Get job status |
| DELETE | `/v1/cognee/jobs/{id}` | Cancel job |

## Request/Response Schemas

### TriggerCognifyRequest

```protobuf
message TriggerCognifyRequest {
  string dataset_id = 1;              // Dataset to process
  CognifyConfig config = 2;           // Optional pipeline configuration
}

message CognifyConfig {
  string chunking_strategy = 1;       // "recursive" | "ast" | "paragraph" | "sentence"
  int32 chunk_size = 2;               // Target chunk size in tokens (default: 512)
  int32 chunk_overlap = 3;            // Chunk overlap in tokens (default: 50)
  string llm_model = 4;               // LLM model for extraction (default: from config)
  bool skip_dedup = 5;                // Skip entity deduplication (default: false)
  bool skip_summarize = 6;            // Skip community summarization (default: false)
}
```

### CognifyJob

```protobuf
message CognifyJob {
  string job_id = 1;
  string dataset_id = 2;
  string tenant_id = 3;
  JobStatus status = 4;               // PENDING | RUNNING | COMPLETED | FAILED | CANCELLED
  string current_stage = 5;           // classify | chunk | extract | dedup | build | embed | summarize
  float progress_percent = 6;         // 0.0 - 100.0
  string error_message = 7;           // Error details if FAILED
  google.protobuf.Timestamp started_at = 8;
  google.protobuf.Timestamp completed_at = 9;
  PipelineMetrics metrics = 10;
}

message PipelineMetrics {
  int32 chunks_created = 1;
  int32 entities_extracted = 2;
  int32 relationships_extracted = 3;
  int32 entities_deduplicated = 4;
  int32 communities_found = 5;
  int32 embeddings_generated = 6;
}
```

## Pipeline Stages

```
classify → chunk → extract_entities → extract_relationships
    → deduplicate → build_graph(Neo4j) → embed(Qdrant)
    → summarize_communities
```

## LLM Integration

| Stage | LLM Call | Model | Purpose |
|-------|---------|-------|---------|
| Classify | Content type detection | Fast model (GPT-4o-mini) | Determine chunking strategy |
| Extract entities | NER + relation extraction | GPT-4o | Identify entities and relationships |
| Deduplicate | Entity resolution | GPT-4o-mini | Merge duplicate entities |
| Summarize | Community summaries | GPT-4o-mini | Generate graph community overviews |

## Authentication

All requests require `x-tenant-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Dataset or job not found |
| `ALREADY_EXISTS` | 409 | Cognify job already running for dataset |
| `INVALID_ARGUMENT` | 400 | Invalid configuration parameters |
| `FAILED_PRECONDITION` | 412 | Dataset has no data items to process |
| `INTERNAL` | 500 | Pipeline execution error |
| `UNAVAILABLE` | 503 | Service unavailable |

## NATS Events

| Direction | Subject | Payload | Peer |
|-----------|---------|---------|------|
| **Subscribe** | `cognee.data.ingested` | `{dataset_id, tenant_id, item_ids[]}` | cognee-ingestion |
| **Publish** | `cognee.pipeline.completed` | `{dataset_id, tenant_id, job_id, metrics}` | cognee-search |
