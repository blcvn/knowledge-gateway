---
id: DOC-S02
service: zep-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-memory — API Reference

## gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.memory.v1;

service MemoryService {
  rpc PutMemory(PutMemoryRequest) returns (google.protobuf.Empty);
  rpc GetMemory(GetMemoryRequest) returns (MemoryResponse);
  rpc DeleteMemory(DeleteMemoryRequest) returns (google.protobuf.Empty);
  rpc GetMessagesForSession(GetMessagesRequest) returns (MessageListResponse);
  rpc GetMessage(GetMessageRequest) returns (MessageResponse);
  rpc UpdateMessageMetadata(UpdateMessageMetadataRequest) returns (MessageResponse);
  rpc GetUserContext(GetUserContextRequest) returns (UserContextResponse);
}
```

## Messages

### PutMemoryRequest

```protobuf
message PutMemoryRequest {
  string session_id = 1;
  repeated MessageInput messages = 2;
}

message MessageInput {
  string role = 1;          // "user" | "assistant" | "system"
  string role_type = 2;     // norole|system|assistant|user|function|tool
  string content = 3;
  google.protobuf.Struct metadata = 4;
}
```

### MemoryResponse

```protobuf
message MemoryResponse {
  repeated MessageResponse messages = 1;
  repeated FactResponse relevant_facts = 2;
  google.protobuf.Struct metadata = 3;
}
```

### FactResponse

```protobuf
message FactResponse {
  string uuid = 1;
  string name = 2;
  string fact = 3;
  google.protobuf.Timestamp created_at = 4;
  optional google.protobuf.Timestamp valid_at = 5;
  optional google.protobuf.Timestamp invalid_at = 6;
  optional google.protobuf.Timestamp expired_at = 7;
}
```

### MessageResponse

```protobuf
message MessageResponse {
  string uuid = 1;
  string session_id = 2;
  string role = 3;
  string role_type = 4;
  string content = 5;
  int32 token_count = 6;
  google.protobuf.Struct metadata = 7;
  google.protobuf.Timestamp created_at = 8;
}
```

### GetUserContextRequest / Response

```protobuf
message GetUserContextRequest {
  string thread_id = 1;
  optional string template_id = 2;  // custom context template
}

message UserContextResponse {
  string context = 1;              // pre-formatted context string for LLM
  repeated FactResponse facts = 2;
}
```

## RPC Details

### PutMemory (Critical Path)

| Attribute | Value |
|-----------|-------|
| **Latency target** | < 200ms (excludes async graph extraction) |
| **Session state** | Auto-upsert via zep-thread if not exists |
| **Validation** | Rejects if session ended, empty messages |
| **Side effects** | Publishes `zep.memory.messages.ingested` (async 10-20s processing) |

### GetMemory (Context Assembly)

| Attribute | Value |
|-----------|-------|
| **Context assembly** | Messages + relevant facts from knowledge graph |
| **Fact retrieval** | Uses last 4 messages as search context against zep-search |
| **Degradation** | Returns messages-only if zep-search unavailable |
| **GroupID** | `user_id` if user linked, else `session_id` |
