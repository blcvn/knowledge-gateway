---
id: DOC-S02
service: memobase-context
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-context — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9033

## gRPC Service Definition

```protobuf
// api/proto/memobase/context/v1/service.proto
service MemobaseContextService {
  // Get assembled context string for LLM prompt injection
  rpc GetContext(GetContextRequest) returns (ContextResponse);

  // Get user profiles with filtering and truncation
  rpc GetProfiles(GetProfilesRequest) returns (ProfilesResponse);

  // Search profiles by semantic similarity
  rpc SearchProfiles(SearchProfilesRequest) returns (SearchProfilesResponse);
}
```

## Messages

### GetContextRequest

```protobuf
message GetContextRequest {
  string user_id = 1;
  string project_id = 2;
  int32 max_token_size = 3;         // Token budget (default: 500)
  repeated string prefer_topics = 4; // Priority topics (moved to front)
  repeated string only_topics = 5;   // Filter: keep only these topics
  float profile_event_ratio = 6;    // Profile vs event token split (default: 0.7)
  string chats_context = 7;         // Current chat for relevance filtering (optional)
}
```

### ContextResponse

```protobuf
message ContextResponse {
  string context = 1;               // Assembled prompt-ready string
  int32 profile_count = 2;
  int32 event_count = 3;
  int32 total_tokens = 4;
}
```

### GetProfilesRequest

```protobuf
message GetProfilesRequest {
  string user_id = 1;
  string project_id = 2;
  repeated string topics = 3;       // Filter by topic (empty = all)
  int32 max_subtopic_size = 4;      // Cap subtopics per topic
  int32 max_token_size = 5;         // Token budget for truncation
}
```

### ProfilesResponse

```protobuf
message ProfilesResponse {
  repeated UserProfile profiles = 1;
}

message UserProfile {
  string id = 1;
  string topic = 2;
  string sub_topic = 3;
  string content = 4;
  google.protobuf.Timestamp updated_at = 5;
}
```

## Endpoints Summary

| RPC Method | Request | Response | Description |
|-----------|---------|----------|-------------|
| GetContext | GetContextRequest | ContextResponse | Assemble prompt-ready context (profiles + events) |
| GetProfiles | GetProfilesRequest | ProfilesResponse | Retrieve user profiles with filtering |
| SearchProfiles | SearchProfilesRequest | SearchProfilesResponse | Semantic profile search |

## Context Output Format

```markdown
# Memory
Unless the user has relevant queries, do not actively mention those memories.
## User Background:
- basic_info::name: Gus
- basic_info::age: 25
- interest::food: Mexican cuisine, Thai food

## Latest Events:
- [2026-05-08] User discussed travel plans to Japan
- [2026-05-07] User mentioned working on a Go project
```

## Caching Behavior

| Aspect | Value |
|--------|-------|
| Cache Backend | Redis 7+ |
| Key Pattern | `user_profiles::{project_id}::{user_id}` |
| TTL | 1200 seconds (20 minutes) |
| Invalidation | On `memobase.profile.changed` NATS event |
| Cache Hit Target | > 90% |

## Authentication

All requests require `x-tenant-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | User not found |
| `INVALID_ARGUMENT` | 400 | Invalid max_token_size or topic filter |
| `INTERNAL` | 500 | Internal server error |
| `UNAVAILABLE` | 503 | Service unavailable |
