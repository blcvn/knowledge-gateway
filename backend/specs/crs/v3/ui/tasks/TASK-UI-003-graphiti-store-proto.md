# TASK-UI-003 — Graphiti Store Proto Extension

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-003 |
| **Wave** | 1 (Backend API) |
| **Solution** | [SOL-UI-001](../solutions/SOL-UI-001-Graph-Studio.md) §2.1 |
| **Component** | `shared/proto/`, `backend/api/proto/graphiti/v1/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Mục tiêu

Verify hoặc extend `graphiti-store.proto` để bao gồm 6 Graph Studio RPCs. Regenerate Go protobuf files nếu cần.

---

## Công việc cụ thể

### 1. Verify/Modify `shared/proto/graphiti/v1/store.proto` [VERIFY/MODIFY]

Kiểm tra xem các RPCs này đã tồn tại chưa:

```protobuf
syntax = "proto3";
package graphiti.v1;
option go_package = "vnp-memory/api/proto/graphiti/v1";

import "google/protobuf/timestamp.proto";

// Graph Studio Service — read-only graph exploration
service GraphitiStoreService {
    // Existing RPCs (verify they exist):
    rpc Search(SearchRequest) returns (SearchResponse);
    rpc GetEpisodes(GetEpisodesRequest) returns (GetEpisodesResponse);

    // New Graph Studio RPCs (add if missing):
    rpc QuerySubgraph(SubgraphRequest) returns (SubgraphResponse);
    rpc GetEntity(GetEntityRequest) returns (GetEntityResponse);
    rpc GetTemporalSubgraph(TemporalSubgraphRequest) returns (TemporalSubgraphResponse);
    rpc GetOntology(GetOntologyRequest) returns (GetOntologyResponse);
    rpc UpdateOntology(UpdateOntologyRequest) returns (UpdateOntologyResponse);
    rpc ExecuteCypher(CypherRequest) returns (CypherResponse);
}

// --- Subgraph ---
message SubgraphRequest {
    string tenant_id          = 1;
    string entity_id          = 2;
    int32  depth              = 3; // 1-5
    repeated string entity_types = 4;
    repeated string relationship_types = 5;
    int32  max_nodes          = 6; // default 200
    int32  max_edges          = 7; // default 500
}

message SubgraphResponse {
    repeated GraphNode nodes  = 1;
    repeated GraphEdge edges  = 2;
    bool truncated            = 3; // true if results were capped
}

message GraphNode {
    string id         = 1;
    string type       = 2;    // entity type
    map<string, string> properties = 3;
}

message GraphEdge {
    string id             = 1;
    string source_id      = 2;
    string target_id      = 3;
    string type           = 4;  // relationship type
    google.protobuf.Timestamp valid_from = 5;
    google.protobuf.Timestamp valid_to   = 6;
    map<string, string> properties = 7;
}

// --- Entity Detail ---
message GetEntityRequest {
    string tenant_id  = 1;
    string entity_id  = 2;
}

message GetEntityResponse {
    GraphNode node        = 1;
    repeated GraphEdge edges = 2;
    repeated string source_episode_ids = 3;  // provenance
}

// --- Temporal Subgraph ---
message TemporalSubgraphRequest {
    string tenant_id      = 1;
    string entity_id      = 2;
    google.protobuf.Timestamp from = 3;
    google.protobuf.Timestamp to   = 4;
}

message TemporalSubgraphResponse {
    repeated TemporalEvent events = 1;
}

message TemporalEvent {
    google.protobuf.Timestamp timestamp = 1;
    string action  = 2;  // "added" | "expired" | "modified"
    GraphEdge edge = 3;
}

// --- Ontology ---
message GetOntologyRequest {
    string tenant_id = 1;
}

message GetOntologyResponse {
    OntologySchema schema = 1;
}

message UpdateOntologyRequest {
    string tenant_id  = 1;
    OntologySchema schema = 2;
}

message UpdateOntologyResponse {
    bool success = 1;
    string message = 2;
}

message OntologySchema {
    repeated EntityType entity_types = 1;
    repeated RelationshipType relationship_types = 2;
}

message EntityType {
    string name = 1;
    string color = 2;      // hex color for UI
    repeated string properties = 3;
}

message RelationshipType {
    string name = 1;
    repeated string valid_from_types = 2;
    repeated string valid_to_types   = 3;
    bool is_temporal = 4;
}

// --- Cypher Query ---
message CypherRequest {
    string tenant_id  = 1;
    string query      = 2;    // already validated read-only by gateway
    int32  max_nodes  = 3;
    int32  max_edges  = 4;
}

message CypherResponse {
    repeated map<string, string> rows = 1;  // tabular results
    repeated GraphNode nodes = 2;            // optional graph viz
    repeated GraphEdge edges = 3;
    bool truncated = 4;
}
```

### 2. Regenerate Go files

```bash
# Run protoc from tools/protoc3/
cd /path/to/vnp-memory
./tools/protoc3/bin/protoc \
    --go_out=backend/api/proto \
    --go-grpc_out=backend/api/proto \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    -I shared/proto \
    shared/proto/graphiti/v1/store.proto
```

### 3. Verify generated files exist

```
backend/api/proto/graphiti/v1/store.pb.go      [VERIFY/REGENERATE]
backend/api/proto/graphiti/v1/store_grpc.pb.go [VERIFY/REGENERATE]
```

---

## Acceptance Criteria

- [ ] All 6 Graph Studio RPCs defined in proto: QuerySubgraph, GetEntity, GetTemporalSubgraph, GetOntology, UpdateOntology, ExecuteCypher
- [ ] `SubgraphResponse.truncated` field present
- [ ] `GraphNode` and `GraphEdge` messages include all required fields (id, type, properties, temporal validity)
- [ ] `OntologySchema` includes EntityType + RelationshipType with color/is_temporal fields
- [ ] Go files regenerated without errors: `go build ./backend/api/proto/...`
- [ ] `go work sync` passes after regeneration

## Files

```
shared/proto/graphiti/v1/store.proto              [VERIFY/MODIFY]
backend/api/proto/graphiti/v1/store.pb.go         [REGENERATE]
backend/api/proto/graphiti/v1/store_grpc.pb.go    [REGENERATE]
```
