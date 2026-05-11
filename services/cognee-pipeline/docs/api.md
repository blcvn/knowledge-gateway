---
id: DOC-S02
service: cognee-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# cognee-pipeline — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9011
> **Dual Service**: CogneeIngestionService + CogneeCognifyService on same port

## Service 1: CogneeIngestionService

### RPC Methods

| RPC Method | Request | Response | Description |
|-----------|---------|----------|-------------|
| `CreateDataset` | `CreateDatasetRequest` | `Dataset` | Create new dataset for tenant |
| `DeleteDataset` | `DeleteDatasetRequest` | `Empty` | Delete dataset and all items |
| `ListDatasets` | `ListDatasetsRequest` | `ListDatasetsResponse` | List tenant datasets with pagination |
| `GetDatasetStatus` | `GetDatasetStatusRequest` | `DatasetStatusResponse` | Get dataset status and metrics |
| `AddData` | `stream AddDataRequest` | `AddDataResponse` | Streaming file upload + extraction |
| `AddText` | `AddTextRequest` | `AddTextResponse` | Direct text ingestion |
| `AddUrl` | `AddUrlRequest` | `AddUrlResponse` | URL scraping + ingestion |

### REST Routes (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/cognee/datasets` | Create dataset |
| GET | `/v1/cognee/datasets` | List datasets |
| DELETE | `/v1/cognee/datasets/{id}` | Delete dataset |
| GET | `/v1/cognee/datasets/{id}/status` | Get dataset status |
| POST | `/v1/cognee/datasets/{id}/data` | Upload file to dataset |
| POST | `/v1/cognee/datasets/{id}/text` | Add text to dataset |
| POST | `/v1/cognee/datasets/{id}/url` | Add URL to dataset |

### Request/Response Schemas

```protobuf
message CreateDatasetRequest {
  string name = 1;
  string description = 2;
  map<string, string> metadata = 3;
}

message Dataset {
  string id = 1;
  string tenant_id = 2;
  string name = 3;
  string description = 4;
  string status = 5;            // PENDING | READY | COGNIFYING | ERROR
  int32 file_count = 6;
  int64 total_size_bytes = 7;
  map<string, string> metadata = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

message AddDataRequest {
  oneof payload {
    DataHeader header = 1;       // First message: filename, mime_type
    bytes chunk = 2;             // Subsequent: file bytes
  }
}

message AddTextRequest {
  string dataset_id = 1;
  string text = 2;
  string source_name = 3;
  map<string, string> metadata = 4;
}

message AddUrlRequest {
  string dataset_id = 1;
  string url = 2;
  map<string, string> metadata = 3;
}
```

---

## Service 2: CogneeCognifyService

### RPC Methods

| RPC Method | Request | Response | Description |
|-----------|---------|----------|-------------|
| `TriggerCognify` | `TriggerCognifyRequest` | `CognifyJob` | Trigger KG construction pipeline |
| `GetJobStatus` | `GetJobStatusRequest` | `CognifyJob` | Query pipeline job status |
| `CancelJob` | `CancelJobRequest` | `Empty` | Cancel running job |

### REST Routes (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/cognee/datasets/{id}/cognify` | Trigger KG pipeline |
| GET | `/v1/cognee/jobs/{id}` | Get job status |
| DELETE | `/v1/cognee/jobs/{id}` | Cancel job |

### Request/Response Schemas

```protobuf
message TriggerCognifyRequest {
  string dataset_id = 1;
  CognifyConfig config = 2;
}

message CognifyConfig {
  string chunking_strategy = 1;   // recursive | ast | paragraph | sentence
  int32 chunk_size = 2;           // default: 512
  int32 chunk_overlap = 3;        // default: 50
  string llm_model = 4;           // override LLM model
  bool skip_dedup = 5;            // skip entity deduplication
  bool skip_summarize = 6;        // skip community summarization
}

message CognifyJob {
  string job_id = 1;
  string dataset_id = 2;
  string tenant_id = 3;
  string status = 4;             // PENDING | RUNNING | COMPLETED | FAILED | CANCELLED
  string current_stage = 5;
  float progress_percent = 6;
  string error_message = 7;
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

---

## Authentication

All requests require `x-tenant-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Dataset or job not found |
| `ALREADY_EXISTS` | 409 | Duplicate dataset or job already running |
| `INVALID_ARGUMENT` | 400 | Invalid request parameters |
| `FAILED_PRECONDITION` | 412 | Dataset has no data items |
| `INTERNAL` | 500 | Pipeline or extraction error |
| `UNAVAILABLE` | 503 | Service unavailable |

## NATS Events

| Direction | Subject | Payload | Peer |
|-----------|---------|---------|------|
| **Publish** | `cognee.pipeline.completed` | `{dataset_id, tenant_id, job_id, metrics}` | cognee-search |

> **Note**: In consolidated mode, `cognee.data.ingested` is NOT published — ingestion triggers cognify via local function call.
