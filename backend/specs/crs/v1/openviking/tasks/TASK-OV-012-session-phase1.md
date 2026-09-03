# TASK-OV-012 — `services/openviking-session` Phase 1 Archive & AddMessages

**Wave:** 5 (Context)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-009 (fs client), TASK-OV-011 (search client)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-OV-004 §3, §4.1, §4.2, §4.6](../solutions/SOL-OV-004-Session-Service.md)  
**Port gRPC:** 9013

---

## Mục tiêu

Tạo phần đầu của `services/openviking-session/` — Domain model, AddMessages (với token tracking và auto-commit trigger), và Phase 1 Archive (synchronous, lock-protected session commit).

---

## Cấu trúc thư mục

```
services/openviking-session/
├── cmd/server/main.go
├── api/proto/session/v1/session.proto
├── internal/
│   ├── domain/
│   │   ├── session.go          # Session, SessionMeta
│   │   ├── message.go          # Message, MessagePart (Text/Tool/Context)
│   │   ├── wm_v2.go            # WMv2Section, WMv2Operation, 7 sections
│   │   ├── memory_category.go  # 8 MemoryCategory constants
│   │   └── redo_log.go         # RedoLogEntry, RedoLogState
│   ├── usecase/
│   │   ├── create_session.go
│   │   ├── get_session.go
│   │   ├── list_sessions.go
│   │   ├── add_messages.go     # Append messages + token tracking + auto-commit
│   │   ├── record_used.go      # Track used context URIs
│   │   ├── commit.go           # Orchestrate Phase 1 + kick Phase 2
│   │   ├── phase1_archive.go   # Synchronous archive with PathLock
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go       # FSClient, EventPublisher, SessionMetaStore
```

---

## 1. Domain Models

**File: `internal/domain/session.go`**

```go
type Session struct {
    SessionID           string
    AccountID           string
    UserID              string
    AgentID             *string
    ParticipantUserIDs  []string
    ParticipantAgentIDs []string
    CreatedAt           time.Time
    LastCommitAt        *time.Time
}

type SessionMeta struct {
    SessionID           string             `json:"session_id"`
    AccountID           string             `json:"account_id"`
    UserID              string             `json:"user_id"`
    AgentID             string             `json:"agent_id,omitempty"`
    ParticipantUserIDs  []string           `json:"participant_user_ids"`
    ParticipantAgentIDs []string           `json:"participant_agent_ids"`
    MessageCount        int                `json:"message_count"`
    CommitCount         int                `json:"commit_count"`
    MemoriesExtracted   map[string]int     `json:"memories_extracted"`
    LLMTokenUsage       map[string]int     `json:"llm_token_usage"`
    PendingTokens       int                `json:"pending_tokens"`
    KeepRecentCount     int                `json:"keep_recent_count"`
    UsedURIs            []string           `json:"used_uris"`
    CreatedAt           time.Time          `json:"created_at"`
    LastCommitAt        *time.Time         `json:"last_commit_at,omitempty"`
}

// Session URI convention:
// viking://session/{accountID}/{sessionID}/
// viking://session/{accountID}/{sessionID}/messages.jsonl      ← active messages
// viking://session/{accountID}/{sessionID}/.meta.json          ← session meta
// viking://session/{accountID}/{sessionID}/history/archive_001/ ← archives
```

**File: `internal/domain/message.go`**

```go
type Message struct {
    ID        string        `json:"id"`
    Role      string        `json:"role"`  // user|assistant|system|tool
    Parts     []RawPart     `json:"parts"`
    CreatedAt time.Time     `json:"created_at"`
    TokenSize int           `json:"token_size"`
}

// RawPart: JSON-serializable sum type
// {"type":"text","text":"..."}
// {"type":"tool","tool_name":"...","arguments":{...},"result":"..."}
// {"type":"context","uri":"viking://...","content":"...","level":2}

type RawPart struct {
    Type     string          `json:"type"`
    Text     string          `json:"text,omitempty"`
    ToolName string          `json:"tool_name,omitempty"`
    Result   string          `json:"result,omitempty"`
    URI      string          `json:"uri,omitempty"`
    Content  string          `json:"content,omitempty"`
    Level    int             `json:"level,omitempty"`
}

func EstimateTokens(msg Message) int {
    total := 10  // base per message (role + overhead)
    for _, part := range msg.Parts {
        switch part.Type {
        case "text":
            total += len(part.Text) / 4
        case "tool":
            total += len(part.Result)/4 + 50
        case "context":
            total += len(part.Content) / 4
        }
    }
    return total
}

// Extract all context URIs referenced in a message list
func ExtractContextURIs(messages []Message) []string {
    seen := make(map[string]bool)
    var uris []string
    for _, m := range messages {
        for _, p := range m.Parts {
            if p.Type == "context" && p.URI != "" && !seen[p.URI] {
                seen[p.URI] = true
                uris = append(uris, p.URI)
            }
        }
    }
    return uris
}
```

**File: `internal/domain/wm_v2.go`**

```go
type WMv2SectionID string
const (
    WMSectionTitle        WMv2SectionID = "title"
    WMSectionCurrentState WMv2SectionID = "current_state"
    WMSectionTaskGoals    WMv2SectionID = "task_goals"
    WMSectionKeyFacts     WMv2SectionID = "key_facts"
    WMSectionFilesContext WMv2SectionID = "files_context"
    WMSectionErrors       WMv2SectionID = "errors"
    WMSectionOpenIssues   WMv2SectionID = "open_issues"
)

type WMv2Operation struct {
    SectionID WMv2SectionID `json:"section_id"`
    Op        string        `json:"op"`   // "KEEP" | "UPDATE" | "APPEND"
    Content   string        `json:"content"`
}

type WMv2Section struct {
    ID      WMv2SectionID
    Content string
}

var WMv2SectionOrder = []WMv2SectionID{
    WMSectionTitle, WMSectionCurrentState, WMSectionTaskGoals,
    WMSectionKeyFacts, WMSectionFilesContext, WMSectionErrors, WMSectionOpenIssues,
}

var WMv2SectionHeadings = map[WMv2SectionID]string{
    WMSectionTitle:        "# Session Title",
    WMSectionCurrentState: "## Current State",
    WMSectionTaskGoals:    "## Task & Goals",
    WMSectionKeyFacts:     "## Key Facts & Decisions",
    WMSectionFilesContext: "## Files & Context",
    WMSectionErrors:       "## Errors & Corrections",
    WMSectionOpenIssues:   "## Open Issues",
}

func ApplyWMOperations(current []WMv2Section, ops []WMv2Operation) []WMv2Section
func SerializeWMv2(sections []WMv2Section) string
func ParseWMv2(markdown string) []WMv2Section  // Parse headings back to sections
```

**File: `internal/domain/memory_category.go`**

```go
var MemoryCategories = []string{
    "profile", "preferences", "entities", "events",
    "cases", "patterns", "tools", "skills",
}

var MemoryCategoryDefinitions = map[string]string{
    "profile":     "Personal background, professional history, demographics",
    "preferences": "Preferences, opinions, dislikes, communication style",
    "entities":    "Named entities: people, organizations, projects, products",
    "events":      "Time-bound events, meetings, deadlines, milestones",
    "cases":       "Problem-solving cases, debugging sessions, solutions found",
    "patterns":    "Behavioral patterns, recurring themes, habits",
    "tools":       "Tool usage patterns, configuration preferences, workflows",
    "skills":      "Demonstrated capabilities, expertise areas, learning goals",
}
```

---

## 2. AddMessages UseCase

**File: `internal/usecase/add_messages.go`**

```go
type AddMessagesUseCase struct {
    fsClient    port.FSClient
    commitUC    *CommitUseCase  // Injected (lazy init to avoid circular)
    config      *Config
}

type AddMessagesRequest struct {
    SessionID string
    AccountID string
    Messages  []domain.Message
}

func (uc *AddMessagesUseCase) Execute(ctx context.Context, req AddMessagesRequest) error {
    sessionURI := sessionURI(req.AccountID, req.SessionID)
    messagesURI := sessionURI + "messages.jsonl"

    // 1. Estimate token size for each message
    totalNewTokens := 0
    for i := range req.Messages {
        req.Messages[i].ID = uuid.New().String()
        req.Messages[i].CreatedAt = time.Now()
        req.Messages[i].TokenSize = domain.EstimateTokens(req.Messages[i])
        totalNewTokens += req.Messages[i].TokenSize
    }

    // 2. Serialize each message as JSON line
    lines := make([][]byte, len(req.Messages))
    for i, msg := range req.Messages {
        lines[i], _ = json.Marshal(msg)
    }

    // 3. Append to messages.jsonl
    if err := uc.fsClient.AppendJSONL(ctx, messagesURI, lines); err != nil {
        return fmt.Errorf("append messages: %w", err)
    }

    // 4. Update .meta.json (pending_tokens += totalNewTokens)
    meta, _ := uc.loadMeta(ctx, req.SessionID, req.AccountID)
    meta.PendingTokens += totalNewTokens
    meta.MessageCount  += len(req.Messages)
    uc.fsClient.WriteJSON(ctx, sessionURI+".meta.json", meta)

    // 5. Auto-commit check
    if meta.PendingTokens >= uc.config.TokenThreshold {
        go func() {
            bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
            defer cancel()
            uc.commitUC.Execute(bgCtx, CommitRequest{
                SessionID: req.SessionID,
                AccountID: req.AccountID,
                Force:     false,
            })
        }()
    }

    return nil
}
```

---

## 3. Phase 1 Archive UseCase

**File: `internal/usecase/phase1_archive.go`**

```go
type Phase1Result struct {
    SessionID     string
    AccountID     string
    UserID        string
    ArchiveURI    string
    ArchivedCount int
    RetainedCount int
    CommitNumber  int
    UsedURIs      []string
}

func (uc *Phase1ArchiveUseCase) Execute(ctx context.Context, sessionID, accountID string, meta *domain.SessionMeta) (*Phase1Result, error) {
    sessionURI  := sessionURI(accountID, sessionID)
    messagesURI := sessionURI + "messages.jsonl"

    // 1. Acquire PathLock (point) on session directory
    release, err := uc.fsClient.AcquireLock(ctx, sessionURI, "point")
    if err != nil {
        return nil, &viking.OpenVikingError{Code: viking.ErrResourceBusy, Message: "session commit in progress"}
    }
    defer release()

    // 2. Read all messages from messages.jsonl
    rawLines, err := uc.fsClient.ReadJSONL(ctx, messagesURI)
    if err != nil { return nil, err }

    messages := make([]domain.Message, 0, len(rawLines))
    for _, line := range rawLines {
        var m domain.Message
        json.Unmarshal(line, &m)
        messages = append(messages, m)
    }

    // 3. Split: keep last N, archive rest
    keepCount := meta.KeepRecentCount
    if keepCount == 0 { keepCount = uc.config.KeepRecentCount }
    if keepCount > len(messages) { keepCount = len(messages) }

    archiveMessages := messages[:len(messages)-keepCount]
    retainMessages  := messages[len(messages)-keepCount:]

    if len(archiveMessages) == 0 {
        return nil, fmt.Errorf("no messages to archive: all within keep_recent window")
    }

    // 4. Overwrite messages.jsonl with retained messages only
    retainLines := make([][]byte, len(retainMessages))
    for i, m := range retainMessages {
        retainLines[i], _ = json.Marshal(m)
    }
    uc.fsClient.WriteJSONL(ctx, messagesURI, retainLines)

    // 5. Create archive directory
    commitNum := meta.CommitCount + 1
    archiveURI := fmt.Sprintf("%shistory/archive_%03d/", sessionURI, commitNum)
    uc.fsClient.Mkdir(ctx, archiveURI, false)

    // 6. Write archived messages to history
    archiveLines := make([][]byte, len(archiveMessages))
    for i, m := range archiveMessages { archiveLines[i], _ = json.Marshal(m) }
    uc.fsClient.WriteJSONL(ctx, archiveURI+"messages.jsonl", archiveLines)

    // 7. Update meta
    meta.CommitCount    = commitNum
    meta.PendingTokens  = sumTokens(retainMessages)
    meta.MessageCount   = len(retainMessages)
    now := time.Now()
    meta.LastCommitAt   = &now
    uc.fsClient.WriteJSON(ctx, sessionURI+".meta.json", meta)

    // 8. Extract used context URIs from archived messages
    usedURIs := domain.ExtractContextURIs(archiveMessages)

    return &Phase1Result{
        SessionID:     sessionID,
        AccountID:     accountID,
        UserID:        meta.UserID,
        ArchiveURI:    archiveURI,
        ArchivedCount: len(archiveMessages),
        RetainedCount: len(retainMessages),
        CommitNumber:  commitNum,
        UsedURIs:      usedURIs,
    }, nil
}
```

---

## 4. Commit Orchestrator

**File: `internal/usecase/commit.go`**

```go
type CommitRequest struct {
    SessionID string
    AccountID string
    Force     bool
}

type CommitResponse struct {
    ArchivedCount int
    RetainedCount int
    ArchiveURI    string
    Phase2Started bool
    Skipped       bool
    SkipReason    string
}

func (uc *CommitUseCase) Execute(ctx context.Context, req CommitRequest) (*CommitResponse, error) {
    meta, err := uc.loadMeta(ctx, req.SessionID, req.AccountID)
    if err != nil { return nil, err }

    // Auto-commit threshold check
    if !req.Force && meta.PendingTokens < uc.config.TokenThreshold {
        return &CommitResponse{
            Skipped:    true,
            SkipReason: fmt.Sprintf("pending_tokens=%d < threshold=%d", meta.PendingTokens, uc.config.TokenThreshold),
        }, nil
    }

    // Phase 1: synchronous
    p1, err := uc.phase1.Execute(ctx, req.SessionID, req.AccountID, meta)
    if err != nil { return nil, fmt.Errorf("phase 1: %w", err) }

    // Phase 2: background (non-blocking)
    go func() {
        bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
        defer cancel()
        if err := uc.phase2.Execute(bgCtx, p1, meta); err != nil {
            slog.Error("phase 2 failed", "session_id", req.SessionID, "error", err)
        }
    }()

    return &CommitResponse{
        ArchivedCount: p1.ArchivedCount,
        RetainedCount: p1.RetainedCount,
        ArchiveURI:    p1.ArchiveURI,
        Phase2Started: true,
    }, nil
}
```

---

## 5. Protobuf (partial — full in TASK-OV-013)

```protobuf
service SessionService {
  rpc CreateSession(CreateSessionRequest) returns (CreateSessionResponse);
  rpc GetSession(GetSessionRequest) returns (GetSessionResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc AddMessages(AddMessagesRequest) returns (AddMessagesResponse);
  rpc RecordUsed(RecordUsedRequest) returns (RecordUsedResponse);
  rpc Commit(CommitRequest) returns (CommitResponse);
  rpc DeleteSession(DeleteSessionRequest) returns (DeleteSessionResponse);
  rpc GetSessionContext(GetSessionContextRequest) returns (GetSessionContextResponse);
}

message AddMessagesRequest {
  string session_id  = 1;
  string account_id  = 2;
  repeated MessageProto messages = 3;
}

message CommitResponse {
  int32  archived_count = 1;
  int32  retained_count = 2;
  string archive_uri    = 3;
  bool   phase2_started = 4;
  bool   skipped        = 5;
  string skip_reason    = 6;
}
```

---

## Unit Tests

```
TestEstimateTokens_TextPart            → 400 chars text → ~100 tokens
TestEstimateTokens_ToolPart            → 400 chars result → ~100 + 50 tokens
TestExtractContextURIs_FromMessages    → context part with URI → extracted
TestExtractContextURIs_DeduplicatesURI → same URI twice → returned once
TestApplyWMOperations_Update           → UPDATE op → replaces content
TestApplyWMOperations_Append           → APPEND op → appends to existing
TestApplyWMOperations_Keep             → KEEP op → content unchanged
TestSerializeWMv2                      → 7 sections → correct markdown headings
TestParseWMv2RoundTrip                 → serialize then parse → same sections
TestAddMessages_AppendToJSONL          → 3 messages → 3 lines in file
TestAddMessages_UpdatesPendingTokens   → tokens summed correctly
TestAddMessages_AutoCommitTriggered    → tokens > threshold → commit goroutine started
TestAddMessages_AutoCommitNotTriggered → tokens < threshold → no commit
TestPhase1Archive_Splits               → 15 messages, keep=5 → archived=10, retained=5
TestPhase1Archive_NothingToArchive     → 5 messages, keep=10 → error
TestPhase1Archive_CreatesArchiveDir    → archiveURI created in FS
TestPhase1Archive_PathLockPrevents     → 2 goroutines commit → sequential (not race)
TestPhase1Archive_ExtractsUsedURIs     → archived msgs with context parts → URIs extracted
TestPhase1Archive_UpdatesMeta          → CommitCount++, PendingTokens updated
TestCommitUseCase_SkippedBelowThreshold → tokens < threshold, !force → Skipped=true
TestCommitUseCase_Phase2Async          → Phase 1 ok → Phase2Started=true
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
buf generate services/openviking-session/
go build ./services/openviking-session/internal/...
go test ./services/openviking-session/internal/... -v -count=1 -race
```

---

## Ghi chú triển khai

- `WriteJSONL` overwrites file (used for retained messages); `AppendJSONL` adds lines
- Session URI helper: `func sessionURI(accountID, sessionID string) string { return "viking://session/" + accountID + "/" + sessionID + "/" }`
- `LoadMeta`: ReadRaw `.meta.json` → unmarshal → return `&SessionMeta`; if not found → return empty meta with defaults
- Lock token management: `AcquireLock` returns gRPC lock token (UUID string), must pass to `ReleaseLock`
