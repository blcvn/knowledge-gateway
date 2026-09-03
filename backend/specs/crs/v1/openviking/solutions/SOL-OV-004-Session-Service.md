# Solution: SOL-OV-004 — Session Service (Two-Phase Commit & Working Memory v2)

**CR:** [CR-OV-004](../CR-OV-004-Session-Service.md)  
**Wave:** 5 (Context — sau Search service)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/openviking-session` — quản lý vòng đời session với **Two-Phase Commit** crash-safe và **Working Memory v2** (7-section Markdown tự động cập nhật bởi VLM).

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Không có session compression | Phase 1: Archive messages cũ, retain tail |
| Không có WM v2 | Phase 2: VLM generates 7-section structured Markdown |
| Không có memory extraction | Phase 2: VLM extracts 8 categories → write to FS |
| Crash safety cho Phase 2 | Redo log (write-ahead log) trước khi Phase 2 bắt đầu |
| Concurrent commit race | PathLock (point) trên session directory |
| Token tracking | `pending_tokens` counter → trigger auto-commit |

---

## 2. Codebase Structure

```
services/openviking-session/
├── cmd/server/main.go
├── api/proto/session/v1/session.proto
├── internal/
│   ├── domain/
│   │   ├── session.go       # Session, SessionMeta
│   │   ├── message.go       # Message, MessagePart (Text|Tool|Context)
│   │   ├── wm_v2.go         # WMv2Section, WMv2Operations, 7 sections
│   │   ├── memory_category.go  # 8 MemoryCategory constants
│   │   ├── redo_log.go      # RedoLogEntry, RedoLogState
│   │   └── errors.go
│   ├── usecase/
│   │   ├── create_session.go
│   │   ├── get_session.go
│   │   ├── add_messages.go   # Append messages + update pending_tokens
│   │   ├── commit.go         # Orchestrate Phase 1 + kick off Phase 2
│   │   ├── phase1_archive.go # Lock-protected synchronous archive
│   │   ├── phase2_extract.go # Background memory extraction
│   │   ├── wm_v2.go          # WM v2 generation logic
│   │   ├── redo_log.go       # Crash recovery on startup
│   │   ├── record_used.go    # Track used context URIs
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go     # FSClient, VLMClient, SearchClient, EventPublisher, RedoLogStore
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   ├── client/
│   │   │   ├── fs_client.go     # gRPC → openviking-fs:9011
│   │   │   ├── search_client.go # gRPC → openviking-search:9012
│   │   │   └── vlm_client.go    # via Bifrost
│   │   ├── store/
│   │   │   ├── redo_log/        # File-based redo log
│   │   │   └── session_meta/    # Session metadata (JSON in VikingFS)
│   │   └── event/
│   │       ├── publisher.go     # ov.session.committed, ov.session.memory.extracted
│   │       └── subscriber.go    # admin.account.deleted
│   └── infra/
```

---

## 3. Domain Model

### 3.1 Session & Messages

```go
// internal/domain/session.go

type Session struct {
    SessionID    string
    AccountID    string
    UserID       string
    AgentID      *string
    CreatedAt    time.Time
    LastCommitAt *time.Time
}

type SessionMeta struct {
    SessionID           string             `json:"session_id"`
    CreatedAt           string             `json:"created_at"`
    ParticipantUserIDs  []string           `json:"participant_user_ids"`
    ParticipantAgentIDs []string           `json:"participant_agent_ids"`
    MessageCount        int                `json:"message_count"`
    CommitCount         int                `json:"commit_count"`
    MemoriesExtracted   map[string]int     `json:"memories_extracted"`   // category → count
    LLMTokenUsage       map[string]int     `json:"llm_token_usage"`
    PendingTokens       int                `json:"pending_tokens"`
    KeepRecentCount     int                `json:"keep_recent_count"`
    UsedURIs            []string           `json:"used_uris"`            // Context URIs accessed this session
}

// internal/domain/message.go

type Message struct {
    ID        string
    Role      string // "user" | "assistant" | "system" | "tool"
    Parts     []MessagePart
    CreatedAt time.Time
    TokenSize int
}

type MessagePart interface {
    partType() string
    TokenCount() int
}

type TextPart struct {
    Text string
}
func (t *TextPart) partType() string  { return "text" }
func (t *TextPart) TokenCount() int   { return estimateTokens(t.Text) }

type ToolCallPart struct {
    ToolName  string
    Arguments map[string]any
    Result    string
}
func (t *ToolCallPart) partType() string  { return "tool" }
func (t *ToolCallPart) TokenCount() int   { return estimateTokens(t.Result) + 50 }

type ContextPart struct {
    URI     string  // viking:// URI of referenced context
    Content string  // Content at time of reference
    Level   int     // L0/L1/L2
}
func (c *ContextPart) partType() string  { return "context" }
func (c *ContextPart) TokenCount() int   { return estimateTokens(c.Content) }
```

### 3.2 Working Memory v2 — 7 Sections

```go
// internal/domain/wm_v2.go

type WMv2SectionID string
const (
    WMSectionTitle       WMv2SectionID = "title"
    WMSectionCurrentState WMv2SectionID = "current_state"
    WMSectionTaskGoals   WMv2SectionID = "task_goals"
    WMSectionKeyFacts    WMv2SectionID = "key_facts"
    WMSectionFilesContext WMv2SectionID = "files_context"
    WMSectionErrors      WMv2SectionID = "errors"
    WMSectionOpenIssues  WMv2SectionID = "open_issues"
)

type WMv2Section struct {
    ID      WMv2SectionID
    Content string
}

type WMv2OperationType string
const (
    WMOpKeep   WMv2OperationType = "KEEP"    // Không thay đổi
    WMOpUpdate WMv2OperationType = "UPDATE"  // Replace toàn bộ
    WMOpAppend WMv2OperationType = "APPEND"  // Thêm items vào cuối
)

type WMv2Operation struct {
    SectionID WMv2SectionID
    Op        WMv2OperationType
    Content   string // For UPDATE: new content; for APPEND: items to add
}

// VLM prompt response format:
// {"operations": [
//   {"section_id": "current_state", "op": "UPDATE", "content": "Agent is debugging auth..."},
//   {"section_id": "key_facts", "op": "APPEND", "content": "- Found null pointer in auth.go:45"},
//   {"section_id": "errors", "op": "KEEP", "content": ""}
// ]}

func ApplyWMOperations(current []WMv2Section, ops []WMv2Operation) []WMv2Section {
    sectionMap := make(map[WMv2SectionID]string)
    for _, s := range current {
        sectionMap[s.ID] = s.Content
    }
    
    for _, op := range ops {
        switch op.Op {
        case WMOpKeep:
            // No change
        case WMOpUpdate:
            sectionMap[op.SectionID] = op.Content
        case WMOpAppend:
            existing := sectionMap[op.SectionID]
            sectionMap[op.SectionID] = existing + "\n" + op.Content
        }
    }
    
    // Reconstruct in fixed order
    result := make([]WMv2Section, 0, 7)
    for _, id := range []WMv2SectionID{
        WMSectionTitle, WMSectionCurrentState, WMSectionTaskGoals,
        WMSectionKeyFacts, WMSectionFilesContext, WMSectionErrors, WMSectionOpenIssues,
    } {
        result = append(result, WMv2Section{ID: id, Content: sectionMap[id]})
    }
    return result
}

func SerializeWMv2(sections []WMv2Section) string {
    var sb strings.Builder
    headings := map[WMv2SectionID]string{
        WMSectionTitle:        "# Session Title",
        WMSectionCurrentState: "## Current State",
        WMSectionTaskGoals:    "## Task & Goals",
        WMSectionKeyFacts:     "## Key Facts & Decisions",
        WMSectionFilesContext: "## Files & Context",
        WMSectionErrors:       "## Errors & Corrections",
        WMSectionOpenIssues:   "## Open Issues",
    }
    for _, s := range sections {
        if s.Content == "" {
            continue
        }
        sb.WriteString(headings[s.ID] + "\n")
        sb.WriteString(s.Content + "\n\n")
    }
    return sb.String()
}
```

### 3.3 Memory Categories

```go
// internal/domain/memory_category.go

var MemoryCategories = []string{
    "profile",      // User background facts
    "preferences",  // Preferences and opinions
    "entities",     // Named entities (people, orgs, projects)
    "events",       // Time-bound events
    "cases",        // Problem-solving cases
    "patterns",     // Behavioral patterns
    "tools",        // Tool usage patterns
    "skills",       // Demonstrated capabilities
}
```

---

## 4. Two-Phase Commit — Chi tiết Implementation

### 4.1 CommitUseCase — Orchestrator

```go
// internal/usecase/commit.go

type CommitUseCase struct {
    phase1   *Phase1ArchiveUseCase
    phase2   *Phase2ExtractUseCase
    fsClient port.FSClient
    config   *Config
}

func (uc *CommitUseCase) Execute(ctx context.Context, req dto.CommitRequest) (*dto.CommitResponse, error) {
    // Load session meta to check if commit needed
    meta, err := uc.loadMeta(ctx, req.SessionID, req.AccountID)
    if err != nil {
        return nil, err
    }
    
    // Auto-commit check (unless force=true)
    if !req.Force && meta.PendingTokens < uc.config.TokenThreshold {
        return &dto.CommitResponse{
            Skipped: true,
            Reason:  fmt.Sprintf("pending_tokens=%d < threshold=%d", meta.PendingTokens, uc.config.TokenThreshold),
        }, nil
    }
    
    // Phase 1: Synchronous, lock-protected
    p1Result, err := uc.phase1.Execute(ctx, req.SessionID, req.AccountID, meta)
    if err != nil {
        return nil, fmt.Errorf("phase 1 failed: %w", err)
    }
    
    // Phase 2: Background goroutine
    go func() {
        bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
        defer cancel()
        if err := uc.phase2.Execute(bgCtx, p1Result, meta); err != nil {
            slog.Error("phase 2 failed", "session_id", req.SessionID, "error", err)
        }
    }()
    
    return &dto.CommitResponse{
        ArchivedCount:  p1Result.ArchivedCount,
        RetainedCount:  p1Result.RetainedCount,
        ArchiveURI:     p1Result.ArchiveURI,
        Phase2Started:  true,
    }, nil
}
```

### 4.2 Phase 1 — Archive (Synchronous, Lock-Protected)

```go
// internal/usecase/phase1_archive.go

type Phase1Result struct {
    SessionID     string
    AccountID     string
    UserID        string
    ArchiveURI    string
    ArchivedCount int
    RetainedCount int
    CommitNumber  int
    UsedURIs      []string // For hotness update
}

func (uc *Phase1ArchiveUseCase) Execute(ctx context.Context, sessionID, accountID string, meta *domain.SessionMeta) (*Phase1Result, error) {
    sessionURI := fmt.Sprintf("viking://session/%s/%s/", accountID, sessionID)
    messagesURI := sessionURI + "messages.jsonl"
    
    // 1. Acquire PathLock (point) on session directory
    // Prevents concurrent commits from same session
    release, err := uc.fsClient.AcquireLock(ctx, sessionURI, "point")
    if err != nil {
        return nil, &viking.OpenVikingError{
            Code:    viking.ErrResourceBusy,
            Message: "session is being committed by another process",
        }
    }
    defer release()
    
    // 2. Load current messages.jsonl
    rawMessages, err := uc.fsClient.ReadJSONL(ctx, messagesURI)
    if err != nil {
        return nil, fmt.Errorf("read messages: %w", err)
    }
    var messages []domain.Message
    for _, line := range rawMessages {
        var m domain.Message
        json.Unmarshal(line, &m)
        messages = append(messages, m)
    }
    
    // 3. Split messages: retain recent N, archive the rest
    keepCount := uc.config.KeepRecentCount
    if keepCount > len(messages) {
        keepCount = len(messages)
    }
    archiveMessages := messages[:len(messages)-keepCount]
    retainMessages  := messages[len(messages)-keepCount:]
    
    if len(archiveMessages) == 0 {
        return nil, fmt.Errorf("nothing to archive: all messages within keep_recent window")
    }
    
    // 4. Overwrite messages.jsonl with retained messages only
    err = uc.fsClient.WriteJSONL(ctx, messagesURI, toJSONLines(retainMessages))
    if err != nil {
        return nil, fmt.Errorf("write retained messages: %w", err)
    }
    
    // 5. Create archive directory
    commitNum := meta.CommitCount + 1
    archiveURI := fmt.Sprintf("%shistory/archive_%03d/", sessionURI, commitNum)
    uc.fsClient.Mkdir(ctx, archiveURI, false)
    
    // 6. Write archived messages to history
    uc.fsClient.WriteJSONL(ctx, archiveURI+"messages.jsonl", toJSONLines(archiveMessages))
    
    // 7. Update .meta.json
    meta.CommitCount     = commitNum
    meta.PendingTokens   = sumTokens(retainMessages)
    meta.MessageCount    = len(retainMessages)
    lastCommitAt         := time.Now().Format(time.RFC3339)
    uc.fsClient.WriteJSON(ctx, sessionURI+".meta.json", meta)
    _ = lastCommitAt
    
    // Collect used URIs from archived messages (ContextParts)
    usedURIs := extractContextURIs(archiveMessages)
    
    return &Phase1Result{
        SessionID:     sessionID,
        AccountID:     accountID,
        UserID:        meta.ParticipantUserIDs[0],
        ArchiveURI:    archiveURI,
        ArchivedCount: len(archiveMessages),
        RetainedCount: len(retainMessages),
        CommitNumber:  commitNum,
        UsedURIs:      usedURIs,
    }, nil
}
```

### 4.3 Phase 2 — Memory Extraction (Background)

```go
// internal/usecase/phase2_extract.go

func (uc *Phase2ExtractUseCase) Execute(ctx context.Context, p1 *Phase1Result, meta *domain.SessionMeta) error {
    // 1. Write redo-log BEFORE doing anything (crash safety)
    entry := &domain.RedoLogEntry{
        SessionID:  p1.SessionID,
        AccountID:  p1.AccountID,
        ArchiveURI: p1.ArchiveURI,
        State:      domain.RedoStateStarted,
        CreatedAt:  time.Now(),
    }
    if err := uc.redoLog.Write(entry); err != nil {
        return fmt.Errorf("write redo log: %w", err)
    }
    
    // 2. Load archived messages for VLM processing
    archiveMessages, err := uc.fsClient.ReadJSONL(ctx, p1.ArchiveURI+"messages.jsonl")
    if err != nil {
        return err
    }
    
    // 3. Generate Working Memory v2 via VLM
    prevWMContent := ""
    prevWM, _ := uc.fsClient.ReadRaw(ctx, p1.ArchiveURI+".overview.md")
    if len(prevWM) > 0 {
        prevWMContent = string(prevWM)
    }
    
    wmOps, err := uc.generateWMv2(ctx, prevWMContent, archiveMessages)
    if err != nil {
        slog.Warn("WM v2 generation failed, using empty WM", "error", err)
        wmOps = []domain.WMv2Operation{}  // Graceful: proceed without WM
    }
    
    currentSections := parseWMv2(prevWMContent)
    newSections := domain.ApplyWMOperations(currentSections, wmOps)
    newWMContent := domain.SerializeWMv2(newSections)
    uc.fsClient.WriteRaw(ctx, p1.ArchiveURI+".overview.md", []byte(newWMContent))
    // FS auto-emits ov.content.written → Search indexes the WM
    
    // 4. Extract 8 categories of memories via VLM (parallel)
    g, gCtx := errgroup.WithContext(ctx)
    sem := make(chan struct{}, 2)  // Max 2 VLM calls concurrently (rate limit)
    
    memoryCounts := make(map[string]int)
    var mu sync.Mutex
    
    for _, category := range domain.MemoryCategories {
        category := category
        g.Go(func() error {
            sem <- struct{}{}
            defer func() { <-sem }()
            
            memories, err := uc.extractCategoryMemories(gCtx, archiveMessages, category)
            if err != nil {
                slog.Warn("memory extraction failed", "category", category, "error", err)
                return nil  // Don't fail entire Phase 2
            }
            
            for i, memory := range memories {
                memURI := fmt.Sprintf("viking://user/%s/%s/memories/%s/%s_%d.md",
                    p1.AccountID, p1.UserID, category, p1.SessionID, i)
                uc.fsClient.WriteRaw(gCtx, memURI, []byte(memory.Content))
                // FS auto-emits ov.content.written → Search indexes each memory
            }
            
            mu.Lock()
            memoryCounts[category] = len(memories)
            mu.Unlock()
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return err
    }
    
    // 5. Update hotness for context URIs used in this session
    if len(p1.UsedURIs) > 0 {
        uc.searchClient.UpdateHotness(ctx, p1.UsedURIs)
    }
    
    // 6. Mark redo-log as committed
    entry.State = domain.RedoStateCommitted
    uc.redoLog.Write(entry)
    
    // 7. Publish events
    uc.publisher.PublishSessionCommitted(ctx, port.SessionCommittedPayload{
        SessionID: p1.SessionID,
        ArchiveURI: p1.ArchiveURI,
        UsedURIs:  p1.UsedURIs,
    })
    uc.publisher.PublishMemoryExtracted(ctx, port.MemoryExtractedPayload{
        SessionID:       p1.SessionID,
        MemoriesCount:   sumMemories(memoryCounts),
        Categories:      memoryCounts,
    })
    
    return nil
}
```

### 4.4 WM v2 VLM Prompt

```go
// internal/usecase/wm_v2.go

func (uc *Phase2ExtractUseCase) generateWMv2(ctx context.Context, prevWM string, messages []json.RawMessage) ([]domain.WMv2Operation, error) {
    // Format messages for VLM
    var msgSummary strings.Builder
    for _, raw := range messages {
        var m domain.Message
        json.Unmarshal(raw, &m)
        msgSummary.WriteString(fmt.Sprintf("[%s]: ", m.Role))
        for _, part := range m.Parts {
            if tp, ok := part.(*domain.TextPart); ok {
                text := tp.Text
                if len(text) > 200 { text = text[:200] + "..." }
                msgSummary.WriteString(text)
            }
        }
        msgSummary.WriteString("\n")
    }
    
    prompt := fmt.Sprintf(`You are updating a structured Working Memory document.

PREVIOUS WORKING MEMORY:
%s

NEW CONVERSATION MESSAGES:
%s

Output a JSON array of section operations. Each operation has:
- section_id: one of [title, current_state, task_goals, key_facts, files_context, errors, open_issues]
- op: KEEP | UPDATE | APPEND
- content: new content (empty string for KEEP)

Only include sections that need changes. Sections not mentioned keep their current content.

Respond with ONLY valid JSON, no markdown:
{"operations": [...]}`, prevWM, msgSummary.String())
    
    raw, err := uc.vlmClient.GenerateStructured(ctx, prompt, WMv2OperationsSchema)
    if err != nil {
        return nil, err
    }
    
    var result struct {
        Operations []domain.WMv2Operation `json:"operations"`
    }
    json.Unmarshal(raw, &result)
    return result.Operations, nil
}

// Memory extraction prompt (per category):
func buildMemoryExtractionPrompt(messages []json.RawMessage, category string) string {
    return fmt.Sprintf(`Extract %s-related memories from the conversation.

CATEGORY DEFINITION:
%s: %s

CONVERSATION:
%s

Output JSON array:
[{"content": "Extracted memory about %s", "importance": 0-1}]

Only include genuinely important %s information. Return empty array if nothing significant.`, 
        category, category, categoryDefinition(category), formatMessages(messages), category, category)
}
```

---

## 5. Redo Log — Crash Recovery

```go
// adapter/store/redo_log/file_store.go

// Redo log là một file JSONL trên disk:
// ~/.openviking/redo_log/sessions.jsonl
// Mỗi line: {"session_id":"..","archive_uri":"..","state":"started|committed","created_at":".."}

type FileRedoLogStore struct {
    path string
    mu   sync.Mutex
}

func (s *FileRedoLogStore) Write(entry *domain.RedoLogEntry) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()
    
    data, _ := json.Marshal(entry)
    _, err = f.WriteString(string(data) + "\n")
    return err
}

func (s *FileRedoLogStore) FindUncommitted() ([]*domain.RedoLogEntry, error) {
    // Read all entries, find latest state per session_id
    // Sessions with state="started" and no "committed" → uncommitted
    entries := readAllEntries(s.path)
    
    latestState := make(map[string]*domain.RedoLogEntry)
    for _, e := range entries {
        if prev, exists := latestState[e.SessionID+e.ArchiveURI]; !exists || e.CreatedAt.After(prev.CreatedAt) {
            latestState[e.SessionID+e.ArchiveURI] = e
        }
    }
    
    var uncommitted []*domain.RedoLogEntry
    for _, e := range latestState {
        if e.State == domain.RedoStateStarted {
            uncommitted = append(uncommitted, e)
        }
    }
    return uncommitted, nil
}

// Startup recovery (called in main.go before serving):
func (uc *ReplayRedoLogUseCase) Execute(ctx context.Context) {
    uncommitted, _ := uc.redoLog.FindUncommitted()
    for _, entry := range uncommitted {
        slog.Info("recovering uncommitted Phase 2", "session_id", entry.SessionID)
        // Re-execute Phase 2 — all operations are idempotent
        // (FS.WriteRaw overwrites same files; VectorDB.Upsert overwrites same URIs)
        meta := uc.loadMeta(ctx, entry.SessionID, entry.AccountID)
        p1Result := reconstructP1Result(entry, meta)
        go uc.phase2.Execute(ctx, p1Result, meta)
    }
}
```

---

## 6. AddMessages & Token Tracking

```go
// internal/usecase/add_messages.go

func (uc *AddMessagesUseCase) Execute(ctx context.Context, req dto.AddMessagesRequest) error {
    sessionURI := fmt.Sprintf("viking://session/%s/%s/", req.AccountID, req.SessionID)
    messagesURI := sessionURI + "messages.jsonl"
    
    // Calculate token size for new messages
    totalNewTokens := 0
    for _, msg := range req.Messages {
        for _, part := range msg.Parts {
            totalNewTokens += part.TokenCount()
        }
        msg.TokenSize = totalNewTokens
    }
    
    // Append to messages.jsonl
    lines := toJSONLines(req.Messages)
    uc.fsClient.AppendJSONL(ctx, messagesURI, lines)
    
    // Update pending_tokens in meta
    meta, _ := uc.loadMeta(ctx, req.SessionID, req.AccountID)
    meta.PendingTokens += totalNewTokens
    meta.MessageCount  += len(req.Messages)
    uc.fsClient.WriteJSON(ctx, sessionURI+".meta.json", meta)
    
    // Auto-commit check
    if meta.PendingTokens >= uc.config.TokenThreshold {
        go func() {
            uc.commitUC.Execute(context.Background(), dto.CommitRequest{
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

## 7. gRPC Service Definition

```protobuf
syntax = "proto3";
package openviking.session.v1;

service SessionService {
  rpc CreateSession(CreateSessionRequest) returns (CreateSessionResponse);
  rpc GetSession(GetSessionRequest) returns (GetSessionResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc AddMessages(AddMessagesRequest) returns (AddMessagesResponse);
  rpc RecordUsed(RecordUsedRequest) returns (RecordUsedResponse);
  rpc Commit(CommitRequest) returns (CommitResponse);
  rpc DeleteSession(DeleteSessionRequest) returns (DeleteSessionResponse);
  rpc GetSessionInfo(GetSessionInfoRequest) returns (GetSessionInfoResponse);
  rpc GetSessionContext(GetSessionContextRequest) returns (GetSessionContextResponse); // For search service
}

message GetSessionContextRequest {
  string session_id = 1;
  string account_id = 2;
}

message GetSessionContextResponse {
  string working_memory = 1;   // Latest WM v2 content
  repeated string used_uris = 2; // URIs accessed in this session
}

message CommitResponse {
  int32  archived_count = 1;
  int32  retained_count = 2;
  string archive_uri    = 3;
  bool   phase2_started = 4;
  bool   skipped        = 5;    // true if below token threshold
  string skip_reason    = 6;
}
```

---

## 8. NATS Events

### Published
```
Subject: "ov.session.committed"
Payload: {"session_id": "...", "archive_uri": "...", "used_uris": ["viking://..."]}
Subscribers: openviking-search (update hotness)

Subject: "ov.session.memory.extracted"
Payload: {"session_id": "...", "memories_count": 12, "categories": {"profile":2, "skills":5}}
Subscribers: (logging/analytics only)
```

### Consumed
```
Subject: "admin.account.deleted"
Action: Delete all sessions for account
  → FS.Rm(viking://session/{account_id}/, recursive=true)
```

---

## 9. Configuration

```yaml
session:
  grpc:
    port: 9013
  health:
    port: 9093
    
  commit:
    token_threshold: 2000         # Auto-commit when pending_tokens > this
    keep_recent_count: 10         # Keep last N messages after archive
    phase2_max_concurrent: 3      # Max concurrent background Phase 2 tasks
    
  redo_log:
    dir: "~/.openviking/redo_log"
    
  vlm:
    service_url: "bifrost:4000"
    wm_model: "gpt-4o-mini"
    extract_model: "gpt-4o-mini"
    max_concurrent_vlm: 2         # Bulkhead for VLM calls
    
  clients:
    fs: "openviking-fs:9011"
    search: "openviking-search:9012"
    
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
```

---

## 10. Testing Strategy

### Unit Tests
- `TestPhase1Archive_SplitsMessages` — 100 msgs, keep=10 → archived=90, retained=10
- `TestPhase1Archive_PathLockPreventsRace` — 2 goroutines commit → sequential (not concurrent)
- `TestPhase1Archive_NothingToArchive` → skip commit
- `TestApplyWMOperations_Update` — UPDATE op → replaces section content
- `TestApplyWMOperations_Append` — APPEND op → adds to existing content
- `TestApplyWMOperations_Keep` — KEEP op → section unchanged
- `TestSerializeWMv2_SevenSections` — output format matches expected Markdown
- `TestPhase2_MemoryExtraction_GracefulOnVLMError` — VLM fails → session not failed
- `TestRedoLog_FindUncommitted` — started entry without committed → returns as uncommitted
- `TestRedoLog_Idempotency` — replay Phase 2 twice → same result (no duplicate memories)
- `TestAddMessages_AutoCommitTriggered` — tokens > threshold → commit goroutine started

### Integration Tests
- `TestCommitE2E_Phase1Fast` — < 500ms
- `TestCommitE2E_Phase2WritesMemories` — WM v2 created, memory files written to FS
- `TestCrashRecovery_Phase2Resumed` — kill after redo write → restart → Phase 2 completes
- `TestSessionAwareSearch_Integration` — session with used URIs → search boosts them

---

## 11. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| VLM timeout trong Phase 2 (> 10 phút) | Trung bình | Phase 2 timeout=10min; graceful skip WM/extraction trên timeout |
| Redo log grows unboundedly | Thấp | Compact redo log daily: remove all "committed" entries older than 7 days |
| Multiple concurrent Phase 2 goroutines OOM | Thấp | `phase2_max_concurrent=3` semaphore |
| FS session directory deleted during Phase 2 | Thấp | Check directory exists at Phase 2 start; no-op if deleted |
| VLM output không đúng JSON schema | Trung bình | Retry với simplified prompt; fallback to empty WM operations |
| Memory extraction duplicates across commits | Thấp | Acceptable; Search's dedup by URI handles duplicates |
