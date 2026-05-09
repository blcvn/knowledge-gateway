---
id: DOC-S02
service: zep-thread
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-thread — API Reference

## gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.thread.v1;

service ThreadService {
  rpc CreateSession(CreateSessionRequest) returns (SessionResponse);
  rpc GetSession(GetSessionRequest) returns (SessionResponse);
  rpc UpdateSession(UpdateSessionRequest) returns (SessionResponse);
  rpc UpsertSession(UpsertSessionRequest) returns (SessionResponse);
  rpc EndSession(EndSessionRequest) returns (google.protobuf.Empty);
  rpc ListSessions(ListSessionsRequest) returns (SessionListResponse);
  rpc ListOrderedSessions(ListSessionsRequest) returns (SessionListResponse);
  rpc ListUserSessions(ListUserSessionsRequest) returns (SessionListResponse);
}
```

## Messages

### SessionResponse

```protobuf
message SessionResponse {
  string uuid = 1;
  string session_id = 2;
  optional string user_id = 3;
  string project_uuid = 4;
  google.protobuf.Struct metadata = 5;
  optional google.protobuf.Timestamp ended_at = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
}
```

### CreateSessionRequest

```protobuf
message CreateSessionRequest {
  string session_id = 1;           // required
  optional string user_id = 2;    // optional FK → User
  google.protobuf.Struct metadata = 3;
}
```

### UpsertSessionRequest

```protobuf
message UpsertSessionRequest {
  string session_id = 1;           // required
  optional string user_id = 2;
  google.protobuf.Struct metadata = 3;
}
```

### UpdateSessionRequest

```protobuf
message UpdateSessionRequest {
  string session_id = 1;
  google.protobuf.Struct metadata = 2;  // merge-patch
}
```

## RPC Details

### CreateSession

| Attribute | Value |
|-----------|-------|
| **Auth** | Required (JWT/API Key via gateway) |
| **Validation** | `session_id` required, alphanumeric with underscores |

**Errors**: `ALREADY_EXISTS`, `INVALID_ARGUMENT`

### UpsertSession

| Attribute | Value |
|-----------|-------|
| **Behavior** | Create if not exists, update if exists |
| **Used by** | zep-memory PutMemory flow |

### UpdateSession

| Attribute | Value |
|-----------|-------|
| **Concurrency** | Advisory lock on session_id (SHA-256 hash) |
| **Metadata** | JSONB merge-patch semantics |
| **Retry Policy** | 200ms→30s exponential backoff, max 15 retries |

**Errors**: `NOT_FOUND`, `DEADLINE_EXCEEDED` (lock timeout)

### EndSession

| Attribute | Value |
|-----------|-------|
| **Behavior** | Sets `ended_at` timestamp — blocks future message ingestion |
| **Event** | Publishes `zep.thread.session.ended` |

**Errors**: `NOT_FOUND`

## NATS Events Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `zep.thread.session.created` | `{session_id, user_id, project_uuid}` | After CreateSession |
| `zep.thread.session.ended` | `{session_id, project_uuid, ended_at}` | After EndSession |
| `zep.thread.session.deleted` | `{session_id, project_uuid}` | After soft delete |
