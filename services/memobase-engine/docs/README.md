---
id: DOC-S01
service: memobase-engine
version: 1.1.0
status: Deprecated
created: 2026-05-09
updated: 2026-05-11
owner: VNP Memory — Memobase Team
---

# memobase-engine (DEPRECATED)

> **⚠️ DEPRECATION NOTICE**: This service has been merged into `memobase-pipeline` per `ARCH-004`. The code and documentation here are kept for historical reference only. Please refer to `memobase-pipeline` for the active implementation.

> **Group**: Memobase | **gRPC Port**: 9032 | **Health Port**: 9099 | **Origin**: Memobase

## Purpose

Core LLM processing engine for the Memobase memory system. Consumes buffered blobs from `memobase-ingestion`, executes a **fixed 3-LLM-call pipeline** (entry_summary → extract_topics → merge_yolo), and produces updated user profiles and temporal events.

### Business Capability

- **Profile Extraction**: Extract structured user profile facts from conversation summaries
- **YOLO Merge**: Single LLM call determines all add/update/delete decisions for profile slots
- **Event Processing**: Generate temporal events with optional tag classification
- **Embedding Generation**: Create vector embeddings for events and event gists (pgvector)
- **Multi-Language**: EN/ZH prompt templates selected per project configuration
- **Token Billing**: Async fire-and-forget token accounting per project

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream consumer/publisher
- **Database**: PostgreSQL (profiles, events, event_gists)
- **LLM Gateway**: Bifrost (multi-provider: OpenAI, Doubao, Ollama)
- **Embedder**: `pkg/adapters/embedder/` (OpenAI, Jina, Ollama, LMStudio)
- **Tokenizer**: tiktoken-go (gpt-4o encoder) via `pkg/tokenizer/`
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire

## Quick Start

```bash
make build-memobase-engine
make run-memobase-engine
docker compose up memobase-engine postgresql nats bifrost
```

## Pipeline

```
NATS: memobase.buffer.ready {user_id, project_id, buffer_ids[]}
  │
  ▼
LLM #1: entry_chat_summary → user_memo_str
  │
  ├── LLM #2: extract_topics → {fact_contents[], fact_attributes[]}
  │
  └── LLM #3: merge_yolo → {add[], update[], delete[]}
  │
  ▼
Output: updated_profiles[] + new_events[] + embeddings[]
```

**Total LLM calls**: Fixed 3 per flush (entry_summary + extract + merge_yolo)

### Profile Schema

```go
type Profile struct {
    ID, UserID, ProjectID string
    Topic, SubTopic, Content string
    Attributes map[string]any
    UpdatedAt time.Time
}
```

## NATS Events

| Direction | Subject | Payload | Peer |
|-----------|---------|---------|------|
| Subscribe | `memobase.buffer.ready` | `{user_id, project_id, buffer_ids[]}` | memobase-ingestion |
| Publish | `memobase.engine.completed` | `{user_id, project_id, profiles_delta}` | memobase-context |
| Publish | `memobase.profile.changed` | `{user_id, project_id}` | memobase-context (cache invalidate) |
| Publish | `memobase.event.created` | `{user_id, event_id, embedding}` | vnp-event |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Profile, event, event_gist persistence |
| NATS JetStream | Consumer/Publisher | Buffer ready → engine completed/profile changed |
| Bifrost (LLM) | HTTP/gRPC | LLM calls (extract, merge, summary) |
| Embedder | HTTP | Embedding generation for events/gists |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/04-memobase-services.md)
- [Memobase Reference](../../../references/memobase/)

## Owner

- **Team**: VNP Memory — Memobase
