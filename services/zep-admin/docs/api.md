---
id: DOC-S02
service: zep-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-admin — API Reference

## gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.admin.v1;

service AdminService {
  rpc AggregatedHealth(google.protobuf.Empty) returns (AggregatedHealthResponse);
  rpc CreateProject(CreateProjectRequest) returns (ProjectResponse);
  rpc GetProject(GetProjectRequest) returns (ProjectResponse);
  rpc ListProjects(ListProjectsRequest) returns (ProjectListResponse);
  rpc UpdateProject(UpdateProjectRequest) returns (ProjectResponse);
  rpc DeleteProject(DeleteProjectRequest) returns (google.protobuf.Empty);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (APIKeyListResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc MigrateSchema(MigrateSchemaRequest) returns (MigrateSchemaResponse);
}
```

## Messages

### AggregatedHealthResponse
```protobuf
message AggregatedHealthResponse {
  string status = 1;          // "SERVING" | "NOT_SERVING" | "DEGRADED"
  map<string, ServiceHealth> services = 2;
  google.protobuf.Timestamp timestamp = 3;
}

message ServiceHealth {
  string name = 1;
  string status = 2;
  int64 latency_ms = 3;
  map<string, string> details = 4;
}
```

### ProjectResponse
```protobuf
message ProjectResponse {
  string uuid = 1;
  string name = 2;
  string description = 3;
  optional string organization_id = 4;
  ProjectSettings settings = 5;
  google.protobuf.Timestamp created_at = 6;
}

message ProjectSettings {
  int64 max_request_size = 1;    // bytes, default 5MB
  int32 request_timeout_ms = 2;  // default 30s
  int32 rate_limit_rps = 3;      // default 100
  bool telemetry_enabled = 4;
  bool graphiti_enabled = 5;
}
```

### CreateAPIKeyResponse
```protobuf
message CreateAPIKeyResponse {
  string key = 1;              // raw key (shown only once)
  string key_prefix = 2;      // first 8 chars for identification
  string project_uuid = 3;
  repeated string scopes = 4; // "read", "write", "admin"
}
```

## RPC Details

### AggregatedHealth

| Attribute | Value |
|-----------|-------|
| **Behavior** | Parallel health check to all 5 domain services |
| **Timeout** | 5s per service |
| **Status logic** | All SERVING → SERVING; any NOT_SERVING → DEGRADED |

### CreateAPIKey

| Attribute | Value |
|-----------|-------|
| **Key format** | 32 bytes random, SHA-256 hashed for storage |
| **Scopes** | "read", "write", "admin" (default: read, write) |
| **Security** | Raw key shown only once at creation |
