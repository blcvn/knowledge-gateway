---
id: DOC-S02
service: memobase-engine
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-engine — API Reference

> **Protocol**: gRPC (internal) + NATS JetStream (event-driven) | **gRPC Port**: 9032

## Primary Interface: NATS Consumer

memobase-engine is primarily event-driven. It subscribes to `memobase.buffer.ready` and processes blobs asynchronously.

```
NATS Subject: memobase.buffer.ready
Payload: {
  "user_id": "string",
  "project_id": "string",
  "buffer_ids": ["uuid", ...],
  "blob_type": "chat|doc|summary"
}
```

## gRPC Service Definition

```protobuf
// api/proto/memobase/engine/v1/service.proto
service MemobaseEngineService {
  // Trigger processing of specific buffer entries (manual/retry)
  rpc ProcessBuffer(ProcessBufferRequest) returns (ProcessBufferResponse);

  // Get pipeline status for a processing job
  rpc GetPipelineStatus(PipelineStatusRequest) returns (PipelineStatus);

  // Get project profile configuration
  rpc GetProfileConfig(ProfileConfigRequest) returns (ProfileConfig);

  // Update project profile configuration
  rpc UpdateProfileConfig(UpdateProfileConfigRequest) returns (ProfileConfig);
}
```

## Pipeline Detail (3 Fixed LLM Calls)

### LLM Call #1: Entry Chat Summary

```
Input:  ChatBlob messages (truncated to max_chat_blob_buffer_process_token_size)
Output: user_memo_str (summarized user facts + events)
Prompt: prompts/summary_entry_chats (EN/ZH)
```

### LLM Call #2: Extract Topics

```
Input:  user_memo_str + project profile schema
Output: {fact_contents: [...], fact_attributes: [{topic, sub_topic}]}
Prompt: prompts/extract_profile (EN/ZH)
```

### LLM Call #3: Merge YOLO

```
Input:  extracted facts + existing user profiles (indexed)
Output: {add: [{topic, sub_topic, memo}], update: [{index, memo}], delete: [index]}
Prompt: prompts/merge_profile_yolo (EN/ZH)
```

### Post-Processing (No LLM)

- `organize_profiles()`: If any topic has > max_profile_subtopics → merge similar
- `re_summary()`: If any profile content > max_pre_profile_token_size → summarize (conditional LLM)

## Profile Config Schema

```go
type ProfileConfig struct {
    Language            string   // "en" | "zh"
    ProfileStrictMode   bool     // Only collect defined profiles
    ProfileValidateMode bool     // Validate extracted profiles
    AdditionalProfiles  []ProfileTopic
    OverwriteProfiles   []ProfileTopic  // Replace default topics
    EventTags           []EventTagDef
    EventThemeRequirement string
}
```

## NATS Events Published

| Subject | Payload | Subscriber |
|---------|---------|------------|
| `memobase.engine.completed` | `{user_id, project_id, profiles_added, profiles_updated, profiles_deleted}` | memobase-context |
| `memobase.profile.changed` | `{user_id, project_id}` | memobase-context (cache invalidate) |
| `memobase.event.created` | `{user_id, event_id, event_data, embedding}` | vnp-event |

## Authentication

All requests require `x-tenant-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Buffer entry or user not found |
| `FAILED_PRECONDITION` | 412 | Buffer not in IDLE state |
| `INTERNAL` | 500 | LLM call failure or processing error |
| `UNAVAILABLE` | 503 | LLM gateway unavailable |
