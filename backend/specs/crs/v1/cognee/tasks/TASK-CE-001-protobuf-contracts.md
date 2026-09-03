# TASK-CE-001 — Protobuf Contracts (All Cognee Services)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-001 |
| **Wave** | 1 (Foundation) |
| **Component** | `api/proto/cognee/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.3, SOL-002 §2.3/§2.6, SOL-003 §2.5, SOL-006 §2.4 |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Context

Cập nhật 3 proto files cho cognee services để thêm các fields và RPCs mới. Đây là **bước đầu tiên phải hoàn thành** trước khi implement bất kỳ service nào khác — vì code gen từ proto là source of truth.

---

## Goal

- `cognify.proto` — thêm `Memify`, `GetPipelineStatus`, `GetPipelineTemplates` RPC + pipeline config messages
- `ingestion.proto` — thêm `AddDataPoints` RPC + `node_sets` field vào `AddDataRequest`
- `search.proto` — thêm `node_sets`, `save_interaction`, `feedback_*` fields
- Chạy `make proto-cognee` để gen Go code

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `api/proto/cognee/cognify/v1/cognify.proto` |
| MODIFY | `api/proto/cognee/ingestion/v1/ingestion.proto` |
| MODIFY | `api/proto/cognee/search/v1/search.proto` |
| MODIFY | `Makefile` |

---

## Implementation

### File 1: `api/proto/cognee/cognify/v1/cognify.proto` (full update)

```protobuf
syntax = "proto3";
package cognee.cognify.v1;
option go_package = "github.com/vnp-memory/api/proto/cognee/cognify/v1";

import "google/protobuf/timestamp.proto";

service CognifyService {
  // Existing
  rpc StartCognify(StartCognifyRequest) returns (StartCognifyResponse);

  // [NEW] CR-001 — Memify
  rpc Memify(MemifyRequest) returns (MemifyResponse);
  rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse);

  // [NEW] CR-006 — Pipeline Templates
  rpc GetPipelineTemplates(GetPipelineTemplatesRequest) returns (GetPipelineTemplatesResponse);
}

// ── StartCognify (updated) ─────────────────────────────────────────────────

message StartCognifyRequest {
  string           dataset_id  = 1;
  string           tenant_id   = 2;
  repeated string  entry_ids   = 3;
  repeated string  node_sets   = 4;  // [NEW] CR-002
  // [NEW] CR-006 Pipeline config
  string           template    = 5;  // "STANDARD" | "EMBED_ONLY" | "FAST_INDEX" | "TEMPORAL" | "GRAPH_ONLY"
  repeated string  steps       = 6;  // custom step list (overrides template)
  PipelineOptions  options     = 7;
}

message StartCognifyResponse {
  string           pipeline_run_id = 1;
  string           status          = 2;
  repeated string  steps_executed  = 3;  // [NEW] list of steps that ran
}

// ── Pipeline Options (CR-006) ──────────────────────────────────────────────

message PipelineOptions {
  int32  chunk_size     = 1;  // default: 512
  string custom_prompt  = 2;  // override LLM extraction prompt
  bool   temporal_mode  = 3;  // use temporal extraction variant
  bool   skip_cache     = 4;  // force re-run
}

message GetPipelineTemplatesRequest {}
message GetPipelineTemplatesResponse {
  repeated PipelineTemplateInfo templates = 1;
}
message PipelineTemplateInfo {
  string          name  = 1;
  repeated string steps = 2;
}

// ── Memify (CR-001) ────────────────────────────────────────────────────────

message MemifyRequest {
  string         dataset_id = 1;
  string         tenant_id  = 2;
  optional MemifyConfig config = 3;
}

message MemifyConfig {
  bool  derive_facts    = 1;  // default: true
  bool  embed_triplets  = 2;  // default: true
  int32 batch_size      = 3;  // default: 50
}

message MemifyResponse {
  string pipeline_run_id = 1;
  string status          = 2;  // QUEUED | RUNNING | COMPLETED | FAILED
}

// ── Pipeline Status (CR-001) ───────────────────────────────────────────────

message GetPipelineStatusRequest {
  string pipeline_run_id = 1;
  string dataset_id      = 2;  // alternative lookup
}

message GetPipelineStatusResponse {
  string pipeline_run_id = 1;
  string status          = 2;  // QUEUED | RUNNING | COMPLETED | FAILED
  int32  new_nodes       = 3;
  int32  new_edges       = 4;
  string error           = 5;
  google.protobuf.Timestamp completed_at = 6;
}
```

### File 2: `api/proto/cognee/ingestion/v1/ingestion.proto` (additions)

```protobuf
syntax = "proto3";
package cognee.ingestion.v1;
option go_package = "github.com/vnp-memory/api/proto/cognee/ingestion/v1";

import "google/protobuf/struct.proto";

service IngestionService {
  rpc AddData(AddDataRequest) returns (AddDataResponse);
  rpc AddDataPoints(AddDataPointsRequest) returns (AddDataPointsResponse);  // [NEW] CR-003
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc ListDataEntries(ListDataEntriesRequest) returns (ListDataEntriesResponse);  // [NEW] for MCP list_data
  rpc DeleteDataset(DeleteDatasetRequest) returns (DeleteDatasetResponse);
}

// ── AddData (updated) ──────────────────────────────────────────────────────

message AddDataRequest {
  string            dataset_id   = 1;
  string            dataset_name = 2;
  string            tenant_id    = 3;
  repeated DataItem items        = 4;
  repeated string   node_sets    = 5;  // [NEW] CR-002
}

message DataItem {
  string content      = 1;
  string url          = 2;
  string content_type = 3;  // TEXT | PDF | PDF_LAYOUT | HTML | URL | DOCX | CSV | TABULAR_FK
  map<string, string> metadata = 4;
  DataItemConfig config = 5;  // [NEW] CR-004
}

message DataItemConfig {
  string pdf_mode = 1;  // "LAYOUT_AWARE" | "PLAIN_TEXT"
}

message AddDataResponse {
  string          dataset_id = 1;
  repeated string entry_ids  = 2;
  int32           count      = 3;
}

// ── AddDataPoints (CR-003) ─────────────────────────────────────────────────

message AddDataPointsRequest {
  string             dataset_id  = 1;
  string             tenant_id   = 2;
  repeated DataPoint data_points = 3;
  repeated string    node_sets   = 4;
}

message DataPoint {
  string                 id           = 1;  // stable deterministic UUID
  string                 type         = 2;  // e.g. "Paper", "Employee", "Product"
  google.protobuf.Struct fields       = 3;  // all field values
  repeated string        index_fields = 4;  // fields to embed into Qdrant
  repeated Relation      relations    = 5;  // explicit FK edges
}

message Relation {
  string target_id = 1;
  string label     = 2;   // edge label: "authored_by", "works_in", ...
  double weight    = 3;   // default: 1.0
}

message AddDataPointsResponse {
  int32 upserted = 1;
  int32 created  = 2;
  int32 updated  = 3;
}

// ── List/Delete ────────────────────────────────────────────────────────────

message ListDatasetsRequest {
  string tenant_id = 1;
  int32  limit     = 2;
  int32  offset    = 3;
}

message DatasetInfo {
  string dataset_id   = 1;
  string dataset_name = 2;
  int32  entry_count  = 3;
  string created_at   = 4;
}

message ListDatasetsResponse { repeated DatasetInfo datasets = 1; }

message ListDataEntriesRequest {
  string tenant_id    = 1;
  string dataset_id   = 2;
  string dataset_name = 3;
  int32  limit        = 4;
  int32  offset       = 5;
}

message DataEntryInfo {
  string entry_id     = 1;
  string content_type = 2;
  string created_at   = 3;
  int32  chunk_count  = 4;
}

message ListDataEntriesResponse { repeated DataEntryInfo entries = 1; }

message DeleteDatasetRequest {
  string tenant_id    = 1;
  string dataset_id   = 2;
  string dataset_name = 3;
}

message DeleteDatasetResponse { string dataset_id = 1; bool deleted = 2; }
```

### File 3: `api/proto/cognee/search/v1/search.proto` (additions)

```protobuf
syntax = "proto3";
package cognee.search.v1;
option go_package = "github.com/vnp-memory/api/proto/cognee/search/v1";

service SearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc ListInteractions(ListInteractionsRequest) returns (ListInteractionsResponse);  // [NEW] CR-005
}

message SearchRequest {
  string           query            = 1;
  repeated string  strategies       = 2;
  string           dataset_id       = 3;
  string           dataset_name     = 4;
  string           tenant_id        = 5;
  repeated string  node_sets        = 6;  // [NEW] CR-002
  int32            top_k            = 7;
  // [NEW] CR-005 Feedback fields
  bool             save_interaction = 8;
  string           session_id       = 9;
  string           feedback_for     = 10;  // interaction UUID — triggers FEEDBACK strategy
  optional double  feedback_score   = 11;  // -1.0 to 1.0
  string           feedback_text    = 12;
}

message SearchResult {
  string id          = 1;
  string content     = 2;
  double score       = 3;
  map<string, string> metadata = 4;
}

message SearchResponse {
  repeated SearchResult results        = 1;
  optional string       interaction_id = 2;  // [NEW] present when save_interaction=true
  map<string, string>   metadata       = 3;  // e.g. {"applied": "true", "affected_nodes": "5"}
}

// ── Interactions (CR-005) ──────────────────────────────────────────────────

message ListInteractionsRequest {
  string tenant_id  = 1;
  string session_id = 2;
  int32  limit      = 3;
  int32  offset     = 4;
}

message InteractionInfo {
  string          id         = 1;
  string          query      = 2;
  string          strategy   = 3;
  repeated string result_ids = 4;
  string          timestamp  = 5;
}

message ListInteractionsResponse { repeated InteractionInfo interactions = 1; }
```

### MODIFY `Makefile` — add proto-cognee target

```makefile
proto-cognee:
	@echo "Compiling cognee protobufs..."
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       api/proto/cognee/cognify/v1/cognify.proto \
	       api/proto/cognee/ingestion/v1/ingestion.proto \
	       api/proto/cognee/search/v1/search.proto

.PHONY: proto-cognee
```

---

## Verification

```bash
make proto-cognee

# Check generated files
ls api/proto/cognee/*/v1/*.pb.go
ls api/proto/cognee/*/v1/*_grpc.pb.go

# Verify compile
go build ./...
```

**Expected:** All `.pb.go` + `_grpc.pb.go` generated. No compile errors.

---

## Acceptance Criteria

| Check | Expected |
|-------|----------|
| `CognifyService.Memify` RPC exists | ✅ |
| `CognifyService.GetPipelineStatus` RPC exists | ✅ |
| `CognifyService.GetPipelineTemplates` RPC exists | ✅ |
| `IngestionService.AddDataPoints` RPC exists | ✅ |
| `AddDataRequest.node_sets` field exists | ✅ |
| `SearchRequest.node_sets` field exists | ✅ |
| `SearchRequest.save_interaction` field exists | ✅ |
| `SearchRequest.feedback_for` field exists | ✅ |
| `go build ./...` passes | ✅ |
