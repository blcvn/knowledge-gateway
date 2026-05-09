---
id: DOC-S01
service: ov-session
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — OpenViking Team
---

# ov-session

> **Group**: OpenViking | **gRPC Port**: 9053 | **Health Port**: 9106 | **Origin**: OpenViking

## Purpose

Session lifecycle management with **2-phase commit** (archive + extract), **Working Memory v2** (structured document), **memory extraction** (LLM-based fact/skill/preference extraction), **memory deduplication**, and **session compression**. Replaces Python `openviking/session/session.py` (107KB).

### Business Capability

- **Session Lifecycle**: Create → AddMessage → Commit → Archive
- **2-Phase Commit**: Phase 1 (Archive conversation → write to ov-fs), Phase 2 (LLM extract memories → write to ov-fs)
- **Working Memory v2**: Structured document `{title, state, goals, facts, errors, context}` — evolves during session
- **Memory Extraction**: LLM extracts categorized memories (facts, preferences, skills, procedures)
- **Memory Deduplication**: Detect and merge duplicate memories using semantic similarity
- **Session Compression**: Summarize long conversations to fit context windows

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Database**: PostgreSQL (session metadata, messages)
- **LLM**: Bifrost (memory extraction, compression)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-ov-session
make run-ov-session
docker compose up ov-session postgresql nats bifrost
```

## API Surface

### gRPC Service

```protobuf
service OvSessionService {
  rpc CreateSession(CreateSessionRequest) returns (Session);
  rpc AddMessage(AddMessageRequest) returns (google.protobuf.Empty);
  rpc GetMessages(GetMessagesRequest) returns (MessagesResponse);
  rpc CommitSession(CommitSessionRequest) returns (CommitResponse);
  rpc GetWorkingMemory(GetWMRequest) returns (WorkingMemory);
  rpc UpdateWorkingMemory(UpdateWMRequest) returns (WorkingMemory);
}
```

### 2-Phase Commit Pipeline

```
CommitSession:
  Phase 1 (Archive):
    1. Compress conversation (SessionCompressor)
    2. Write compressed archive → ov-fs
    3. Persist archive metadata

  Phase 2 (Extract):
    1. LLM extract memories (MemoryExtractor)
       → Categories: facts, preferences, skills, procedures, tool_skills
    2. Deduplicate against existing memories (MemoryDeduplicator)
       → Actions: CREATE, MERGE, SKIP, ARCHIVE
    3. Write extracted memories → ov-fs
    4. Emit: ov.session.committed, ov.session.memory.extracted
```

### Working Memory v2 Schema

```go
type WorkingMemory struct {
    Title    string            // Current session/task title
    State    string            // ongoing | paused | completed
    Goals    []string          // Active goals
    Facts    []FactEntry       // Key facts from conversation
    Errors   []ErrorEntry      // Errors encountered
    Context  map[string]string // Contextual metadata
}
```

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| ov-fs | gRPC | Write archives and extracted memories |
| ov-search | NATS | Hotness boost for referenced files |
| Bifrost (LLM) | gRPC | Memory extraction, compression |
| PostgreSQL | SQL | Session metadata, messages, WM state |

## NATS Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `ov.session.committed` | Publish | Session archived + memories extracted |
| `ov.session.memory.extracted` | Publish | Extracted memories ready for ov-fs write |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — OpenViking
