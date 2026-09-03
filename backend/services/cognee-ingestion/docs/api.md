---
id: DOC-S02
service: cognee-ingestion
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-ingestion — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9011
> **Proto Path**: `api/proto/cognee/ingestion/v1/service.proto`

## gRPC Service Definition

```protobuf
service CogneeIngestionService {
  // Dataset management
  rpc CreateDataset(CreateDatasetRequest) returns (Dataset);
  rpc DeleteDataset(DeleteDatasetRequest) returns (google.protobuf.Empty);
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc GetDatasetStatus(GetDatasetStatusRequest) returns (DatasetStatus);

  // Data ingestion
  rpc AddData(stream AddDataRequest) returns (AddDataResponse);
  rpc AddText(AddTextRequest) returns (AddTextResponse);
  rpc AddUrl(AddUrlRequest) returns (AddUrlResponse);
}
```

## Endpoints

| RPC Method | Request | Response | Description |
|-----------|---------|----------|-------------|
| `CreateDataset` | `CreateDatasetRequest` | `Dataset` | Create a new dataset namespace for a tenant |
| `DeleteDataset` | `DeleteDatasetRequest` | `Empty` | Delete a dataset and all associated data |
| `ListDatasets` | `ListDatasetsRequest` | `ListDatasetsResponse` | List all datasets for the current tenant |
| `GetDatasetStatus` | `GetDatasetStatusRequest` | `DatasetStatus` | Get dataset status (PENDING/READY/COGNIFYING/ERROR) |
| `AddData` | `stream AddDataRequest` | `AddDataResponse` | Upload file data via streaming (PDF/DOCX/PPTX/CSV) |
| `AddText` | `AddTextRequest` | `AddTextResponse` | Ingest direct text content |
| `AddUrl` | `AddUrlRequest` | `AddUrlResponse` | Ingest content from URL via web scraping |

## REST Routes (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/cognee/datasets` | Create dataset |
| POST | `/v1/cognee/datasets/{id}/data` | Upload data to dataset |
| DELETE | `/v1/cognee/datasets/{id}` | Delete dataset |
| GET | `/v1/cognee/datasets` | List datasets |
| GET | `/v1/cognee/datasets/{id}/status` | Get dataset status |

## Request/Response Schemas

### CreateDatasetRequest

```protobuf
message CreateDatasetRequest {
  string name = 1;                    // Dataset name (unique per tenant)
  map<string, string> metadata = 2;   // Optional metadata key-value pairs
}
```

### AddDataRequest (streaming)

```protobuf
message AddDataRequest {
  string dataset_id = 1;             // Target dataset UUID
  string filename = 2;               // Original filename
  string mime_type = 3;              // MIME type (application/pdf, etc.)
  bytes chunk = 4;                   // File data chunk (max 64KB per message)
  map<string, string> metadata = 5;  // Optional item-level metadata
}
```

### AddTextRequest

```protobuf
message AddTextRequest {
  string dataset_id = 1;             // Target dataset UUID
  string content = 2;                // Text content to ingest
  string source_name = 3;            // Source identifier
  map<string, string> metadata = 4;  // Optional metadata
}
```

### AddUrlRequest

```protobuf
message AddUrlRequest {
  string dataset_id = 1;             // Target dataset UUID
  string url = 2;                    // URL to scrape and ingest
  map<string, string> metadata = 3;  // Optional metadata
}
```

## Authentication

All requests require `x-tenant-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Dataset not found |
| `ALREADY_EXISTS` | 409 | Dataset with same name already exists for tenant |
| `INVALID_ARGUMENT` | 400 | Invalid request parameters (empty name, bad MIME type) |
| `RESOURCE_EXHAUSTED` | 429 | Upload size limit exceeded |
| `INTERNAL` | 500 | Internal server error |
| `UNAVAILABLE` | 503 | Service unavailable |

## NATS Events Published

| Subject | Payload | Subscriber |
|---------|---------|------------|
| `cognee.data.ingested` | `{dataset_id, tenant_id, item_ids[]}` | cognee-cognify |

## Rate Limiting

- File upload: 100MB max per file
- Text ingestion: 1MB max per request
- URL scraping: 10 concurrent scrapes per tenant
