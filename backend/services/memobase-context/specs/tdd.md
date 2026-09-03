---
id: TDD-memobase-context
title: Technical Design — memobase-context
service: memobase-context
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-09
group: Memobase
---

# Technical Design — memobase-context

> **Group**: Memobase | **gRPC Port**: 9033 | **Health Port**: 9100

## 1. Service Overview

Read-path context assembly service. Assembles prompt-ready context strings from user profiles and event gists. Target: < 100ms p95 latency. Serves as Memobase search endpoint for vnp-search-hub cross-engine recall.

## 2. Clean Architecture Layers

### Domain Layer (Layer 1)
- **Profile**: read-only view (topic, sub_topic, content, updated_at)
- **EventGist**: read-only view (gist_data, embedding, created_at)
- **ContextResult**: assembled context string with token count
- **PromptTemplate**: EN/ZH/Custom template with `{profile_section}` and `{event_section}` placeholders
- **TruncationPolicy**: prefer_topics, only_topics, max_token_size, max_subtopic_size

### Usecase Layer (Layer 2)
- **GetContextUseCase**: Token budget allocation → parallel profile+event fetch → assemble template
- **GetProfilesUseCase**: Redis cache first → DB fallback → truncate by token budget
- **SearchProfilesUseCase**: Semantic search over profiles (optional chat-aware LLM filtering)

### Adapter Layer (Layer 3)
- **gRPC handler**: GetContext, GetProfiles, SearchProfiles
- **PostgreSQL repos**: ProfileReadRepository, EventGistSearchRepository (pgvector)
- **Redis cache**: ProfileCacheAdapter (GET/SET/DEL with TTL 1200s)
- **NATS consumer**: Subscribe to `memobase.profile.changed` and `memobase.engine.completed`

### Infrastructure Layer (Layer 4)
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel)

## 3. gRPC API

```protobuf
service MemobaseContextService {
  rpc GetContext(GetContextRequest) returns (ContextResponse);
  rpc GetProfiles(GetProfilesRequest) returns (ProfilesResponse);
  rpc SearchProfiles(SearchProfilesRequest) returns (SearchProfilesResponse);
}
```

## 4. Context Assembly Algorithm

```
GetContext(user_id, project_id, max_token_size=500, profile_event_ratio=0.7)
  │
  ├── profile_budget = max_token_size × 0.7 = 350 tokens
  │
  ├── Parallel (errgroup):
  │   ├── profiles = getProfiles(user_id, project_id)
  │   │   ├── Redis.Get("user_profiles::{project_id}::{user_id}")
  │   │   ├── miss → SQL: SELECT * FROM user_profiles WHERE user_id=? AND project_id=?
  │   │   ├── Redis.Set(key, profiles_json, TTL=1200s)
  │   │   └── truncate_profiles(profiles, prefer_topics, only_topics, profile_budget)
  │   │       ├── Sort by updated_at DESC
  │   │       ├── Move prefer_topics to front
  │   │       ├── Filter only_topics if specified
  │   │       └── Accumulate tokens until budget exceeded
  │   │
  │   └── events = searchEventGists(user_id, project_id, chats_context)
  │       └── pgvector: cosine_distance < (1 - 0.2), created_at > now()-21d, LIMIT 10
  │
  ├── profile_section = "- {topic}::{sub_topic}: {content}\n" per profile
  ├── event_budget = max_token_size - profile_actual_tokens
  ├── event_section = truncated gists within event_budget
  │
  └── return prompt_template(profile_section, event_section)
```

## 5. Profile Truncation Algorithm

```
truncate_profiles(profiles, prefer_topics, only_topics, max_token_size, max_subtopic_size, topic_limits)
  1. Sort by updated_at DESC (most recent first)
  2. Priority ordering: move prefer_topics to front, maintain internal order
  3. Topic filter: keep only_topics if specified
  4. Subtopic limit: cap subtopics per topic (max_subtopic_size or per-topic limits)
  5. Token budget: accumulate tokens per profile, stop when budget exceeded
```

## 6. NATS Events

| Direction | Subject | Source | Action |
|-----------|---------|--------|--------|
| Subscribe | `memobase.engine.completed` | memobase-engine | Update context state |
| Subscribe | `memobase.profile.changed` | memobase-engine | Redis.DEL(profile cache key) |

## 7. Caching Strategy

```
Key Pattern: user_profiles::{project_id}::{user_id}
Value: JSON-serialized []Profile
TTL: 1200 seconds (20 minutes)

Write Path: Any profile mutation → NATS event → Redis.DEL(key)
Read Path: GetProfiles → Redis.GET(key) → miss → DB query → Redis.SET(key, TTL)
```

**Target**: > 90% cache hit rate

## 8. Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL + pgvector | SQL (read-only) | Profile retrieval + event gist vector search |
| Redis | Cache | Profile cache (TTL 20min, event-driven invalidation) |
| NATS JetStream | Consumer | Cache invalidation events |
| vnp-search-hub | gRPC (caller) | Cross-engine recall fan-out target |

## 9. Observability

- **Metrics**: context_latency_ms (p95 target < 100ms), cache_hit_ratio, profiles_served_total, event_gists_searched_total
- **Traces**: OTel spans for GetContext, GetProfiles, SearchProfiles, cache operations
- **Logs**: Structured JSON via slog with request_id, tenant_id, user_id
- **Health**: gRPC health check + HTTP /healthz on port 9100

## 10. Multi-Tenancy

Tenant isolation via `project_id` composite PK on all queries. Redis key includes project_id for cache isolation.

---

> **Next Steps**: Decompose into FEAT-001 (Context Assembly API), FEAT-002 (Redis Caching), FEAT-003 (Profile Truncation), FEAT-004 (Event Gist Search) in `specs/features/`.
