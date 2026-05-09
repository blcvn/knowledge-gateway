---
id: TDD-ov-session
title: Technical Design — ov-session
service: ov-session
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: OpenViking
---

# Technical Design — ov-session

> **Group**: OpenViking | **gRPC Port**: 9053 | **Origin**: OpenViking (Session + Compressor + MemoryExtractor)

## 1. Service Overview

Session lifecycle management with 2-phase commit (archive + extract), Working Memory v2, LLM-based memory extraction (5 categories), and memory deduplication with semantic similarity.

**Origin mapping**: `openviking/session/session.py` (107KB) + `openviking/session/compressor_v2.py` + `openviking/session/memory_extractor.py` + `openviking/session/memory_deduplicator.py`.

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── session.go               # Session, SessionMeta, SessionStatus
│   ├── message.go               # Message, MessageRole, ToolCall
│   ├── working_memory.go        # WorkingMemory v2 (title, state, goals, facts, errors)
│   ├── memory.go               # CandidateMemory, MemoryCategory enum
│   └── compression.go          # ExtractionStats, CompressionVersion
├── repository/
│   ├── session_repo.go          # SessionRepository
│   └── message_repo.go          # MessageRepository
├── event.go                     # SessionCommitted, MemoryExtracted
└── errors.go                    # SessionNotFound, AlreadyCommitted
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── session_lifecycle.go         # Create, AddMessage, GetMessages
├── commit.go                    # 2-phase: archive (phase1) + extract (phase2)
├── compressor.go               # SessionCompressor v1 (legacy) + v2 (template)
├── memory_extractor.go          # LLM extraction → 5 categories
├── memory_deduplicator.go       # Semantic dedup: CREATE/MERGE/SKIP/ARCHIVE
├── working_memory.go            # WM v2 CRUD
├── port/
│   ├── input.go                # SessionUseCase, CommitUseCase
│   └── output.go              # FileWriterPort, LLMPort, EventPublisherPort
└── dto/
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/handler.go              # OvSessionService gRPC
├── event/publisher.go           # NATS: ov.session.committed, ov.session.memory.extracted
└── client/
    ├── fs_client.go             # ov-fs gRPC (write archives + memories)
    └── llm_client.go            # Bifrost LLM (extraction + compression)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/
│   ├── session_repo.go          # PostgreSQL session repo
│   └── message_repo.go          # Message persistence
├── config/config.go
└── wire/wire.go
```

## 3. gRPC API

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

## 4. NATS Events

### Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `ov.session.committed` | `{session_id, account_id, archive_path}` | Phase 1 complete |
| `ov.session.memory.extracted` | `{session_id, memories[], fs_paths[]}` | Phase 2 complete |

## 5. Data Model

- **ov_sessions**: Session metadata + status + archive_path
- **ov_messages**: Message content + role + sequence
- **ov_working_memory**: WM v2 JSONB (goals, facts, errors, context)
- **ov_extracted_memories**: Category + content + dedup_action + fs_path

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Purpose |
|---------|-----------|----------|---------|
| ov-fs | Outbound | gRPC | Write archives + extracted memories |
| Bifrost (LLM) | Outbound | gRPC | Memory extraction, session compression |
| ov-search | Outbound (NATS) | Async | Hotness boost via `ov.session.committed` |
| ov-fs | Outbound (NATS) | Async | Write memories via `ov.session.memory.extracted` |

## 7. Observability

- **Metrics**: Session created/committed, commit duration, extraction by category, dedup actions
- **Traces**: OTel spans: `ov-session.CommitSession` (Phase1 + Phase2 sub-spans)
- **Health**: gRPC Health v1 + HTTP `/healthz` on port 9106

## 8. Multi-Tenancy

- `account_id` in all queries, session namespace: `viking://{account}/{user}/sessions/`

---

> **Next Steps**: Decompose into FEAT specs: FEAT-001 (Session Lifecycle), FEAT-002 (2-Phase Commit), FEAT-003 (Memory Extraction), FEAT-004 (Working Memory v2), FEAT-005 (Deduplication).
