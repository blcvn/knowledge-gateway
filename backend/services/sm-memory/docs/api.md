---
id: DOC-S02
service: sm-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-memory — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9072

## gRPC Service Definition

```protobuf
service SmMemoryService {
  rpc CreateMemory(CreateMemoryRequest) returns (Memory);
  rpc GetMemory(GetMemoryRequest) returns (Memory);
  rpc UpdateMemory(UpdateMemoryRequest) returns (Memory);
  rpc ForgetMemory(ForgetMemoryRequest) returns (google.protobuf.Empty);
  rpc ListMemories(ListMemoriesRequest) returns (ListMemoriesResponse);
  rpc CreateRelation(CreateRelationRequest) returns (Relation);
  rpc GetMemoryWithContext(GetContextRequest) returns (MemoryWithContext);
}
```

## Endpoints

### CreateMemory

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| memory | string | Yes | Memory content text |
| space_id | string | Yes | Target space |
| metadata | map | No | Key-value metadata |
| is_static | bool | No | Exempt from forgetting curve |
| parent_memory_id | string | No | Previous version link |
| relation_type | string | No | updates/extends/derives |

### GetMemoryWithContext

Returns memory entry with parent chain (up to 3 ancestors) and child chain (up to 3 descendants), including relation types and version distances.

### ForgetMemory

Marks memory as `is_forgotten=true` with optional `forget_reason`. Does not physically delete.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Memory not found |
| `INVALID_ARGUMENT` | 400 | Invalid request parameters |
| `ALREADY_EXISTS` | 409 | Duplicate memory in chain |
| `INTERNAL` | 500 | Internal server error |
