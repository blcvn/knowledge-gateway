# Solution: SOL-MB-002 — Memory Engine: Profile Extraction & YOLO Merge

**CR:** [CR-MB-002](../CR-MB-002-Memory-Engine-Profile-YOLO.md)  
**Wave:** 2 (Processing)  
**Priority:** Critical  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/memobase-engine` — core LLM processing service với **3-call fixed pipeline** (entry_summary → extract_topics → YOLO_merge) và **parallel processing** (profile + event goroutines chạy đồng thời).

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Không có 3-call pipeline | Implement `ProcessBlobsUseCase` orchestrating 3 sequential LLM calls |
| Không có YOLO merge | `MergeProfileUseCase` — single LLM call nhận indexed existing profiles + new facts |
| Không có parallel processing | `errgroup.WithContext` cho profile pipeline + event pipeline |
| Không có multilingual prompts | `pkg/prompt` registry với EN/ZH `PromptProvider` |
| Không có profile constraints | `OrganizeProfileUseCase` (no LLM) + `ReeSummaryUseCase` (LLM) |
| Không có event gist | `SummarizeEventUseCase` split event_tip by "-" prefix lines |

---

## 2. Pipeline Architecture

```
NATS: memobase.buffer.ready → EngineService.ProcessBlobs()
                                      │
                            ┌─────────▼──────────┐
                            │  Fetch blobs (gRPC) │
                            │  Truncate to 16384  │
                            │  Get profile config │
                            │  Get existing profs │
                            └─────────┬──────────┘
                                      │
                            ┌─────────▼──────────┐
                            │  LLM Call #1        │
                            │  entry_summary      │
                            │  → memoStr          │
                            └─────────┬──────────┘
                                      │
                    ┌─────────────────┴────────────────────┐
                    │ errgroup.WithContext(ctx)             │
         ┌──────────▼──────────┐          ┌───────────────▼──────────┐
         │ Goroutine 1: Profile │          │ Goroutine 2: Event        │
         │ LLM #2: extract_topics          │ summarize_event (LLM)     │
         │ LLM #3: YOLO merge  │          │ tag_event (optional LLM)  │
         │ organize (no LLM)   │          │ embed event_tip (embedder)│
         │ validate (opt LLM)  │          └───────────────┬──────────┘
         └──────────┬──────────┘                          │
                    └─────────────────┬────────────────────┘
                                      │ g.Wait()
                            ┌─────────▼──────────┐
                            │  Persist:           │
                            │  - upsert profiles  │
                            │  - save event+gists │
                            │  NATS publish:      │
                            │  - engine.completed │
                            │  - profile.changed  │
                            │  - event.created    │
                            └────────────────────┘
```

---

## 3. Vị trí trong Codebase

```
vnp-memory/
├── services/
│   └── memobase-engine/               ← [NEW]
│       ├── cmd/server/main.go
│       ├── internal/
│       │   ├── domain/
│       │   │   ├── entity.go          # Profile, ProfileSlot, MergeResult, EventSummary
│       │   │   ├── value_object.go    # MergeAction (add|update|delete), Topic, SubTopic
│       │   │   ├── event.go           # Domain events
│       │   │   └── errors.go
│       │   ├── usecase/
│       │   │   ├── process_blobs.go   # Orchestrator
│       │   │   ├── extract_profile.go # LLM Call #1 + #2
│       │   │   ├── merge_profile.go   # LLM Call #3 (YOLO)
│       │   │   ├── organize_profile.go
│       │   │   ├── validate_profile.go
│       │   │   ├── re_summary.go
│       │   │   ├── summarize_event.go
│       │   │   ├── tag_event.go
│       │   │   └── port/
│       │   ├── adapter/
│       │   │   ├── grpc/handler.go
│       │   │   ├── repository/postgres/
│       │   │   │   ├── profile_repo.go
│       │   │   │   └── event_repo.go
│       │   │   ├── client/
│       │   │   │   ├── llm_client.go       # via pkg/adapters/llm
│       │   │   │   ├── embedder_client.go  # via pkg/adapters/embedder
│       │   │   │   └── ingestion_client.go # gRPC fetch blobs
│       │   │   └── event/
│       │   │       ├── publisher.go
│       │   │       └── subscriber.go      # memobase.buffer.ready
│       │   └── infra/
└── pkg/                                   ← Shared packages (CR-007)
    ├── adapters/llm/
    ├── adapters/embedder/
    ├── tokenizer/
    └── prompt/
```

---

## 4. Database Schema

**File:** `services/memobase-engine/internal/infra/migrations/001_init.up.sql`

```sql
-- User profiles (extracted by YOLO merge)
CREATE TABLE IF NOT EXISTS user_profiles (
    id         UUID        NOT NULL,
    project_id VARCHAR     NOT NULL,
    user_id    UUID        NOT NULL,
    content    TEXT        NOT NULL,
    attributes JSONB       NOT NULL,  -- {topic, sub_topic}
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_profiles_user ON user_profiles(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_topic ON user_profiles USING gin(attributes);

-- User events (timeline)
CREATE TABLE IF NOT EXISTS user_events (
    id         UUID        NOT NULL,
    project_id VARCHAR     NOT NULL,
    user_id    UUID        NOT NULL,
    event_data JSONB       NOT NULL,  -- {event_tip, event_tags[], profile_delta}
    embedding  vector(1536),           -- pgvector (null if embedding disabled)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, project_id),
    FOREIGN KEY (user_id, project_id) REFERENCES users(id, project_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_events_user ON user_events(user_id, project_id);
CREATE INDEX IF NOT EXISTS idx_user_events_time ON user_events(user_id, project_id, created_at DESC);

-- User event gists (fine-grained descriptions)
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
CREATE INDEX IF NOT EXISTS idx_user_event_gists_user ON user_event_gists(user_id, project_id);
```

---

## 5. YOLO Merge Algorithm — Chi tiết Implementation

### 5.1 Prompt Design

Prompt LLM #3 (YOLO merge) phải:
1. Liệt kê existing profiles với **index number** `[0]`, `[1]`, ...
2. Liệt kê new extracted facts
3. Yêu cầu LLM output JSON với 3 fields: `add`, `update`, `delete`

```
System: You are a user profile manager. Merge the new facts with existing profiles.

Existing profiles (indexed):
[0] basic_info::name: Alice
[1] work::company: Acme Corp
[2] lifestyle::diet: vegetarian

New facts extracted:
- basic_info::name: Alice (unchanged)
- work::company: Beta Corp (changed job)
- lifestyle::diet: (no longer vegetarian - remove)
- interest::music: jazz (new)

Output JSON:
{
  "add": [{"topic": "interest", "sub_topic": "music", "content": "jazz"}],
  "update": [{"index": 1, "topic": "work", "sub_topic": "company", "content": "Beta Corp"}],
  "delete": [2]
}
```

### 5.2 Profile Constraints Post-Merge

```go
// organize_profile.go — không cần LLM
func Organize(slots []ProfileSlot, config ProfileSchema) []ProfileSlot {
    // Group by topic
    byTopic := groupByTopic(slots)
    
    var result []ProfileSlot
    for topic, topicSlots := range byTopic {
        if len(topicSlots) > config.MaxSubtopics {
            // Giữ MaxSubtopics slots mới nhất (by updated_at)
            topicSlots = topicSlots[:config.MaxSubtopics]
        }
        result = append(result, topicSlots...)
    }
    return result
}

// re_summary.go — gọi LLM nếu slot quá dài
func ReSummary(slot ProfileSlot, maxTokens int, llm LLMClient) ProfileSlot {
    tokenCount := tokenizer.Count(slot.Content)
    if tokenCount <= maxTokens {
        return slot
    }
    // LLM: "Summarize this profile slot to under N tokens: ..."
    summarized, _ := llm.Complete(ctx, buildReSummaryPrompt(slot, maxTokens))
    slot.Content = summarized
    return slot
}
```

### 5.3 Strict Mode

```go
// validate_profile.go
func FilterBySchema(slots []ProfileSlot, schema ProfileSchema) []ProfileSlot {
    if !schema.StrictMode {
        return slots
    }
    allowed := schema.AllowedTopics() // from Additional + built-in topics
    var result []ProfileSlot
    for _, s := range slots {
        if allowed.Contains(s.Topic) {
            result = append(result, s)
        }
    }
    return result
}
```

---

## 6. Event Gist Generation

```go
// summarize_event.go
func (uc *SummarizeEventUseCase) Execute(ctx context.Context, memoStr string) (*EventSummary, error) {
    // LLM call (embedded trong goroutine 2 của errgroup)
    raw, err := uc.llm.CompleteJSON(ctx, uc.prompts.SummarizeEvent(memoStr))
    
    var eventData struct {
        EventTip     string     `json:"event_tip"`
        EventTags    []EventTag `json:"event_tags"`
        ProfileDelta []any      `json:"profile_delta"`
    }
    json.Unmarshal(raw, &eventData)
    
    // Parse gists từ event_tip (các dòng bắt đầu bằng "-")
    gists := parseGists(eventData.EventTip)
    // "- Discussed project deadline\n- Mentioned feeling stressed"
    // → ["Discussed project deadline", "Mentioned feeling stressed"]
    
    return &EventSummary{
        EventTip:   eventData.EventTip,
        EventGists: gists,
        EventTags:  eventData.EventTags,
    }, nil
}

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

## 7. Bulkhead cho LLM Calls

```go
// adapter/client/llm_client.go

type LLMClientWithBulkhead struct {
    inner    llm.LLMClient
    bulkhead *resilience.Bulkhead  // pkg/resilience
}

func (c *LLMClientWithBulkhead) CompleteJSON(ctx context.Context, prompt string, opts ...llm.Option) (json.RawMessage, error) {
    var result json.RawMessage
    err := c.bulkhead.Execute(ctx, func() error {
        var err error
        result, err = c.inner.CompleteJSON(ctx, prompt, opts...)
        return err
    })
    return result, err
}
// Bulkhead default: max 10 concurrent LLM calls per engine instance
// Config: engine.nats.max_concurrent = 10
```

---

## 8. NATS Consumer

```go
// adapter/event/subscriber.go

func (s *Subscriber) Start(ctx context.Context) error {
    // Subscribe với WorkQueue semantics (at-least-once)
    sub, err := s.js.Subscribe("memobase.buffer.ready",
        s.handleBufferReady,
        nats.Durable("memobase-engine"),
        nats.MaxDeliver(3),  // retry 3 lần
        nats.AckWait(120*time.Second),  // LLM calls có thể lâu
    )
    // ...
}

func (s *Subscriber) handleBufferReady(msg *nats.Msg) {
    var payload BufferReadyPayload
    json.Unmarshal(msg.Data, &payload)
    
    // Acquire bulkhead slot
    err := s.processUseCase.Execute(ctx, ProcessBlobsRequest{
        UserID:    payload.UserID,
        ProjectID: payload.ProjectID,
        BufferIDs: payload.BufferIDs,
        BlobType:  payload.BlobType,
    })
    
    if err != nil {
        // Sẽ được NATS redeliver (max 3 lần)
        msg.Nak()
        return
    }
    msg.Ack()
}
```

---

## 9. Prompt Template Registry

**Cấu trúc** (`pkg/prompt/`):

```
pkg/prompt/
├── provider.go          # PromptProvider interface
├── registry.go          # map[string]PromptProvider
├── en/
│   ├── provider.go      # ENPromptProvider implements PromptProvider
│   ├── entry_chat_summary.tmpl
│   ├── extract_profile.tmpl
│   ├── merge_profile_yolo.tmpl
│   ├── summarize_event.tmpl
│   └── tag_event.tmpl
└── zh/
    ├── provider.go      # ZHPromptProvider implements PromptProvider
    └── *.tmpl           # Chinese versions
```

**Template engine:** Go `text/template` với custom delimiters `{{` `}}`.

---

## 10. Configuration

```yaml
engine:
  server:
    grpc_port: 9042
    health_port: 9092

  llm:
    provider: "bifrost"            # MEMOBASE_LLM_PROVIDER
    api_key: "${MEMOBASE_LLM_API_KEY}"
    model: "gpt-4o-mini"           # MEMOBASE_BEST_LLM_MODEL
    max_tokens: 1024
    max_process_token_size: 16384  # truncate blobs before LLM

  embedding:
    provider: "openai"             # MEMOBASE_EMBEDDING_PROVIDER
    model: "text-embedding-3-small"
    dimension: 1536
    enabled: true                  # MEMOBASE_ENABLE_EVENT_EMBEDDING

  profile:
    max_subtopics: 15              # MEMOBASE_MAX_PROFILE_SUBTOPICS
    max_slot_token_size: 128       # MEMOBASE_MAX_PRE_PROFILE_TOKEN_SIZE
    strict_mode: false             # MEMOBASE_PROFILE_STRICT_MODE
    validate_mode: true
    language: "en"                 # MEMOBASE_LANGUAGE

  nats:
    url: "${NATS_URL}"
    stream: "memobase"
    consumer_group: "engine"
    max_concurrent: 10             # bulkhead semaphore size

  services:
    ingestion:
      address: "memobase-ingestion:9041"
      timeout: 30s
    admin:
      address: "memobase-admin:9045"
      timeout: 5s
    event:
      address: "memobase-event:9044"
      timeout: 10s

  database:
    url: "${DATABASE_URL}"
    pool_size: 20
```

---

## 11. Testing Strategy

### Unit Tests
- `TestProcessBlobsUseCase_HappyPath` — verify 3 LLM calls invoked với đúng thứ tự
- `TestMergeProfileUseCase_AddUpdateDelete` — YOLO merge decisions
- `TestOrganizeProfile_ExceedsMaxSubtopics` — verify truncation
- `TestSummarizeEvent_GistParsing` — split by "- " prefix
- `TestBulkhead_MaxConcurrent` — 15 goroutines → only 10 execute simultaneously

### Integration Tests
- `TestEngineProcessE2E` — mock LLM → verify profiles upserted + event saved
- `TestLanguageSwitching` — `language=zh` → ZH prompt provider selected
- `TestStrictModeFiltering` — only defined schema topics survive

---

## 12. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| LLM JSON parse failure | Trung bình | Retry with `Nak()`, fallback graceful error |
| LLM output index out of range (YOLO merge) | Trung bình | Validate indices trước khi apply |
| Embedding API timeout (goroutine 2) | Thấp | errgroup cancel ngay khi có error |
| Profile schema update chưa được reload | Thấp | Subscribe `memobase.admin.project.updated` → invalidate config cache |
| Cost overrun với validate_mode LLM calls | Thấp | validate_mode chỉ chạy nếu `validate_mode=true` và slot count > threshold |
