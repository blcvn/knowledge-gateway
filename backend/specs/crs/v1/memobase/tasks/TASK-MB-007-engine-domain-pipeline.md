# TASK-MB-007 — `services/memobase-engine` Domain, DB & 3-Call LLM Pipeline

**Wave:** 2 (LLM Processing)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-MB-005 (pkg/adapters), TASK-MB-006 (pkg/prompt), TASK-MB-003/004 (ingestion blobs)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-MB-002 §2, §3, §4, §5, §6, §7](../solutions/SOL-MB-002-Memory-Engine-Profile-YOLO.md)  
**Port gRPC:** 9042

**Trạng thái:** ✅ Implemented  
**Ghi chú:** memobase-engine: 30 .go - full pipeline domain  
---

## Mục tiêu

Tạo `services/memobase-engine/` phần cốt lõi: database schema, domain models, và `ProcessBlobsUseCase` orchestrating 3-call LLM pipeline với parallel goroutines (errgroup) cho profile + event processing.

---

## Cấu trúc thư mục

```
services/memobase-engine/
├── cmd/server/main.go
├── api/proto/memobase/engine/v1/engine.proto
├── internal/
│   ├── domain/
│   │   ├── profile.go       # Profile, ProfileSlot, MergeAction, MergeResult
│   │   ├── event.go         # EventSummary, EventGist, EventTag
│   │   ├── value_object.go  # Topic, SubTopic constants
│   │   └── errors.go        # ErrLLMParseFailed, ErrBlobTruncated
│   ├── usecase/
│   │   ├── process_blobs.go        # Orchestrator — 3-call pipeline
│   │   ├── extract_profile.go      # LLM Call #1 (entry_summary) + #2 (extract_topics)
│   │   ├── merge_profile.go        # LLM Call #3 (YOLO merge)
│   │   ├── organize_profile.go     # Post-merge constraints (no LLM)
│   │   ├── validate_profile.go     # Strict mode filter (optional LLM)
│   │   ├── re_summary.go           # Slot content truncation (optional LLM)
│   │   ├── summarize_event.go      # Event goroutine LLM
│   │   ├── tag_event.go            # Event tagging (optional LLM)
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go   # ProfileRepository, EventRepository, IngestionClient, AdminClient, EventPublisher
│   └── adapter/
│       └── repository/postgres/
│           ├── profile_repo.go
│           └── event_repo.go
└── internal/infra/
    └── migrations/
        └── 003_engine.up.sql
```

---

## 1. Database Migration

**File: `internal/infra/migrations/003_engine.up.sql`**

```sql
-- Runs AFTER 001_foundation.up.sql

-- User profiles (extracted by YOLO merge)
CREATE TABLE IF NOT EXISTS user_profiles (
    id         UUID        NOT NULL,
    project_id VARCHAR     NOT NULL,
    user_id    UUID        NOT NULL,
    content    TEXT        NOT NULL,
    attributes JSONB       NOT NULL,  -- {topic, sub_topic, updated_at}
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_profiles_user    ON user_profiles(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_topic   ON user_profiles USING gin(attributes);
CREATE INDEX IF NOT EXISTS idx_user_profiles_updated ON user_profiles(user_id, project_id, updated_at DESC);

-- User events (timeline)
CREATE TABLE IF NOT EXISTS user_events (
    id         UUID        NOT NULL,
    project_id VARCHAR     NOT NULL,
    user_id    UUID        NOT NULL,
    event_data JSONB       NOT NULL,  -- {event_tip, event_tags[], profile_delta}
    embedding  vector(1536),          -- pgvector (NULL if embedding disabled)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_events_user ON user_events(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_user_events_time ON user_events(user_id, project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_events_embedding ON user_events USING ivfflat (embedding vector_cosine_ops)
    WHERE embedding IS NOT NULL;

-- User event gists (fine-grained descriptions for semantic search)
CREATE TABLE IF NOT EXISTS user_event_gists (
    id         UUID        NOT NULL,
    project_id VARCHAR     NOT NULL,
    event_id   UUID        NOT NULL,
    user_id    UUID        NOT NULL,
    gist_data  JSONB       NOT NULL,  -- {gist_content}
    embedding  vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (event_id, project_id) REFERENCES user_events(id, project_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_event_gists_user  ON user_event_gists(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_user_event_gists_event ON user_event_gists(event_id, project_id);
CREATE INDEX IF NOT EXISTS idx_user_event_gists_emb   ON user_event_gists USING ivfflat (embedding vector_cosine_ops)
    WHERE embedding IS NOT NULL;

-- Daily usage tracking (upsert by engine after processing)
-- NOTE: usage_records table is in 001_foundation.up.sql (admin)
```

---

## 2. Domain Models

**File: `internal/domain/profile.go`**

```go
type ProfileSlot struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    ProjectID string
    Topic     string
    SubTopic  string
    Content   string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type MergeResult struct {
    Added   []ProfileSlot
    Updated []ProfileSlot
    Deleted []uuid.UUID
}

// Mirrors prompt.MergeAction for LLM output
type MergeAction struct {
    Add    []struct{ Topic, SubTopic, Content string } `json:"add"`
    Update []struct{ Index int; Topic, SubTopic, Content string } `json:"update"`
    Delete []int `json:"delete"`
}
```

**File: `internal/domain/event.go`**

```go
type EventTag struct {
    Tag string `json:"tag"`
}

type EventSummary struct {
    EventTip    string      // Multi-line, "-" prefixed gists
    EventGists  []string    // Parsed from EventTip: each "- " prefixed line
    EventTags   []EventTag
    ProfileDelta []any      // Profile changes implied by this event
}

type UserEvent struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    ProjectID   string
    EventData   EventSummary
    Embedding   []float32   // nil if embedding disabled
    CreatedAt   time.Time
}

type UserEventGist struct {
    ID          uuid.UUID
    EventID     uuid.UUID
    UserID      uuid.UUID
    ProjectID   string
    GistContent string
    Embedding   []float32
    CreatedAt   time.Time
}

// parseGists: split EventTip by "\n", keep lines starting with "- "
func parseGists(eventTip string) []string {
    var gists []string
    for _, line := range strings.Split(eventTip, "\n") {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "- ") {
            gists = append(gists, strings.TrimPrefix(line, "- "))
        }
    }
    return gists
}
```

---

## 3. ProcessBlobsUseCase — Orchestrator

**File: `internal/usecase/process_blobs.go`**

```go
type ProcessBlobsRequest struct {
    UserID    uuid.UUID
    ProjectID string
    BufferIDs []uuid.UUID
    BlobType  domain.BlobType
}

type ProcessBlobsUseCase struct {
    ingestionClient port.IngestionClient   // gRPC to memobase-ingestion
    adminClient     port.AdminClient       // gRPC to memobase-admin (get profile config)
    profileRepo     port.ProfileRepository
    extractProfile  *ExtractProfileUseCase
    mergeProfile    *MergeProfileUseCase
    organizeProfile *OrganizeProfileUseCase
    validateProfile *ValidateProfileUseCase
    summarizeEvent  *SummarizeEventUseCase
    tagEvent        *TagEventUseCase
    embedder        embedder.EmbedderClient
    eventRepo       port.EventRepository
    publisher       port.EventPublisher
    tokenizer       tokenizer.Tokenizer
    config          EngineConfig
}

type EngineConfig struct {
    MaxProcessTokenSize int     // default: 16384
    MaxSubtopics        int     // default: 15
    MaxSlotTokenSize    int     // default: 128
    StrictMode          bool
    ValidateMode        bool
    Language            string  // "en" | "zh"
    EnableEventEmbedding bool
}

func (uc *ProcessBlobsUseCase) Execute(ctx context.Context, req ProcessBlobsRequest) error {
    // ── STEP 1: Fetch blobs from ingestion service ─────────────────
    blobs, err := uc.ingestionClient.GetBlobsForProcessing(ctx, req.BufferIDs, req.ProjectID)
    if err != nil { return fmt.Errorf("fetch blobs: %w", err) }

    // ── STEP 2: Get profile config from admin service ──────────────
    profileConfig, err := uc.adminClient.GetProfileConfig(ctx, req.ProjectID)
    if err != nil { return fmt.Errorf("get profile config: %w", err) }

    // ── STEP 3: Fetch existing user profiles ───────────────────────
    existingSlots, err := uc.profileRepo.GetByUser(ctx, req.UserID, req.ProjectID)
    if err != nil { return fmt.Errorf("fetch profiles: %w", err) }

    // ── STEP 4: Prepare conversation text (truncate to 16384 tokens) ─
    rawConversation := serializeBlobs(blobs)
    rawConversation = uc.tokenizer.TruncateToTokens(rawConversation, uc.config.MaxProcessTokenSize)

    // ── STEP 5: LLM Call #1 — entry_summary → memoStr ─────────────
    prompts := uc.prompts.GetOrDefault(profileConfig.Language)
    memoStr, err := uc.llm.Complete(ctx, prompts.EntrySummary(rawConversation),
        llm.WithMaxTokens(1024))
    if err != nil { return fmt.Errorf("entry summary: %w", err) }

    // ── STEP 6: Parallel goroutines ────────────────────────────────
    g, gCtx := errgroup.WithContext(ctx)

    var mergeResult *domain.MergeResult
    var eventSummary *domain.EventSummary

    // Goroutine 1: Profile pipeline
    g.Go(func() error {
        // LLM Call #2: extract profile facts
        facts, err := uc.extractProfile.Execute(gCtx, memoStr, existingSlots, profileConfig)
        if err != nil { return fmt.Errorf("extract profile: %w", err) }

        // LLM Call #3: YOLO merge
        mr, err := uc.mergeProfile.Execute(gCtx, existingSlots, facts)
        if err != nil { return fmt.Errorf("merge profile: %w", err) }

        // Post-merge organize (no LLM)
        mr = uc.organizeProfile.Execute(mr, profileConfig)

        // Optional: validate strict mode (LLM if needed)
        if profileConfig.StrictMode || profileConfig.ValidateMode {
            mr, err = uc.validateProfile.Execute(gCtx, mr, profileConfig)
            if err != nil { return err }
        }

        mergeResult = mr
        return nil
    })

    // Goroutine 2: Event pipeline
    g.Go(func() error {
        es, err := uc.summarizeEvent.Execute(gCtx, memoStr)
        if err != nil {
            slog.Warn("event summarization failed, continuing", "error", err)
            return nil  // Event pipeline failure should not fail profile pipeline
        }

        if len(profileConfig.EventTags) > 0 {
            tags, _ := uc.tagEvent.Execute(gCtx, es.EventTip, profileConfig.EventTags)
            es.EventTags = append(es.EventTags, tags...)
        }

        eventSummary = es
        return nil
    })

    if err := g.Wait(); err != nil { return err }

    // ── STEP 7: Persist results ────────────────────────────────────
    if mergeResult != nil {
        if err := uc.profileRepo.ApplyMergeResult(ctx, req.UserID, req.ProjectID, mergeResult); err != nil {
            return fmt.Errorf("apply merge result: %w", err)
        }
    }

    if eventSummary != nil {
        event, err := uc.persistEvent(ctx, req.UserID, req.ProjectID, eventSummary)
        if err != nil {
            slog.Warn("persist event failed", "error", err)
            // Don't fail — event is secondary
        } else {
            // Publish event.created for memobase-event to embed
            uc.publisher.PublishEventCreated(ctx, event)
        }
    }

    // ── STEP 8: Notify ingestion (done) + publish profile.changed ──
    uc.publisher.PublishEngineCompleted(ctx, port.EngineCompletedPayload{
        UserID:    req.UserID.String(),
        ProjectID: req.ProjectID,
        BufferIDs: uuidsToStrings(req.BufferIDs),
    })

    if mergeResult != nil && mergeResult.HasChanges() {
        uc.publisher.PublishProfileChanged(ctx, port.ProfileChangedPayload{
            UserID:    req.UserID.String(),
            ProjectID: req.ProjectID,
        })
    }

    return nil
}

func (uc *ProcessBlobsUseCase) persistEvent(ctx context.Context, userID uuid.UUID, projectID string, es *domain.EventSummary) (*domain.UserEvent, error) {
    event := &domain.UserEvent{
        ID: uuid.New(), UserID: userID, ProjectID: projectID,
        EventData: *es, CreatedAt: time.Now(),
    }

    // Embed event if enabled (non-blocking fail)
    if uc.config.EnableEventEmbedding && uc.embedder.IsEnabled() {
        emb, err := uc.embedder.EmbedQuery(ctx, es.EventTip)
        if err == nil { event.Embedding = emb }
    }

    return uc.eventRepo.Save(ctx, event)
}
```

---

## 4. ExtractProfileUseCase

**File: `internal/usecase/extract_profile.go`**

```go
type ExtractProfileUseCase struct {
    llm     llm.LLMClient
    prompts *prompt.Registry
}

type ExtractedFacts struct {
    Facts []struct {
        Topic    string `json:"topic"`
        SubTopic string `json:"sub_topic"`
        Content  string `json:"content"`
    } `json:"facts"`
}

func (uc *ExtractProfileUseCase) Execute(
    ctx context.Context,
    memoStr string,
    existingSlots []domain.ProfileSlot,
    config ProfileConfig,
) (string, error) {
    provider := uc.prompts.GetOrDefault(config.Language)

    existingTopics := extractUniqueTopics(existingSlots)
    promptStr := provider.ExtractProfile(memoStr, existingTopics)

    raw, err := uc.llm.CompleteJSON(ctx, promptStr, llm.WithJSONMode(), llm.WithMaxTokens(512))
    if err != nil { return "", fmt.Errorf("extract profile LLM: %w", err) }

    // Validate JSON parses correctly
    var result ExtractedFacts
    if err := json.Unmarshal(raw, &result); err != nil {
        return "", fmt.Errorf("parse extracted facts: %w: %s", domain.ErrLLMParseFailed, string(raw))
    }

    // Return as formatted string for merge prompt
    return formatFacts(result.Facts), nil
}
```

---

## 5. SummarizeEventUseCase

**File: `internal/usecase/summarize_event.go`**

```go
type SummarizeEventUseCase struct {
    llm     llm.LLMClient
    prompts *prompt.Registry
    lang    string
}

func (uc *SummarizeEventUseCase) Execute(ctx context.Context, memoStr string) (*domain.EventSummary, error) {
    provider := uc.prompts.GetOrDefault(uc.lang)
    raw, err := uc.llm.CompleteJSON(ctx, provider.SummarizeEvent(memoStr),
        llm.WithJSONMode(), llm.WithMaxTokens(512))
    if err != nil { return nil, err }

    var result struct {
        EventTip     string     `json:"event_tip"`
        EventTags    []struct{ Tag string `json:"tag"` } `json:"event_tags"`
        ProfileDelta []any      `json:"profile_delta"`
    }
    if err := json.Unmarshal(raw, &result); err != nil {
        return nil, fmt.Errorf("%w: %v", domain.ErrLLMParseFailed, err)
    }

    // Parse gists from EventTip (lines starting with "- ")
    gists := domain.ParseGists(result.EventTip)

    tags := make([]domain.EventTag, len(result.EventTags))
    for i, t := range result.EventTags { tags[i] = domain.EventTag{Tag: t.Tag} }

    return &domain.EventSummary{
        EventTip:    result.EventTip,
        EventGists:  gists,
        EventTags:   tags,
        ProfileDelta: result.ProfileDelta,
    }, nil
}
```

---

## 6. Repository Interfaces

**File: `internal/usecase/port/output.go`**

```go
type ProfileRepository interface {
    GetByUser(ctx context.Context, userID uuid.UUID, projectID string) ([]domain.ProfileSlot, error)
    ApplyMergeResult(ctx context.Context, userID uuid.UUID, projectID string, result *domain.MergeResult) error
    // Implements: INSERT for Add, UPDATE for Update, DELETE for Delete (indexed)
}

type EventRepository interface {
    Save(ctx context.Context, event *domain.UserEvent) (*domain.UserEvent, error)
    SaveGists(ctx context.Context, gists []domain.UserEventGist) error
}

type IngestionClient interface {
    GetBlobsForProcessing(ctx context.Context, bufferIDs []uuid.UUID, projectID string) ([]*ingestiondomain.Blob, error)
    MarkBufferDone(ctx context.Context, bufferIDs []uuid.UUID, projectID string) error
    MarkBufferFailed(ctx context.Context, bufferIDs []uuid.UUID, projectID string, errMsg string) error
}

type AdminClient interface {
    GetProfileConfig(ctx context.Context, projectID string) (*ProjectProfileConfig, error)
}
```

---

## Unit Tests

```
TestProcessBlobsUseCase_ThreeLLMCalls         → mock LLM → called exactly 3 times in order
TestProcessBlobsUseCase_ParallelGoroutines    → profile + event run in parallel (use timing)
TestProcessBlobsUseCase_EventFailureNoAbort   → event LLM fails → profile still processed
TestProcessBlobsUseCase_TruncatesBlobs        → >16384 tokens → truncated before LLM call
TestProcessBlobsUseCase_PublishesEvents       → engine.completed + profile.changed published
TestExtractProfileUseCase_ParsesJSON          → LLM returns {"facts":[...]} → parsed correctly
TestExtractProfileUseCase_LLMFails            → LLM error → error returned
TestExtractProfileUseCase_InvalidJSON         → LLM returns garbage → ErrLLMParseFailed
TestSummarizeEvent_ParsesGists                → "- Point 1\n- Point 2" → 2 gists
TestSummarizeEvent_EmptyGists                 → no "- " lines → empty gists
TestSummarizeEvent_LLMFails                   → error returned (not nil)
TestParseGists_WithDashPrefix                 → "- item" → ["item"]
TestParseGists_IgnoresNonDash                 → "item without dash" → []
TestParseGists_MixedLines                     → dash + non-dash → only dash items
TestProfileRepo_GetByUser                     → mock DB → returns ProfileSlots
TestProfileRepo_ApplyMergeResult_Add          → Add action → INSERT called
TestProfileRepo_ApplyMergeResult_Update       → Update action → UPDATE called with index
TestProfileRepo_ApplyMergeResult_Delete       → Delete action → DELETE called
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

buf generate services/memobase-engine/
go build ./services/memobase-engine/internal/...
go test ./services/memobase-engine/internal/domain/... -v -count=1
go test ./services/memobase-engine/internal/usecase/... -v -count=1
```

---

## Ghi chú triển khai

- `errgroup.WithContext` — nếu goroutine 1 (profile) fail → cancel goroutine 2 (event)
- Event goroutine failure: return `nil` (không phải `err`) để profile vẫn tiếp tục
- `serializeBlobs(blobs)`: concatenate blob contents với separator, chat blobs format "user: ... \nassistant: ..."
- `ApplyMergeResult`: dùng transaction — ADD + UPDATE + DELETE trong 1 DB transaction
- `ivfflat` index cần `lists` parameter — dùng `CREATE INDEX ... WITH (lists=100)` cho dev
