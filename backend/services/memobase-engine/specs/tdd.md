---
id: TDD-memobase-engine
title: Technical Design — memobase-engine
service: memobase-engine
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-09
group: Memobase
---

# Technical Design — memobase-engine

> **Group**: Memobase | **gRPC Port**: 9032 | **Health Port**: 9099

## 1. Service Overview

Core LLM processing engine. Subscribes to `memobase.buffer.ready`, executes a fixed 3-LLM-call pipeline (entry_summary → extract_topics → merge_yolo), produces updated user profiles and temporal events with embeddings.

## 2. Clean Architecture Layers

### Domain Layer (Layer 1)
- **Profile**: id, user_id, project_id, topic, sub_topic, content, attributes, updated_at
- **UserEvent**: id, user_id, project_id, event_data (JSONB), embedding (vector), created_at
- **EventGist**: id, event_id, gist_data (JSONB), embedding (vector)
- **MergeDecision**: add[], update[], delete[] — output of YOLO merge
- **ProfileConfig**: language, profile_strict_mode, additional_profiles, event_tags
- **PipelineResult**: profiles_delta, events_created, tokens_consumed

### Usecase Layer (Layer 2)
- **ProcessBufferUseCase**: Orchestrate full pipeline (fetch blobs → 3 LLM calls → persist)
- **ExtractProfileUseCase**: LLM call #2 — extract structured profile facts
- **MergeProfileUseCase**: LLM call #3 — YOLO merge with existing profiles
- **ProcessEventUseCase**: Event tagging + embedding generation
- **OrganizeProfileUseCase**: Merge similar subtopics if exceeding limit (no LLM)

### Adapter Layer (Layer 3)
- **gRPC handler**: ProcessBuffer, GetPipelineStatus, GetProfileConfig
- **NATS consumer**: Subscribe to `memobase.buffer.ready`
- **NATS publisher**: Publish engine.completed, profile.changed, event.created
- **Bifrost LLM client**: Multi-provider LLM abstraction
- **Embedder client**: Vector embedding generation (OpenAI/Jina/Ollama)
- **PostgreSQL repos**: ProfileRepository, EventRepository

### Infrastructure Layer (Layer 4)
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel), Prompt templates (EN/ZH)

## 3. Processing Pipeline (3 Fixed LLM Calls)

```
memobase.buffer.ready → fetch blobs from PostgreSQL
  │
  ├── LLM #1: entry_chat_summary
  │   Input: ChatBlob messages (truncated to 16384 tokens)
  │   Output: user_memo_str
  │
  ├── Parallel:
  │   ├── LLM #2: extract_topics
  │   │   Input: user_memo_str + profile schema
  │   │   Output: {fact_contents[], fact_attributes[{topic, sub_topic}]}
  │   │
  │   └── Event Processing:
  │       ├── tag_event (conditional LLM if event_tags configured)
  │       └── append_user_event (embed + store)
  │
  ├── LLM #3: merge_yolo
  │   Input: extracted facts + existing profiles (indexed)
  │   Output: {add[], update[{index, memo}], delete[index]}
  │
  ├── Post-processing (no LLM):
  │   ├── organize_profiles (if subtopics > max)
  │   └── re_summary (conditional, if content > max tokens)
  │
  └── Persist: upsert profiles + store events + invalidate cache
```

## 4. YOLO Merge Algorithm

**Input**:
```
Existing Profiles:
[0] basic_info::name: "Gus"
[1] interest::food: "Mexican cuisine"

New Facts:
- basic_info::age: "25"
- interest::food: "Also likes Thai food"
```

**Output** (JSON):
```json
{
  "add": [{"topic": "basic_info", "sub_topic": "age", "memo": "25"}],
  "update": [{"index": 1, "memo": "Mexican cuisine, Thai food"}],
  "delete": []
}
```

## 5. NATS Events

| Direction | Subject | Payload |
|-----------|---------|---------|
| Subscribe | `memobase.buffer.ready` | `{user_id, project_id, buffer_ids[]}` |
| Publish | `memobase.engine.completed` | `{user_id, project_id, profiles_delta}` |
| Publish | `memobase.profile.changed` | `{user_id, project_id}` |
| Publish | `memobase.event.created` | `{user_id, event_id, embedding}` |

## 6. Data Model

### Tables (owned)
- `user_profiles`: id, user_id, project_id, content, attributes (JSONB: {topic, sub_topic}), updated_at
- `user_events`: id, user_id, project_id, event_data (JSONB), embedding (VECTOR), created_at
- `user_event_gists`: id, user_id, project_id, event_id (FK), gist_data (JSONB), embedding (VECTOR)

### Key Indexes
- `idx_profiles_user`: (user_id, project_id) — profile retrieval
- `idx_events_embedding`: HNSW on embedding — semantic search
- `idx_gists_embedding`: HNSW on gist embedding — fine-grained search

## 7. Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL + pgvector | SQL | Profile, event, gist persistence |
| NATS JetStream | Consumer/Publisher | Pipeline orchestration |
| Bifrost (LLM) | HTTP/gRPC | LLM calls (3 per flush) |
| Embedder | HTTP | Event/gist embedding generation |

## 8. Observability

- **Metrics**: llm_invocations_total, llm_latency_ms, llm_tokens_input/output, profile_updates_total, events_created_total, pipeline_latency_ms
- **Traces**: OTel spans per pipeline step (entry_summary, extract, merge, persist)
- **Logs**: Structured JSON with request_id, tenant_id, user_id, pipeline_id
- **Health**: gRPC health check + HTTP /healthz on port 9099

## 9. Multi-Tenancy

Composite PK `(id, project_id)`. Per-project config (language, profile schema, event tags) stored in `projects.profile_config`.

---

> **Next Steps**: Decompose into FEAT-001 (Processing Pipeline), FEAT-002 (YOLO Merge), FEAT-003 (Event Processing), ARCH-001 (Prompt Template System) in `specs/features/` and `specs/architecture/`.
