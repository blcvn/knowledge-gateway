---
id: DOC-S02
service: memobase-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-ingestion — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9031

## gRPC Service Definition

```protobuf
// api/proto/memobase/ingestion/v1/service.proto
service MemobaseIngestionService {
  // Insert a blob (chat, doc, or summary) into the buffer zone
  rpc InsertBlob(InsertBlobRequest) returns (InsertBlobResponse);

  // Get current buffer status for a user
  rpc GetBufferStatus(BufferStatusRequest) returns (BufferStatus);

  // Force-flush the buffer zone for a user
  rpc FlushBuffer(FlushBufferRequest) returns (FlushResponse);

  // Delete a specific blob by ID
  rpc DeleteBlob(DeleteBlobRequest) returns (google.protobuf.Empty);
}
```

## Messages

### InsertBlobRequest

```protobuf
message InsertBlobRequest {
  string user_id = 1;
  string project_id = 2;          // Tenant isolation key
  BlobType blob_type = 3;          // CHAT | DOC | SUMMARY
  google.protobuf.Struct blob_data = 4;  // JSON blob payload
  bool persistent = 5;             // If false, blob deleted after processing
}

enum BlobType {
  BLOB_TYPE_UNSPECIFIED = 0;
  BLOB_TYPE_CHAT = 1;              // ChatBlob: messages[{role, content}]
  BLOB_TYPE_DOC = 2;               // DocBlob: document content
  BLOB_TYPE_SUMMARY = 3;           // SummaryBlob: pre-summarized content
}
```

### InsertBlobResponse

```protobuf
message InsertBlobResponse {
  string blob_id = 1;              // UUID of stored blob
  bool buffer_flushed = 2;         // True if flush was triggered
  int32 buffer_token_count = 3;    // Current buffer token sum
}
```

### BufferStatus

```protobuf
message BufferStatus {
  string user_id = 1;
  string project_id = 2;
  int32 idle_count = 3;            // Blobs in IDLE state
  int32 processing_count = 4;      // Blobs in PROCESSING state
  int32 failed_count = 5;          // Blobs in FAILED state
  int32 total_tokens = 6;          // Sum of token_size for IDLE blobs
  int32 threshold = 7;             // Flush threshold (default: 1024)
}
```

## Endpoints Summary

| RPC Method | Request | Response | Description |
|-----------|---------|----------|-------------|
| InsertBlob | InsertBlobRequest | InsertBlobResponse | Store blob + buffer entry, auto-flush if threshold reached |
| GetBufferStatus | BufferStatusRequest | BufferStatus | Query buffer zone state for a user |
| FlushBuffer | FlushBufferRequest | FlushResponse | Force-flush idle buffer entries |
| DeleteBlob | DeleteBlobRequest | Empty | Remove a specific blob |

## Authentication

All requests require `x-tenant-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `INVALID_ARGUMENT` | 400 | Invalid blob_type or missing required fields |
| `NOT_FOUND` | 404 | User or blob not found |
| `ALREADY_EXISTS` | 409 | Duplicate blob ID |
| `RESOURCE_EXHAUSTED` | 429 | Rate limit exceeded |
| `INTERNAL` | 500 | Internal server error |
| `UNAVAILABLE` | 503 | Service unavailable |

## Example Usage

### Insert ChatBlob

```bash
grpcurl -d '{
  "user_id": "user-123",
  "project_id": "proj-456",
  "blob_type": "BLOB_TYPE_CHAT",
  "blob_data": {
    "messages": [
      {"role": "user", "content": "I love Thai food"},
      {"role": "assistant", "content": "Great taste! Thai cuisine is excellent."}
    ]
  }
}' localhost:9031 memobase.ingestion.v1.MemobaseIngestionService/InsertBlob
```
