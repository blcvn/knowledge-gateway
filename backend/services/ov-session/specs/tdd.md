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

## 7. Core Algorithms

### 7.1. Two-Phase Commit Pipeline

When a session reaches a logical conclusion, it undergoes a two-phase commit:
- **Phase 1 (Archive)**:
  1. Load all uncommitted messages from `ov_messages`.
  2. Compress the chat history using the Bifrost LLM `SessionCompressor`.
  3. Write the compressed archive to `ov-fs` (e.g., `viking://{account}/{user}/sessions/archives/`).
  4. Publish `ov.session.committed` to NATS to boost the hotness of referenced files.
- **Phase 2 (Extract & Deduplicate)**:
  1. Trigger the `MemoryExtractor` against the compressed session archive.
  2. Run the semantic `MemoryDeduplicator` against existing memories.
  3. Write final memories to `ov-fs`.
  4. Publish `ov.session.memory.extracted`.

### 7.2. Working Memory v2 Lifecycle

A JSONB state machine representing active cognitive context:
- **State Fields**: Track `User Context`, `Goals`, `Known Facts`, and `Errors/Obstacles`.
- **Update Logic**: With each `AddMessage`, an LLM evaluator determines if the working memory needs updating. If goals are met, facts are crystallized, or errors occur, the state is patched.

### 7.3. LLM Memory Extraction (5 Categories)

The extractor scans the session and classifies raw candidate memories into 5 domains:
1. **User Persona**: Traits, preferences, writing style.
2. **Project Context**: Active codebases, tech stack, architectures.
3. **Decisions**: Architectural or business decisions agreed upon.
4. **Action Items**: Pending tasks or next steps.
5. **Errors/Blockers**: Persistent bugs or constraints.

### 7.4. Semantic Deduplication Algorithm

For each extracted candidate memory, the system uses semantic similarity to determine the storage action:
1. Calculate similarity with existing memories in the same category.
2. If `sim == 1.0` (exact duplicate): Action = **SKIP**.
3. If `sim > 0.85` (high overlap): Action = **MERGE** (ask LLM to fuse them).
4. If `sim > 0.60` (related): Action = **CREATE** (store as distinct but related).
5. If `sim < 0.60` (new topic): Action = **CREATE**.
6. If a memory is invalidated by a new fact: Action = **ARCHIVE** (soft delete).

## 8. Observability

- **Metrics**: Session created/committed, commit duration, extraction by category, dedup actions
- **Traces**: OTel spans: `ov-session.CommitSession` (Phase1 + Phase2 sub-spans)
- **Health**: gRPC Health v1 + HTTP `/healthz` on port 9106

## 9. Multi-Tenancy

- `account_id` in all queries, session namespace: `viking://{account}/{user}/sessions/`

---

> **Next Steps**: Decompose into FEAT specs: FEAT-001 (Session Lifecycle), FEAT-002 (2-Phase Commit), FEAT-003 (Memory Extraction), FEAT-004 (Working Memory v2), FEAT-005 (Deduplication).
