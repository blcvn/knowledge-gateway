---
id: DOC-S02
service: ov-session
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-session — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9053

## gRPC Service Definition

```protobuf
// api/proto/openviking/session/v1/service.proto
service OvSessionService {
  rpc CreateSession(CreateSessionRequest) returns (Session);
  rpc AddMessage(AddMessageRequest) returns (google.protobuf.Empty);
  rpc GetMessages(GetMessagesRequest) returns (MessagesResponse);
  rpc CommitSession(CommitSessionRequest) returns (CommitResponse);
  rpc GetWorkingMemory(GetWMRequest) returns (WorkingMemory);
  rpc UpdateWorkingMemory(UpdateWMRequest) returns (WorkingMemory);
}
```

## Endpoints

### CreateSession

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_id` | string | Yes | Tenant account |
| `user_id` | string | Yes | User creating session |
| `agent_id` | string | No | Agent ID (default: "default") |
| `title` | string | No | Session title |
| `metadata` | map | No | Custom session metadata |

**Response**: `Session { id, account_id, user_id, agent_id, title, status, created_at }`

### AddMessage

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session_id` | string | Yes | Active session ID |
| `role` | MessageRole | Yes | USER / ASSISTANT / SYSTEM / TOOL |
| `content` | string | Yes | Message content |
| `tool_calls` | []ToolCall | No | Tool invocations |

### CommitSession

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session_id` | string | Yes | Session to commit |
| `extract_memories` | bool | No | Run LLM memory extraction (default: true) |
| `compression_version` | string | No | "v1" (legacy) or "v2" (template system) |

**Response**: `CommitResponse { archive_path, memories_count, extraction_stats }`

### GetWorkingMemory / UpdateWorkingMemory

Working Memory v2 structured document:

```json
{
  "title": "Debugging auth middleware",
  "state": "ongoing",
  "goals": ["Fix JWT validation", "Add test coverage"],
  "facts": [{"key": "auth_mode", "value": "api_key", "confidence": 0.95}],
  "errors": [{"message": "token expired", "resolved": false}],
  "context": {"file": "server/auth.py", "line": 105}
}
```

## Memory Categories (from MemoryExtractor)

| Category | Description | Example |
|----------|-------------|---------|
| `fact` | Declarative knowledge | "User prefers dark mode" |
| `preference` | User preferences | "Always use TypeScript" |
| `skill` | Procedural knowledge | "Deploy via kubectl apply" |
| `procedure` | Multi-step workflow | "1. Build → 2. Test → 3. Deploy" |
| `tool_skill` | Tool usage patterns | "Use grep -r for recursive search" |

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Session not found |
| `FAILED_PRECONDITION` | 412 | Session already committed |
| `INVALID_ARGUMENT` | 400 | Invalid message role or session ID |
| `INTERNAL` | 500 | LLM extraction failure |
