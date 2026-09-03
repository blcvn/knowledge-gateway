---
id: DOC-S01
service: memobase-context
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Memobase Team
---

# memobase-context

> **Group**: Memobase | **gRPC Port**: 9033 | **Health Port**: 9100 | **Origin**: Memobase

## Purpose

Read-path context assembly service for the Memobase memory system. Assembles **prompt-ready context strings** from user profiles and event gists with a target of **< 100ms p95 latency**. Serves as the Memobase search endpoint for `vnp-search-hub` cross-engine recall.

### Business Capability

- **Context Assembly**: Build LLM-ready context strings from profiles + event gists with token budget allocation
- **Profile Retrieval**: Fetch user profiles with topic filtering, priority ordering, and token truncation
- **Profile Search**: Semantic search over user profiles (optional chat-aware filtering via LLM)
- **Event Gist Search**: pgvector cosine similarity search over fine-grained event descriptions
- **Redis Caching**: Profile cache with 20-minute TTL, invalidated on `memobase.profile.changed`
- **Prompt Templates**: EN/ZH/Custom template support with `{profile_section}` and `{event_section}` placeholders

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream consumer
- **Database**: PostgreSQL + pgvector (profiles, event gist vector search)
- **Cache**: Redis 7+ (profile cache, TTL 20min)
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire

## Quick Start

```bash
make build-memobase-context
make run-memobase-context
docker compose up memobase-context postgresql redis
```

## API Surface

### gRPC Service

```protobuf
service MemobaseContextService {
  rpc GetContext(GetContextRequest) returns (ContextResponse);
  rpc GetProfiles(GetProfilesRequest) returns (ProfilesResponse);
  rpc SearchProfiles(SearchProfilesRequest) returns (SearchProfilesResponse);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/memobase/users/{uid}/context` | Get assembled context string |
| GET | `/v1/memobase/users/{uid}/profiles` | Get user profiles |

### Context Assembly Algorithm

```
GetContext(user_id, project_id, max_token_size)
  │
  ├── Token budget: profile_tokens = max_token_size × profile_event_ratio
  │
  ├── Parallel fetch (asyncio.gather equivalent):
  │   ├── GetProfiles → Redis cache → DB fallback → truncate by token budget
  │   └── SearchEventGists → pgvector cosine similarity (threshold > 0.2)
  │
  ├── Profile section: "- {topic}::{sub_topic}: {content}" per profile
  │
  ├── Event section: fill remaining token budget with event gists
  │
  └── Assemble via prompt template → context string
```

### SLA Targets

| Metric | Target |
|--------|--------|
| Context retrieval (p95) | < 100ms |
| Profile cache hit rate | > 90% |
| Cache TTL | 20 minutes |

## NATS Events Subscribed

| Subject | Source | Action |
|---------|--------|--------|
| `memobase.engine.completed` | memobase-engine | Refresh context state |
| `memobase.profile.changed` | memobase-engine | Invalidate Redis profile cache |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL + pgvector | SQL | Profile retrieval + event gist vector search |
| Redis | Cache | Profile cache (TTL 20min) |
| NATS JetStream | Consumer | Subscribe to profile.changed + engine.completed |
| vnp-search-hub | gRPC (caller) | Provides Memobase results for cross-engine recall |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/04-memobase-services.md)
- [Memobase Reference](../../../references/memobase/)

## Owner

- **Team**: VNP Memory — Memobase
