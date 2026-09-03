---
id: DOC-S02
service: ov-resource
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-resource — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9054

## gRPC Service Definition

```protobuf
// api/proto/openviking/resource/v1/service.proto
service OvResourceService {
  rpc Ingest(IngestRequest) returns (IngestResponse);
  rpc Parse(ParseRequest) returns (ParseResponse);
  rpc Watch(WatchRequest) returns (stream WatchEvent);
  rpc Refresh(RefreshRequest) returns (RefreshResponse);
}
```

## Endpoints

### Ingest

Full ingestion pipeline: parse → chunk → write to ov-fs → notify ov-search.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | bytes | Yes | Raw file content |
| `filename` | string | Yes | Original filename (for parser detection) |
| `path` | string | Yes | Target VikingFS path |
| `account_id` | string | Yes | Tenant account |
| `force_parser` | string | No | Override auto-detected parser |

**Response**: `IngestResponse { chunks_count, total_tokens, path, parse_duration_ms }`

### Parse

Parse-only (no write). Returns structured chunks.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | bytes | Yes | Raw content |
| `filename` | string | Yes | Filename for parser detection |
| `chunk_size` | int32 | No | Target chunk size in tokens |
| `chunk_overlap` | int32 | No | Overlap between chunks |

**Response**: `ParseResponse { chunks: []Chunk }`

### Watch

Stream-based directory watching for auto-ingestion.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source_path` | string | Yes | External directory to watch |
| `target_path` | string | Yes | VikingFS destination path |
| `poll_interval_ms` | int64 | No | Polling interval (default: 30000) |
| `patterns` | []string | No | Glob patterns to include |

**Response (stream)**: `WatchEvent { type: CREATED/MODIFIED/DELETED, path, timestamp }`

### Refresh

Re-parse and re-index stale resources.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `paths` | []string | Yes | Paths to refresh |
| `force` | bool | No | Re-parse even if not stale |

## Parse Engine Details

| Format | Parser | Chunk Strategy | Avg Chunk Size |
|--------|--------|---------------|----------------|
| `.go`, `.py`, `.js`, `.ts` | tree-sitter | AST-aware (function/class) | ~500 tokens |
| `.md` | Custom | Section-based (H1-H6) | ~800 tokens |
| `.pdf`, `.docx` | Go libraries | Page-based + overlap | ~1000 tokens |
| `.txt` | Default | Paragraph (double newline) | ~500 tokens |

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `INVALID_ARGUMENT` | 400 | Unsupported file format |
| `NOT_FOUND` | 404 | Watch source path not found |
| `RESOURCE_EXHAUSTED` | 429 | Ingestion queue full |
| `INTERNAL` | 500 | Parser or write failure |
