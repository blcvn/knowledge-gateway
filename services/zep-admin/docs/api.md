# Zep Admin API

## Overview
The Zep Admin Service manages the health aggregation, project/tenant isolation, and API key lifecycle across the Zep ecosystem.

## gRPC Services

### ZepAdminService (Port 9066)

#### Project Management
```protobuf
message Project {
  string uuid = 1;
  string name = 2;
  string description = 3;
  string api_key_hash = 4;
  int64 created_at = 5;
  int64 updated_at = 6;
}

rpc CreateProject(CreateProjectRequest) returns (Project);
rpc GetProject(GetProjectRequest) returns (Project);
rpc UpdateProject(UpdateProjectRequest) returns (Project);
rpc DeleteProject(DeleteProjectRequest) returns (Empty);
rpc ListProjects(ListProjectsRequest) returns (ListProjectsResponse);
```

#### API Key Management
```protobuf
rpc GenerateApiKey(GenerateApiKeyRequest) returns (GenerateApiKeyResponse);
rpc RevokeApiKey(RevokeApiKeyRequest) returns (Empty);
```

#### Health & Status
```protobuf
rpc GetSystemHealth(Empty) returns (SystemHealthResponse);
rpc GetProjectStats(GetProjectStatsRequest) returns (ProjectStatsResponse);
```

## Events Produced
- `zep.admin.project.created`: Emitted when a project is created (triggers schema init in other services).
- `zep.admin.project.deleted`: Emitted when a project is deleted (triggers cascade deletes).
