# TASK-OV-013 — `services/openviking-session` Phase 2: WM v2, Memory Extraction & Redo Log

**Wave:** 5 (Context)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-012 (Phase 1 + domain)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-OV-004 §4.3, §4.4, §4.5, §5](../solutions/SOL-OV-004-Session-Service.md)

**Trạng thái:** ⏳ Pending  
**Ghi chú:** ov-session phase 2 (two-phase commit) not started  
---

## Mục tiêu

Implement Phase 2 của session commit: Working Memory v2 generation (VLM với 7-section Markdown), memory extraction (8 categories), crash-safe redo log, gRPC server, NATS events và main.go.

---

## Các file cần tạo

### 1. `internal/usecase/phase2_extract.go` — Background Extraction

```go
type Phase2ExtractUseCase struct {
    fsClient    port.FSClient
    vlmClient   port.VLMClient
    redoLog     port.RedoLogStore
    publisher   port.EventPublisher
    searchClient port.SearchClient
    config      *Config
}

func (uc *Phase2ExtractUseCase) Execute(ctx context.Context, p1 *Phase1Result, meta *domain.SessionMeta) error {
    // ── 1. Write redo-log BEFORE any work (crash safety) ──────
    entry := &domain.RedoLogEntry{
        SessionID:  p1.SessionID,
        AccountID:  p1.AccountID,
        UserID:     p1.UserID,
        ArchiveURI: p1.ArchiveURI,
        State:      domain.RedoStateStarted,
        CreatedAt:  time.Now(),
    }
    if err := uc.redoLog.Write(entry); err != nil {
        return fmt.Errorf("write redo log: %w", err)
    }

    // ── 2. Load archived messages ──────────────────────────────
    rawLines, err := uc.fsClient.ReadJSONL(ctx, p1.ArchiveURI+"messages.jsonl")
    if err != nil { return err }
    messages := parseMessages(rawLines)

    // ── 3. Load previous WM (for update context) ──────────────
    prevWMContent := ""
    if wm, err := uc.fsClient.Read(ctx, p1.ArchiveURI+".overview.md", 2); err == nil {
        prevWMContent = string(wm)
    } else {
        // Try current session overview
        sessionWMURI := "viking://session/" + p1.AccountID + "/" + p1.SessionID + "/.overview.md"
        if wm, err := uc.fsClient.Read(ctx, sessionWMURI, 2); err == nil {
            prevWMContent = string(wm)
        }
    }

    // ── 4. Generate WM v2 via VLM ─────────────────────────────
    wmOps, err := uc.generateWMv2(ctx, prevWMContent, messages)
    if err != nil {
        slog.Warn("WM v2 generation failed, using empty ops", "error", err)
        wmOps = nil  // Graceful degradation
    }
    currentSections := domain.ParseWMv2(prevWMContent)
    newSections := domain.ApplyWMOperations(currentSections, wmOps)
    newWMContent := domain.SerializeWMv2(newSections)

    // Write WM to archive (FS auto-emits ov.content.written → Search indexes WM)
    uc.fsClient.Write(ctx, p1.ArchiveURI+".overview.md", []byte(newWMContent), p1.AccountID)

    // ── 5. Extract 8 memory categories (parallel, max 2 VLM concurrent) ──
    memoryCounts := make(map[string]int)
    var mu sync.Mutex
    g, gCtx := errgroup.WithContext(ctx)
    sem := make(chan struct{}, uc.config.MaxConcurrentVLM)

    for _, category := range domain.MemoryCategories {
        category := category
        g.Go(func() error {
            sem <- struct{}{}
            defer func() { <-sem }()

            memories, err := uc.extractCategoryMemories(gCtx, messages, category)
            if err != nil {
                slog.Warn("memory extraction failed", "category", category, "error", err)
                return nil  // Don't fail entire Phase 2
            }

            for i, memory := range memories {
                memURI := fmt.Sprintf("viking://user/%s/%s/memories/%s/%s_%03d.md",
                    p1.AccountID, p1.UserID, category, p1.SessionID[:8], i)
                // Write → FS auto-emits ov.content.written → Search indexes each memory
                uc.fsClient.Write(gCtx, memURI, []byte(memory), p1.AccountID)
            }

            mu.Lock()
            memoryCounts[category] = len(memories)
            mu.Unlock()
            return nil
        })
    }
    g.Wait()

    // ── 6. Update search hotness for used URIs ────────────────
    if len(p1.UsedURIs) > 0 {
        uc.searchClient.UpdateHotness(ctx, p1.UsedURIs)
    }

    // ── 7. Mark redo log as committed ─────────────────────────
    entry.State = domain.RedoStateCommitted
    uc.redoLog.Write(entry)

    // ── 8. Publish NATS events ────────────────────────────────
    uc.publisher.PublishSessionCommitted(ctx, port.SessionCommittedPayload{
        SessionID:  p1.SessionID,
        ArchiveURI: p1.ArchiveURI,
        UsedURIs:   p1.UsedURIs,
    })
    uc.publisher.PublishMemoryExtracted(ctx, port.MemoryExtractedPayload{
        SessionID:     p1.SessionID,
        MemoriesCount: sumMemories(memoryCounts),
        Categories:    memoryCounts,
    })

    return nil
}
```

### 2. WM v2 VLM Generation

```go
func (uc *Phase2ExtractUseCase) generateWMv2(ctx context.Context, prevWM string, messages []domain.Message) ([]domain.WMv2Operation, error) {
    // Format messages summary (max 200 chars per message to avoid huge prompt)
    msgSummary := formatMessagesSummary(messages, 200)

    prompt := fmt.Sprintf(`You are updating a structured Working Memory document for an AI agent session.

PREVIOUS WORKING MEMORY:
%s

NEW CONVERSATION MESSAGES:
%s

Output a JSON object with section operations. Each operation specifies:
- section_id: one of [title, current_state, task_goals, key_facts, files_context, errors, open_issues]  
- op: KEEP (no change) | UPDATE (replace full content) | APPEND (add to existing)
- content: new content (empty string for KEEP)

Only include sections that need changes.

Respond with ONLY valid JSON, no markdown fences:
{"operations": [{"section_id": "current_state", "op": "UPDATE", "content": "..."}, ...]}`,
        prevWM, msgSummary)

    raw, err := uc.vlmClient.GenerateStructured(ctx, prompt, nil,
        vlm.WithVLMMaxTokens(1000),
        vlm.WithVLMTemperature(0.3),
    )
    if err != nil { return nil, err }

    var result struct {
        Operations []domain.WMv2Operation `json:"operations"`
    }
    if err := json.Unmarshal(raw, &result); err != nil { return nil, err }
    return result.Operations, nil
}

func (uc *Phase2ExtractUseCase) extractCategoryMemories(ctx context.Context, messages []domain.Message, category string) ([]string, error) {
    def := domain.MemoryCategoryDefinitions[category]
    msgSummary := formatMessagesSummary(messages, 300)

    prompt := fmt.Sprintf(`Extract %s-related information from this conversation.

CATEGORY: %s
DEFINITION: %s

CONVERSATION:
%s

Output JSON array of extracted facts. Only include genuinely important information.
Return empty array if nothing significant for this category.

{"memories": ["Fact 1 about %s", "Fact 2 about %s"]}`,
        category, category, def, msgSummary, category, category)

    raw, err := uc.vlmClient.GenerateStructured(ctx, prompt, nil,
        vlm.WithVLMMaxTokens(500),
    )
    if err != nil { return nil, err }

    var result struct{ Memories []string `json:"memories"` }
    json.Unmarshal(raw, &result)
    return result.Memories, nil
}
```

### 3. Redo Log

**File: `internal/domain/redo_log.go`**

```go
type RedoState string
const (
    RedoStateStarted   RedoState = "started"
    RedoStateCommitted RedoState = "committed"
    RedoStateFailed    RedoState = "failed"
)

type RedoLogEntry struct {
    SessionID  string    `json:"session_id"`
    AccountID  string    `json:"account_id"`
    UserID     string    `json:"user_id"`
    ArchiveURI string    `json:"archive_uri"`
    State      RedoState `json:"state"`
    CreatedAt  time.Time `json:"created_at"`
}
```

**File: `internal/adapter/store/redo_log/file_store.go`**

```go
// File: {redoLogDir}/sessions.jsonl
// Each line: JSON-encoded RedoLogEntry

type FileRedoLogStore struct {
    path string    // absolute path to sessions.jsonl
    mu   sync.Mutex
}

func NewFileRedoLogStore(dir string) (*FileRedoLogStore, error)
// Create dir if not exists; path = dir/sessions.jsonl

func (s *FileRedoLogStore) Write(entry *domain.RedoLogEntry) error
// O_APPEND|O_CREATE|O_WRONLY

func (s *FileRedoLogStore) FindUncommitted() ([]*domain.RedoLogEntry, error)
// Read all entries
// Group by sessionID+archiveURI → latest state
// Return entries with state="started" and no "committed" counterpart
// (Compare by CreatedAt timestamp)

func (s *FileRedoLogStore) Compact() error
// Remove all "committed" entries older than 7 days
// Keep "started" entries (might need recovery)
// Rewrite file with filtered entries
```

**File: `internal/usecase/redo_log.go`** — Startup Recovery

```go
type ReplayRedoLogUseCase struct {
    redoLog  port.RedoLogStore
    phase2   *Phase2ExtractUseCase
    fsClient port.FSClient
}

func (uc *ReplayRedoLogUseCase) Execute(ctx context.Context) {
    uncommitted, err := uc.redoLog.FindUncommitted()
    if err != nil { slog.Error("redo log scan failed", "error", err); return }

    for _, entry := range uncommitted {
        slog.Info("recovering uncommitted Phase 2", "session_id", entry.SessionID, "archive_uri", entry.ArchiveURI)
        
        // Load meta from FS
        sessionURI := "viking://session/" + entry.AccountID + "/" + entry.SessionID + "/"
        metaBytes, _ := uc.fsClient.Read(ctx, sessionURI+".meta.json", 2)
        var meta domain.SessionMeta
        json.Unmarshal(metaBytes, &meta)

        // Reconstruct Phase1Result from redo log entry
        p1 := &Phase1Result{
            SessionID:  entry.SessionID,
            AccountID:  entry.AccountID,
            UserID:     entry.UserID,
            ArchiveURI: entry.ArchiveURI,
        }

        // Run Phase 2 in background — idempotent (overwrites same files)
        go func(e *domain.RedoLogEntry, p *Phase1Result, m *domain.SessionMeta) {
            bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
            defer cancel()
            uc.phase2.Execute(bgCtx, p, m)
        }(entry, p1, &meta)
    }
}
```

---

## 4. gRPC Handler

**File: `internal/adapter/grpc/handler.go`**

```go
type Handler struct {
    sessionv1.UnimplementedSessionServiceServer
    createSessionUC *usecase.CreateSessionUseCase
    getSessionUC    *usecase.GetSessionUseCase
    listSessionsUC  *usecase.ListSessionsUseCase
    addMessagesUC   *usecase.AddMessagesUseCase
    recordUsedUC    *usecase.RecordUsedUseCase
    commitUC        *usecase.CommitUseCase
    deleteSessionUC *usecase.DeleteSessionUseCase
}

// GetSessionContext — used by Search service to get WM + used_uris
func (h *Handler) GetSessionContext(ctx context.Context, req *sessionv1.GetSessionContextRequest) (*sessionv1.GetSessionContextResponse, error) {
    meta, _ := h.getSessionUC.GetMeta(ctx, req.SessionId, req.AccountId)
    
    // Load latest WM from session dir
    sessionURI := "viking://session/" + req.AccountId + "/" + req.SessionId + "/"
    wmBytes, _ := h.fsClient.Read(ctx, sessionURI+".overview.md", 2)
    
    return &sessionv1.GetSessionContextResponse{
        WorkingMemory: string(wmBytes),
        UsedUris:      meta.UsedURIs,
    }, nil
}
```

---

## 5. NATS Events

**Published:**
```go
// Subject: "ov.session.committed"
// Payload: {session_id, archive_uri, used_uris}
// Subscribers: Search service → update hotness

// Subject: "ov.session.memory.extracted"
// Payload: {session_id, memories_count, categories}
// Subscribers: Logging/analytics
```

**Subscribed:**
```go
// Subject: "admin.account.deleted"
// Action: Delete all sessions for account
// → fsClient.Rm("viking://session/{accountID}/", recursive=true)
```

---

## 6. Config

```yaml
session:
  grpc:
    port: 9013
  health:
    port: 9093
  commit:
    token_threshold: 2000
    keep_recent_count: 10
    max_concurrent_phase2: 3
  redo_log:
    dir: "~/.openviking/redo_log"
  vlm:
    service_url: "bifrost:4000"
    wm_model: "gpt-4o-mini"
    extract_model: "gpt-4o-mini"
    max_concurrent_vlm: 2
  clients:
    fs: "openviking-fs:9011"
    search: "openviking-search:9012"
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
```

---

## Unit Tests

```
TestGenerateWMv2_ValidOps            → VLM mock returns valid JSON → ops parsed
TestGenerateWMv2_VLMFails_GracefulDeg → VLM error → nil ops, no panic
TestGenerateWMv2_InvalidJSON_Empty   → VLM returns garbage → empty ops
TestExtractMemories_ReturnsStrings   → VLM returns facts → memory files written
TestExtractMemories_EmptyResult      → VLM returns [] → 0 files written
TestPhase2_AllCategoriesProcessed    → 8 categories → 8 goroutines → all processed
TestPhase2_VLMFail_ContinuesOtherCat → 1 category fails → other 7 succeed
TestPhase2_WritesWMToArchive         → WM file written to archiveURI/.overview.md
TestPhase2_UpdatesHotness            → usedURIs → searchClient.UpdateHotness called
TestPhase2_PublishesEvents           → both session.committed + memory.extracted published
TestPhase2_MarksRedoLogCommitted     → redo log state = committed after success
TestRedoLog_Write                    → write entry → file has 1 line
TestRedoLog_FindUncommitted_Started  → started without committed → returned
TestRedoLog_FindUncommitted_Committed → both started+committed → empty result
TestRedoLog_FindUncommitted_Multiple → 3 sessions, 1 uncommitted → only 1 returned
TestRedoLog_Compact                  → old committed entries → removed; started kept
TestReplayRedoLog_RecoveryRuns       → uncommitted → phase2 goroutine started
TestPhase2_Idempotent                → run twice → same WM (overwrite idempotent)
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
buf generate services/openviking-session/
go build ./services/openviking-session/...
go test ./services/openviking-session/... -v -count=1 -race
```

---

## Ghi chú triển khai

- Phase 2 max timeout: `10*time.Minute` — VLM calls có thể chậm cho large sessions
- Redo log compact: chạy khi service start, và schedule daily via `time.Ticker(24*time.Hour)`
- VLM prompt: sử dụng model `gpt-4o-mini` để balance cost vs quality
- `formatMessagesSummary(messages, maxPerMsg int) string`: truncate mỗi message content tới `maxPerMsg` chars trước khi concat
- Memory file content: plain text fact, không có metadata header
