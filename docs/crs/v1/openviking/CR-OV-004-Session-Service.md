# Change Request: CR-OV-004 — Session Service (Two-Phase Commit & Working Memory v2)

**CR ID:** CR-OV-004  
**Component:** `services/openviking-session` [NEW SERVICE]  
**Priority:** High  
**Status:** Implemented
**Reference:** OpenViking PRD §4.4, SRS §2.4, specs/services/04-session-service.md  
**Maps from Python:** `session/session.py` (2629 lines), `session/compressor.py`, `service/session_service.py`

---

## 1. Mô tả

Xây dựng **openviking-session** — quản lý vòng đời session với Two-Phase Commit và Working Memory v2:

1. **Session CRUD**: Tạo, đọc, liệt kê, xóa sessions.
2. **Message Management**: Thêm messages (Text/Tool/Context parts), track URIs đã dùng.
3. **Two-Phase Commit**:
   - **Phase 1** (lock-protected, sync): Archive messages cũ, giữ lại tail gần nhất.
   - **Phase 2** (background goroutine): Generate Working Memory v2, extract 8 loại memories, update hotness.
4. **Working Memory v2 (WM v2)**: 7-section structured Markdown document tự động cập nhật bởi VLM.
5. **Memory Extraction**: Extract 8 categories (profile, preferences, entities, events, cases, patterns, tools, skills) → lưu vào `viking://user/{id}/memories/`.
6. **Redo Log**: Crash safety cho Phase 2 (write-ahead log, replay on startup).
7. **Token Accounting**: Track `pending_tokens`, `keep_recent_count` để kiểm soát sliding window.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có automated memory extraction từ conversations.
- Thiếu session compression (nén chat cũ thành structured document).
- Không có Working Memory concept — không track state/goals/decisions của agent.
- Thiếu crash safety cho background processing.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/openviking-session/` (Port gRPC: 9013)

### 3.2. Session Data Structure trong VikingFS

```
viking://session/{session_id}/
├── messages.jsonl              # Live messages (retained tail after commit)
├── .meta.json                  # SessionMeta
└── history/
    ├── archive_001/
    │   ├── messages.jsonl      # Archived messages (Phase 1 output)
    │   └── .overview.md        # Working Memory v2 (Phase 2 output)
    ├── archive_002/
    │   └── ...
    └── archive_NNN/
```

### 3.3. Domain Model

```go
// domain/session.go
type Session struct {
    SessionID   string
    AccountID   string
    UserID      string
    AgentID     *string
    CreatedAt   time.Time
    LastCommitAt *time.Time
}

type SessionMeta struct {
    SessionID           string
    CreatedAt           string
    ParticipantUserIDs  []string
    ParticipantAgentIDs []string
    MessageCount        int
    CommitCount         int
    MemoriesExtracted   map[string]int   // category → count
    LLMTokenUsage       map[string]int   // type → token count
    PendingTokens       int              // Tokens in current window
    KeepRecentCount     int              // How many recent messages to retain
}

// domain/message.go
type Message struct {
    ID          string
    Role        string          // "user" | "assistant" | "system" | "tool"
    Parts       []MessagePart
    CreatedAt   time.Time
}

type MessagePart interface{ partType() string }

type TextPart struct {
    Text string
}

type ToolCallPart struct {
    ToolName  string
    Arguments map[string]any
    Result    string
}

type ContextPart struct {
    URI      string    // viking:// URI of referenced context
    Content  string    // Content at time of reference
    Level    int       // L0/L1/L2
}
```

### 3.4. Two-Phase Commit — Chi tiết

#### Phase 1: Archive (Synchronous, Lock-Protected)

```go
// usecase/archive_phase.go
func (uc *ArchivePhaseUseCase) Execute(ctx context.Context, sessionID string) (*Phase1Result, error) {
    // 1. Acquire PathLock (point) on session directory via FS service
    lock, err := uc.fsClient.AcquireLock(ctx, sessionURI, LockModePoint)
    defer lock.Release()
    
    // 2. Load current messages.jsonl from FS service
    messages, err := uc.fsClient.ReadJSONL(ctx, messagesURI)
    
    // 3. Split messages
    keepCount := uc.calculateKeepCount(messages) // based on token budget
    archiveMessages := messages[:len(messages)-keepCount]
    retainMessages  := messages[len(messages)-keepCount:]
    
    // 4. Write retained → messages.jsonl (overwrite)
    uc.fsClient.Write(ctx, messagesURI, toJSONL(retainMessages))
    
    // 5. Create archive directory
    archiveURI := fmt.Sprintf("%s/history/archive_%03d/", sessionURI, commitCount+1)
    uc.fsClient.Mkdir(ctx, archiveURI)
    
    // 6. Write archive messages
    uc.fsClient.Write(ctx, archiveURI+"messages.jsonl", toJSONL(archiveMessages))
    
    // 7. Update .meta.json (commit_count++, pending_tokens reset)
    uc.fsClient.Write(ctx, sessionURI+".meta.json", updatedMeta)
    
    // 8. Release PathLock (defer)
    return &Phase1Result{ArchiveURI: archiveURI, ArchivedCount: len(archiveMessages)}, nil
}
```

#### Phase 2: Memory Extract (Background Goroutine)

```go
// usecase/extract_memory.go
func (uc *ExtractMemoryUseCase) Execute(ctx context.Context, p1 Phase1Result) error {
    // 1. Write redo-log marker (crash safety)
    uc.redoLog.Write(p1.SessionID, p1.ArchiveURI, StateStarted)
    
    // 2. Generate Working Memory v2 via VLM
    prevWM := uc.fsClient.Read(ctx, p1.ArchiveURI+".overview.md")  // Previous WM (may not exist)
    archivedMsgs := uc.fsClient.ReadJSONL(ctx, p1.ArchiveURI+"messages.jsonl")
    
    // VLM prompt: "Given previous WM + new messages, generate section operations"
    wmOps, _ := uc.vlmClient.GenerateWMv2(ctx, prevWM, archivedMsgs)
    // Operations per section: KEEP | UPDATE (full replace) | APPEND (add items)
    
    newWM := applyWMOperations(prevWM, wmOps)
    uc.fsClient.Write(ctx, p1.ArchiveURI+".overview.md", newWM)  // Emit ov.content.written
    
    // 3. Extract 8 categories of memories via VLM
    for _, category := range MemoryCategories {
        memories, _ := uc.vlmClient.ExtractMemories(ctx, archivedMsgs, category)
        for _, memory := range memories {
            memURI := fmt.Sprintf("viking://user/%s/memories/%s/%s.md", userID, category, uuid.New())
            uc.fsClient.Write(ctx, memURI, memory.Content)
            // FS auto-emits ov.content.written → Search indexes it
        }
    }
    
    // 4. Update hotness for used URIs
    usedURIs := extractContextURIs(archivedMsgs)
    uc.searchClient.UpdateHotness(ctx, usedURIs)
    
    // 5. Mark redo-log as committed
    uc.redoLog.MarkCommitted(p1.SessionID, p1.ArchiveURI)
    
    // 6. Publish event
    uc.publisher.PublishMemoryExtracted(ctx, p1.SessionID, MemoryStats{...})
    return nil
}
```

### 3.5. Working Memory v2 — 7 Sections

```markdown
# Session Title
[Auto-generated from conversation context]

## Current State
[What the agent is currently doing]

## Task & Goals
- Primary goal: ...
- Sub-goals: ...

## Key Facts & Decisions
- Decided to use X approach because Y
- Discovered that Z

## Files & Context
- viking://resources/myrepo/src/main.go — Main entry point
- viking://user/alice/memories/skills/go-patterns.md

## Errors & Corrections
- Initially tried X, failed because Y, fixed with Z

## Open Issues
- Need to handle edge case for null input
- TODO: add unit test for session commit
```

**Section update operations** (returned by VLM):
- `KEEP` — Không thay đổi section này
- `UPDATE` — Replace toàn bộ section với nội dung mới
- `APPEND` — Thêm items vào cuối section

### 3.6. 8 Memory Categories

```go
var MemoryCategories = []string{
    "profile",      // User profile facts (name, role, background)
    "preferences",  // Preferences and opinions (tools, styles, habits)
    "entities",     // Named entities (people, orgs, projects)
    "events",       // Time-bound events (meetings, launches, deadlines)
    "cases",        // Problem-solving cases (bugs fixed, decisions made)
    "patterns",     // Behavioral patterns (coding style, decision tendencies)
    "tools",        // Tool usage patterns (preferred commands, workflows)
    "skills",       // Learned skills (capabilities demonstrated)
}
```

### 3.7. Crash Recovery (Redo Log)

```go
// Startup scan:
func (uc *ReplayRedoLogUseCase) Execute(ctx context.Context) error {
    entries, _ := uc.redoLog.FindUncommitted()
    for _, entry := range entries {
        log.Info("recovering uncommitted Phase 2", "session", entry.SessionID)
        // Re-execute Phase 2 (all operations are idempotent)
        uc.extractMemory.Execute(ctx, Phase1Result{from: entry})
    }
    return nil
}
```

### 3.8. gRPC Service Definition

```protobuf
service SessionService {
  rpc CreateSession(CreateSessionRequest) returns (CreateSessionResponse);
  rpc GetSession(GetSessionRequest) returns (GetSessionResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc AddMessages(AddMessagesRequest) returns (AddMessagesResponse);
  rpc RecordUsed(RecordUsedRequest) returns (RecordUsedResponse);  // Track used context URIs
  rpc Commit(CommitRequest) returns (CommitResponse);               // Trigger 2-phase commit
  rpc DeleteSession(DeleteSessionRequest) returns (DeleteSessionResponse);
  rpc GetSessionInfo(GetSessionInfoRequest) returns (GetSessionInfoResponse);
}

message AddMessagesRequest {
  string session_id = 1;
  string account_id = 2;
  repeated MessageProto messages = 3;
}

message MessageProto {
  string id = 1;
  string role = 2;          // user | assistant | system | tool
  repeated MessagePartProto parts = 3;
  google.protobuf.Timestamp created_at = 4;
}

message CommitRequest {
  string session_id = 1;
  string account_id = 2;
  bool force = 3;           // Force commit even below token threshold
}

message CommitResponse {
  int32 archived_count = 1;
  int32 retained_count = 2;
  string archive_uri = 3;
  bool phase2_started = 4;  // Phase 2 runs in background
}
```

### 3.9. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `ov.session.committed` | `{session_id, archive_uri, used_uris[]}` | Search (update hotness) |
| `ov.session.memory.extracted` | `{session_id, memories_count, categories}` | FS (write memories — already done in Phase 2) |

### 3.10. Configuration

```yaml
session:
  grpc:
    port: 9013
  health:
    port: 9093
  commit:
    token_threshold: 2000          # Auto-commit when pending_tokens exceeds this
    keep_recent_count: 10          # Keep last N messages after archive
    phase2_max_concurrent: 3       # Max concurrent background extractions
  vlm:
    service_url: "bifrost:4000"    # VLM gateway
    wm_model: "gpt-4o-mini"       # Model for WM v2 generation
    extract_model: "gpt-4o-mini"  # Model for memory extraction
  redo_log:
    dir: "~/.openviking/redo_log"
  clients:
    fs: "openviking-fs:9011"
    search: "openviking-search:9012"
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
```

---

## 4. Acceptance Criteria

- [ ] `CreateSession → AddMessages(100 messages) → Commit` → Phase 1 hoàn thành trong < 500ms (sync, với lock).
- [ ] Sau Phase 1: `messages.jsonl` chỉ còn 10 messages (keep_recent); `history/archive_001/messages.jsonl` có 90 messages.
- [ ] Phase 2 chạy background → `history/archive_001/.overview.md` xuất hiện với 7 sections đúng format.
- [ ] Memories được extract → `viking://user/alice/memories/preferences/xxx.md` được tạo và Search indexing nó.
- [ ] Server crash trong Phase 2 → restart → redo log detect uncommitted → Phase 2 tự động retry.
- [ ] `RecordUsed([uri1, uri2])` trong session → sau Commit, Search.UpdateHotness được gọi với list đó.
- [ ] WM v2 section `APPEND` operation → chỉ thêm items mới, không xóa items cũ.
- [ ] Concurrent `Commit` từ 2 goroutines cùng session → PathLock đảm bảo chỉ 1 Phase 1 chạy, cái còn lại đợi.
