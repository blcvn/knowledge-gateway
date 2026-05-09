# 04 — OpenViking Session Service

> **Service**: `openviking-session`  
> **Port**: 9013 (gRPC) · 9093 (Health/Metrics)  
> **Origin**: L2 SessionService + L4 Session (2629 lines) + L4 SessionCompressor  
> **Role**: Session lifecycle, two-phase commit, Working Memory v2, memory extraction

---

## 1. Responsibilities

| Capability | Description |
|-----------|-------------|
| **Session CRUD** | Create, get, list, delete sessions |
| **Message Management** | Add messages, record used URIs |
| **Two-Phase Commit** | Phase 1: Archive (lock-protected). Phase 2: Memory extract (background) |
| **Working Memory v2** | 7-section structured document generated via VLM |
| **Memory Extraction** | Extract 8 categories of memories from session history via VLM |
| **Redo Log** | Crash safety for Phase 2 background processing |
| **Token Accounting** | Track pending_tokens, keep_recent_count for sliding window |

---

## 2. Clean Architecture Layout

```
services/openviking-session/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── session.go                  # Session, SessionMeta
│   │   ├── message.go                  # Message, Part (Text/Tool/Context)
│   │   ├── working_memory.go           # WorkingMemoryV2, 7 sections
│   │   ├── memory_category.go          # 8 memory categories enum
│   │   ├── commit_phase.go             # CommitPhase1Result, Phase2Task
│   │   ├── redo_log.go                 # RedoLogEntry, RedoState
│   │   └── errors.go
│   ├── usecase/
│   │   ├── create_session.go
│   │   ├── add_messages.go
│   │   ├── record_used.go
│   │   ├── commit_session.go           # Two-phase commit orchestration
│   │   ├── archive_phase.go            # Phase 1: lock + archive
│   │   ├── extract_memory.go           # Phase 2: WM v2 + memory extraction
│   │   ├── generate_working_memory.go  # WM v2 via VLM
│   │   ├── delete_session.go
│   │   ├── replay_redo_log.go          # Crash recovery on startup
│   │   ├── port/
│   │   │   ├── input.go               # SessionUseCase interfaces
│   │   │   └── output.go             # SessionStore, FSClient, VLMClient
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── session_store/         # Session storage (via FS service)
│   │   ├── client/
│   │   │   ├── fs_client.go           # Read/write session files
│   │   │   ├── search_client.go       # Context-aware retrieval
│   │   │   └── vlm_client.go          # VLM for WM v2 + memory extract
│   │   └── event/
│   │       └── publisher.go            # NATS: ov.session.committed/memory.extracted
│   └── infra/
```

---

## 3. Session Data Structure (in VikingFS)

```
viking://session/{session_id}/
├── messages.jsonl              # Live messages (retained tail)
├── .meta.json                  # SessionMeta
└── history/
    ├── archive_001/
    │   ├── messages.jsonl      # Archived messages
    │   └── .overview.md        # Working Memory v2
    ├── archive_002/
    │   └── ...
    └── archive_NNN/
```

---

## 4. Two-Phase Commit

### Phase 1 — Archive (Synchronous, Lock-Protected)

```
1. Acquire PathLock (point) on session directory via FS service
2. Load current messages from messages.jsonl
3. Split: messages[:-keep_recent] → archive
         messages[-keep_recent:] → retained
4. Write retained → messages.jsonl (overwrite)
5. Write archive → history/archive_{N}/messages.jsonl
6. Update .meta.json (commit_count++, message_count)
7. Release PathLock
8. Return Phase1Result to caller
```

### Phase 2 — Memory Extract (Background, Go goroutine)

```
1. Write redo-log marker (crash safety)
2. Wait for any previous Phase 2 to complete
3. Generate Working Memory v2:
   - Read previous WM v2 (if exists) from FS service
   - Feed archived messages to VLM via Bifrost
   - VLM returns section-level operations: KEEP / UPDATE / APPEND
   - Write updated .overview.md to FS service
4. Extract memories via VLM (8 categories):
   - profile, preferences, entities, events
   - cases, patterns, tools, skills
5. For each extracted memory:
   a. Write to viking://user/{id}/memories/{category}/ via FS service
   b. FS service auto-emits ov.content.written → Search indexes it
6. Update active_count on used URIs via Search service
7. Mark redo-log as committed
8. Publish ov.session.memory.extracted
```

---

## 5. Working Memory v2 — 7 Sections

| Section | Description |
|---------|-------------|
| Session Title | Auto-generated conversation title |
| Current State | What the agent is currently doing |
| Task & Goals | Objectives and sub-goals |
| Key Facts & Decisions | Important discoveries |
| Files & Context | URIs and files referenced |
| Errors & Corrections | Mistakes and fixes |
| Open Issues | Unresolved items |

Section update operations: `KEEP` | `UPDATE` (full replace) | `APPEND` (add items)

---

## 6. Memory Categories (8 types)

```go
var MemoryCategories = []string{
    "profile",       // User profile facts
    "preferences",   // Preferences and opinions
    "entities",      // Named entities (people, orgs)
    "events",        // Time-bound events
    "cases",         // Problem-solving cases
    "patterns",      // Behavioral patterns
    "tools",         // Tool usage patterns
    "skills",        // Learned skills
}
```

---

## 7. gRPC Service Definition

```protobuf
service SessionService {
  rpc CreateSession(CreateSessionRequest) returns (CreateSessionResponse);
  rpc GetSession(GetSessionRequest) returns (GetSessionResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc AddMessages(AddMessagesRequest) returns (AddMessagesResponse);
  rpc RecordUsed(RecordUsedRequest) returns (RecordUsedResponse);
  rpc Commit(CommitRequest) returns (CommitResponse);
  rpc DeleteSession(DeleteSessionRequest) returns (DeleteSessionResponse);
  rpc GetSessionInfo(GetSessionInfoRequest) returns (GetSessionInfoResponse);
}
```

---

## 8. Crash Recovery

```
Server startup:
  → Scan redo-log directory
  → For each uncommitted entry:
    → Re-execute Phase 2 (idempotent)
    → Mark committed on success
  → Log recovery results
```
