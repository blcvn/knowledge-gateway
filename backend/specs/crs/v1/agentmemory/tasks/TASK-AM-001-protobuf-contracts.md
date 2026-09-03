# TASK-AM-001 — Protobuf Contracts (All AgentMemory Services)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-001 |
| **Wave** | 1 (Foundation) |
| **Component** | `api/proto/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.2, SOL-002 §2.9, SOL-003 §2.7, SOL-004 §2.7 |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Context

Tạo 4 proto files mới cho AgentMemory services. Đây là **bước đầu tiên phải hoàn thành** vì tất cả services phụ thuộc vào generated code.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `api/proto/observe/v1/observe.proto` |
| CREATE | `api/proto/memory/v1/agentmemory.proto` |
| CREATE | `api/proto/search/v1/observe_search.proto` |
| CREATE | `api/proto/orchestration/v1/orchestration.proto` |
| MODIFY | `Makefile` |

---

## Implementation

### File 1: `api/proto/observe/v1/observe.proto`

```protobuf
syntax = "proto3";
package observe.v1;
option go_package = "github.com/vnp-memory/api/proto/observe/v1";

import "google/protobuf/timestamp.proto";

service ObserveService {
  rpc Observe(ObserveRequest) returns (ObserveResponse);
  rpc StartSession(StartSessionRequest) returns (StartSessionResponse);
  rpc EndSession(EndSessionRequest) returns (EndSessionResponse);
  rpc GetSession(GetSessionRequest) returns (GetSessionResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc GetObservations(GetObservationsRequest) returns (GetObservationsResponse);
  rpc DeleteSession(DeleteSessionRequest) returns (DeleteSessionResponse);
  rpc StreamEvents(StreamEventsRequest) returns (stream StreamEvent);  // server-side streaming
}

// ── Observe ────────────────────────────────────────────────────────────────

message ObserveRequest {
  string session_id         = 1;
  string hook_type          = 2;  // session_start|prompt_submit|pre_tool_use|post_tool_use|...
  string tool_name          = 3;
  bytes  tool_input         = 4;  // JSON
  bytes  tool_output        = 5;  // JSON
  string user_prompt        = 6;
  string assistant_response = 7;
  string agent_id           = 8;
  string tenant_id          = 9;
  google.protobuf.Timestamp timestamp = 10;
  string project            = 11;
}

message ObserveResponse {
  string observation_id    = 1;
  bool   deduplicated      = 2;
  CompressedObservationProto compressed = 3;
  string injected_context  = 4;  // populated when hook=session_start + inject_context=true
  int32  context_tokens    = 5;
}

message CompressedObservationProto {
  string   id         = 1;
  string   obs_type   = 2;
  string   title      = 3;
  string   subtitle   = 4;
  repeated string facts    = 5;
  string   narrative  = 6;
  repeated string concepts = 7;
  repeated string files    = 8;
  double   importance = 9;
  double   confidence = 10;
  string   agent_id   = 11;
  google.protobuf.Timestamp timestamp = 12;
}

// ── Sessions ───────────────────────────────────────────────────────────────

message StartSessionRequest {
  string tenant_id   = 1;
  string project     = 2;
  string cwd         = 3;
  string model       = 4;
  string agent_id    = 5;
  string first_prompt = 6;
}

message StartSessionResponse {
  string session_id = 1;
  string status     = 2;  // active
}

message EndSessionRequest {
  string session_id = 1;
  string tenant_id  = 2;
}

message EndSessionResponse {
  string session_id       = 1;
  string status           = 2;  // completed
  int32  observation_count = 3;
}

message GetSessionRequest  { string session_id = 1; string tenant_id = 2; }
message DeleteSessionRequest { string session_id = 1; string tenant_id = 2; }
message DeleteSessionResponse { bool deleted = 1; }

message SessionProto {
  string   session_id        = 1;
  string   tenant_id         = 2;
  string   project           = 3;
  string   cwd               = 4;
  string   model             = 5;
  string   agent_id          = 6;
  string   status            = 7;
  string   summary           = 8;
  int32    observation_count = 9;
  repeated string tags       = 10;
  google.protobuf.Timestamp started_at    = 11;
  google.protobuf.Timestamp ended_at      = 12;
  google.protobuf.Timestamp last_active_at = 13;
}

message GetSessionResponse { SessionProto session = 1; }

message ListSessionsRequest {
  string tenant_id = 1;
  string status    = 2;  // "active" | "completed" | ""
  string project   = 3;
  int32  limit     = 4;
  int32  offset    = 5;
}

message ListSessionsResponse { repeated SessionProto sessions = 1; }

message GetObservationsRequest {
  string session_id  = 1;
  string tenant_id   = 2;
  bool   compressed  = 3;  // true = compressed obs, false = raw obs
  int32  limit       = 4;
  int32  offset      = 5;
}

message GetObservationsResponse {
  repeated CompressedObservationProto observations = 1;
}

message StreamEventsRequest {
  string session_id = 1;  // empty = all sessions for tenant
  string tenant_id  = 2;
}

message StreamEvent {
  string event_type = 1;  // raw_observation | session_started | session_ended
  bytes  data       = 2;  // JSON
  google.protobuf.Timestamp timestamp = 3;
}
```

### File 2: `api/proto/memory/v1/agentmemory.proto`

```protobuf
syntax = "proto3";
package memory.v1;
option go_package = "github.com/vnp-memory/api/proto/memory/v1";

import "google/protobuf/timestamp.proto";

service AgentMemoryService {
  rpc RememberAgent(RememberAgentRequest) returns (RememberAgentResponse);
  rpc ListAgentMemories(ListAgentMemoriesRequest) returns (ListAgentMemoriesResponse);
  rpc GetAgentMemory(GetAgentMemoryRequest) returns (GetAgentMemoryResponse);
  rpc DeleteAgentMemory(DeleteAgentMemoryRequest) returns (DeleteAgentMemoryResponse);
  rpc GetRetentionScore(GetRetentionScoreRequest) returns (GetRetentionScoreResponse);
  rpc EvictMemories(EvictMemoriesRequest) returns (EvictMemoriesResponse);
  rpc AutoForgetSweep(AutoForgetSweepRequest) returns (AutoForgetSweepResponse);
  rpc GetSlot(GetSlotRequest) returns (GetSlotResponse);
  rpc WriteSlot(WriteSlotRequest) returns (WriteSlotResponse);
  rpc DeleteSlot(DeleteSlotRequest) returns (DeleteSlotResponse);
  rpc ListSlots(ListSlotsRequest) returns (ListSlotsResponse);
}

message RememberAgentRequest {
  string   type      = 1;  // pattern|preference|architecture|bug|workflow|fact
  string   title     = 2;
  string   content   = 3;
  repeated string concepts = 4;
  repeated string files    = 5;
  string   session_id = 6;
  double   strength   = 7;  // default 0.7
  string   tenant_id  = 8;
  string   project    = 9;
  string   agent_id   = 10;
  optional google.protobuf.Timestamp forget_after = 11;
}

message RememberAgentResponse {
  string   memory_id  = 1;
  int32    version    = 2;
  repeated string superseded = 3;
}

message AgentMemoryProto {
  string   id       = 1;
  string   type     = 2;
  string   title    = 3;
  string   content  = 4;
  repeated string concepts = 5;
  repeated string files    = 6;
  double   strength = 7;
  int32    version  = 8;
  bool     is_latest = 9;
  string   agent_id = 10;
  string   project  = 11;
  google.protobuf.Timestamp created_at = 12;
  google.protobuf.Timestamp updated_at = 13;
}

message ListAgentMemoriesRequest {
  string tenant_id = 1;
  string project   = 2;
  string type      = 3;
  bool   latest_only = 4;
  int32  limit     = 5;
  int32  offset    = 6;
}

message ListAgentMemoriesResponse { repeated AgentMemoryProto memories = 1; }
message GetAgentMemoryRequest  { string memory_id = 1; }
message GetAgentMemoryResponse { AgentMemoryProto memory = 1; }
message DeleteAgentMemoryRequest { string memory_id = 1; string tenant_id = 2; }
message DeleteAgentMemoryResponse { bool deleted = 1; }

message GetRetentionScoreRequest { string memory_id = 1; }
message GetRetentionScoreResponse {
  double score             = 1;
  double recency_factor    = 2;
  double frequency_factor  = 3;
  double importance_factor = 4;
  double days_since_access = 5;
  string recommend_action  = 6;  // keep|review|evict
}

message EvictMemoriesRequest {
  string tenant_id    = 1;
  string project      = 2;
  int32  max_memories = 3;
  bool   dry_run      = 4;
}

message EvictMemoriesResponse {
  repeated string evicted_ids = 1;
  bool dry_run = 2;
}

message AutoForgetSweepRequest { string tenant_id = 1; }
message AutoForgetSweepResponse { int32 deleted_count = 1; }

// ── Slots ─────────────────────────────────────────────────────────────────

message GetSlotRequest   { string tenant_id = 1; string scope = 2; string label = 3; }
message GetSlotResponse  { string content = 1; bool exists = 2; }

message WriteSlotRequest {
  string tenant_id = 1;
  string scope     = 2;
  string label     = 3;
  string content   = 4;
  string mode      = 5;  // replace|append
  string project   = 6;
}

message WriteSlotResponse { bool ok = 1; }

message DeleteSlotRequest    { string tenant_id = 1; string scope = 2; string label = 3; }
message DeleteSlotResponse   { bool deleted = 1; }

message SlotInfo {
  string scope      = 1;
  string label      = 2;
  string description = 3;
  bool   pinned     = 4;
  bool   read_only  = 5;
  int32  size_limit = 6;
}

message ListSlotsRequest  { string tenant_id = 1; string scope = 2; }
message ListSlotsResponse { repeated SlotInfo slots = 1; }
```

### File 3: `api/proto/search/v1/observe_search.proto`

```protobuf
syntax = "proto3";
package search.v1;
option go_package = "github.com/vnp-memory/api/proto/search/v1";

service ObserveSearchService {
  rpc SmartSearch(SmartSearchRequest) returns (SmartSearchResponse);
  rpc BM25Search(BM25SearchRequest) returns (BM25SearchResponse);
  rpc VectorSearch(VectorSearchRequest) returns (VectorSearchResponse);
  rpc BuildContext(ContextRequest) returns (ContextResponse);
  rpc IndexAdd(IndexAddRequest) returns (IndexAddResponse);
  rpc IndexRemove(IndexRemoveRequest) returns (IndexRemoveResponse);
  rpc RebuildIndex(RebuildIndexRequest) returns (RebuildIndexResponse);
  rpc GetIndexStats(GetIndexStatsRequest) returns (GetIndexStatsResponse);
}

message SearchWeights { double bm25 = 1; double vector = 2; double graph = 3; }

message SmartSearchRequest {
  string  query     = 1;
  string  tenant_id = 2;
  string  project   = 3;
  string  session_filter = 4;
  int32   limit     = 5;
  SearchWeights weights = 6;
}

message SearchResultProto {
  string id         = 1;
  string session_id = 2;
  string obs_type   = 3;
  string title      = 4;
  string narrative  = 5;
  repeated string facts    = 6;
  repeated string concepts = 7;
  double combined_score    = 8;
  double bm25_score        = 9;
  double vector_score      = 10;
}

message SmartSearchResponse {
  repeated SearchResultProto results = 1;
  int64 took_ms = 2;
}

message BM25SearchRequest  { string query = 1; string tenant_id = 2; int32 limit = 3; }
message BM25SearchResponse { repeated SearchResultProto results = 1; }
message VectorSearchRequest { string query = 1; string tenant_id = 2; int32 limit = 3; }
message VectorSearchResponse { repeated SearchResultProto results = 1; }

message ContextRequest {
  string tenant_id    = 1;
  string project      = 2;
  string session_id   = 3;
  string query        = 4;
  int32  token_budget = 5;
}

message ContextBlock {
  string type    = 1;  // memory|summary|observation
  string content = 2;
  int32  tokens  = 3;
  string source  = 4;
}

message ContextResponse {
  repeated ContextBlock blocks = 1;
  int32  total_tokens = 2;
  string formatted    = 3;
}

message IndexAddRequest {
  string obs_id     = 1;
  string session_id = 2;
  string agent_id   = 3;
  string tenant_id  = 4;
  string title      = 5;
  repeated string facts    = 6;
  repeated string concepts = 7;
}

message IndexAddResponse  { bool ok = 1; }
message IndexRemoveRequest { string doc_id = 1; }
message IndexRemoveResponse { bool ok = 1; }
message RebuildIndexRequest { string tenant_id = 1; }
message RebuildIndexResponse { int32 documents_indexed = 1; }

message GetIndexStatsRequest {}
message GetIndexStatsResponse {
  int32 bm25_documents   = 1;
  int32 vector_documents = 2;
  bool  bm25_loaded      = 3;
  bool  vector_loaded    = 4;
}
```

### File 4: `api/proto/orchestration/v1/orchestration.proto`

```protobuf
syntax = "proto3";
package orchestration.v1;
option go_package = "github.com/vnp-memory/api/proto/orchestration/v1";

import "google/protobuf/timestamp.proto";

service OrchestrationService {
  // Actions
  rpc CreateAction(CreateActionRequest)   returns (CreateActionResponse);
  rpc GetAction(GetActionRequest)         returns (GetActionResponse);
  rpc ListActions(ListActionsRequest)     returns (ListActionsResponse);
  rpc UpdateAction(UpdateActionRequest)   returns (UpdateActionResponse);
  rpc DeleteAction(DeleteActionRequest)   returns (DeleteActionResponse);

  // Leases
  rpc AcquireLease(AcquireLeaseRequest) returns (AcquireLeaseResponse);
  rpc RenewLease(RenewLeaseRequest)     returns (RenewLeaseResponse);
  rpc ReleaseLease(ReleaseLeaseRequest) returns (ReleaseLeaseResponse);
  rpc GetLease(GetLeaseRequest)         returns (GetLeaseResponse);

  // Signals
  rpc SendSignal(SendSignalRequest)       returns (SendSignalResponse);
  rpc ListSignals(ListSignalsRequest)     returns (ListSignalsResponse);
  rpc MarkSignalRead(MarkSignalReadRequest) returns (MarkSignalReadResponse);
  rpc DeleteSignal(DeleteSignalRequest)   returns (DeleteSignalResponse);

  // Routines
  rpc CreateRoutine(CreateRoutineRequest)     returns (CreateRoutineResponse);
  rpc ListRoutines(ListRoutinesRequest)       returns (ListRoutinesResponse);
  rpc ExecuteRoutine(ExecuteRoutineRequest)   returns (ExecuteRoutineResponse);

  // Checkpoints
  rpc CreateCheckpoint(CreateCheckpointRequest)   returns (CreateCheckpointResponse);
  rpc ListCheckpoints(ListCheckpointsRequest)     returns (ListCheckpointsResponse);
  rpc ApproveCheckpoint(ApproveCheckpointRequest) returns (ApproveCheckpointResponse);
  rpc RejectCheckpoint(RejectCheckpointRequest)   returns (RejectCheckpointResponse);

  // Sentinels
  rpc CreateSentinel(CreateSentinelRequest) returns (CreateSentinelResponse);
  rpc ListSentinels(ListSentinelsRequest)   returns (ListSentinelsResponse);
  rpc DeleteSentinel(DeleteSentinelRequest) returns (DeleteSentinelResponse);

  // Sketches & Crystals
  rpc CreateSketch(CreateSketchRequest)         returns (CreateSketchResponse);
  rpc ListSketches(ListSketchesRequest)         returns (ListSketchesResponse);
  rpc AddActionToSketch(AddActionToSketchRequest) returns (AddActionToSketchResponse);
  rpc PromoteSketch(PromoteSketchRequest)       returns (PromoteSketchResponse);
  rpc ListCrystals(ListCrystalsRequest)         returns (ListCrystalsResponse);
  rpc GetCrystal(GetCrystalRequest)             returns (GetCrystalResponse);
}

// ── Actions ───────────────────────────────────────────────────────────────

message CreateActionRequest {
  string   tenant_id    = 1;
  string   project      = 2;
  string   agent_id     = 3;
  string   title        = 4;
  string   description  = 5;
  int32    priority     = 6;  // 0-100, default 50
  repeated string requires     = 7;  // action IDs
  repeated string conflicts_with = 8;
  repeated string tags         = 9;
}

message ActionProto {
  string   id           = 1;
  string   tenant_id    = 2;
  string   status       = 3;  // pending|active|blocked|done|cancelled|failed
  string   title        = 4;
  string   description  = 5;
  int32    priority     = 6;
  string   result       = 7;
  repeated string tags  = 8;
  google.protobuf.Timestamp created_at   = 9;
  google.protobuf.Timestamp completed_at = 10;
}

message CreateActionResponse  { string action_id = 1; }
message GetActionRequest      { string action_id = 1; }
message GetActionResponse     { ActionProto action = 1; }
message ListActionsRequest    { string tenant_id = 1; string status = 2; int32 limit = 3; }
message ListActionsResponse   { repeated ActionProto actions = 1; }
message UpdateActionRequest   { string action_id = 1; string status = 2; string result = 3; }
message UpdateActionResponse  { bool ok = 1; }
message DeleteActionRequest   { string action_id = 1; }
message DeleteActionResponse  { bool deleted = 1; }

// ── Leases ────────────────────────────────────────────────────────────────

message AcquireLeaseRequest  { string action_id = 1; string agent_id = 2; int32 ttl_secs = 3; }
message AcquireLeaseResponse { string lease_id = 1; bool conflict = 2; string conflicting_agent = 3; }
message RenewLeaseRequest    { string lease_id = 1; int32 extend_secs = 2; }
message RenewLeaseResponse   { bool ok = 1; }
message ReleaseLeaseRequest  { string lease_id = 1; }
message ReleaseLeaseResponse { bool ok = 1; }
message GetLeaseRequest      { string action_id = 1; }
message GetLeaseResponse     { string lease_id = 1; string status = 2; string agent_id = 3; google.protobuf.Timestamp expires_at = 4; }

// ── Signals ───────────────────────────────────────────────────────────────

message SendSignalRequest {
  string tenant_id  = 1;
  string from_agent = 2;
  string to_agent   = 3;
  string signal_type = 4;  // handoff|update|cancel|request|response|alert
  string content    = 5;
  string thread_id  = 6;
  string reply_to   = 7;
  int32  expire_secs = 8;  // default 3600
}

message SendSignalResponse   { string signal_id = 1; }
message ListSignalsRequest   { string tenant_id = 1; string agent_id = 2; bool unread_only = 3; }
message ListSignalsResponse  { repeated SignalProto signals = 1; }
message SignalProto           { string id = 1; string from_agent = 2; string to_agent = 3; string signal_type = 4; string content = 5; bool is_read = 6; google.protobuf.Timestamp created_at = 7; }
message MarkSignalReadRequest  { string signal_id = 1; }
message MarkSignalReadResponse { bool ok = 1; }
message DeleteSignalRequest    { string signal_id = 1; }
message DeleteSignalResponse   { bool deleted = 1; }

// ── Checkpoints ───────────────────────────────────────────────────────────

message CreateCheckpointRequest {
  string tenant_id   = 1;
  string title       = 2;
  string description = 3;
  string action_id   = 4;
  int32  expire_secs = 5;  // default 86400
}

message CreateCheckpointResponse  { string checkpoint_id = 1; }
message ListCheckpointsRequest    { string tenant_id = 1; string status = 2; }
message ListCheckpointsResponse   { repeated CheckpointProto checkpoints = 1; }
message CheckpointProto           { string id = 1; string title = 2; string status = 3; google.protobuf.Timestamp expires_at = 4; }
message ApproveCheckpointRequest  { string checkpoint_id = 1; string approved_by = 2; }
message ApproveCheckpointResponse { bool ok = 1; }
message RejectCheckpointRequest   { string checkpoint_id = 1; string rejected_by = 2; string reason = 3; }
message RejectCheckpointResponse  { bool ok = 1; }

// ── Sketches & Crystals ───────────────────────────────────────────────────

message CreateSketchRequest      { string tenant_id = 1; string project = 2; string title = 3; int32 expire_hours = 4; }
message CreateSketchResponse     { string sketch_id = 1; }
message ListSketchesRequest      { string tenant_id = 1; string status = 2; }
message ListSketchesResponse     { repeated SketchProto sketches = 1; }
message SketchProto               { string id = 1; string title = 2; string status = 3; repeated string action_ids = 4; }
message AddActionToSketchRequest { string sketch_id = 1; string action_id = 2; }
message AddActionToSketchResponse { bool ok = 1; }
message PromoteSketchRequest     { string sketch_id = 1; }
message PromoteSketchResponse    { string crystal_id = 1; }
message ListCrystalsRequest      { string tenant_id = 1; }
message ListCrystalsResponse     { repeated CrystalProto crystals = 1; }
message GetCrystalRequest        { string crystal_id = 1; }
message GetCrystalResponse       { CrystalProto crystal = 1; }
message CrystalProto             { string id = 1; string narrative = 2; repeated string key_outcomes = 3; repeated string lessons = 4; google.protobuf.Timestamp created_at = 5; }

// ── Routines ─────────────────────────────────────────────────────────────

message CreateRoutineRequest  { string tenant_id = 1; string name = 2; string description = 3; repeated string steps = 4; }
message CreateRoutineResponse { string routine_id = 1; }
message ListRoutinesRequest   { string tenant_id = 1; }
message ListRoutinesResponse  { repeated RoutineProto routines = 1; }
message RoutineProto          { string id = 1; string name = 2; repeated string steps = 3; }
message ExecuteRoutineRequest { string routine_id = 1; string agent_id = 2; }
message ExecuteRoutineResponse { string execution_id = 1; }

// ── Sentinels ─────────────────────────────────────────────────────────────

message SentinelCondition { string type = 1; string target = 2; string value = 3; }
message CreateSentinelRequest  { string tenant_id = 1; string name = 2; SentinelCondition condition = 3; string action_id = 4; string signal_to = 5; int32 expire_hours = 6; }
message CreateSentinelResponse { string sentinel_id = 1; }
message ListSentinelsRequest   { string tenant_id = 1; string status = 2; }
message ListSentinelsResponse  { repeated SentinelProto sentinels = 1; }
message SentinelProto           { string id = 1; string name = 2; string status = 3; SentinelCondition condition = 4; }
message DeleteSentinelRequest   { string sentinel_id = 1; }
message DeleteSentinelResponse  { bool deleted = 1; }
```

### MODIFY `Makefile` — add proto-agentmemory target

```makefile
proto-agentmemory:
	@echo "Compiling AgentMemory protobufs..."
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       api/proto/observe/v1/observe.proto \
	       api/proto/memory/v1/agentmemory.proto \
	       api/proto/search/v1/observe_search.proto \
	       api/proto/orchestration/v1/orchestration.proto

.PHONY: proto-agentmemory
```

---

## Verification

```bash
make proto-agentmemory

# Check generated files
ls api/proto/observe/v1/*.pb.go
ls api/proto/memory/v1/*agentmemory*.pb.go
ls api/proto/search/v1/*observe_search*.pb.go
ls api/proto/orchestration/v1/*.pb.go

# Verify compile
go build ./...
```

## Acceptance Criteria

| Check | Expected |
|-------|----------|
| `ObserveService` có 8 RPCs | ✅ |
| `AgentMemoryService` có 11 RPCs (7 memory + 4 slots) | ✅ |
| `ObserveSearchService` có 8 RPCs | ✅ |
| `OrchestrationService` có 30 RPCs | ✅ |
| `go build ./...` passes | ✅ |
