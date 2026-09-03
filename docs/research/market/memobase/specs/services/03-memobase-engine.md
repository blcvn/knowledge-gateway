# 03 — Memobase Memory Engine Service

> **gRPC**: 9042 | **Health**: 9092

---

## 1. Purpose

Core LLM processing pipeline: nhận buffer-ready events, thực hiện profile extraction, YOLO merge, event summary. Đây là service **CPU/LLM-intensive**, tách biệt để scaling độc lập.

---

## 2. Clean Architecture

```
services/memobase-engine/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Profile, ProfileSlot, MergeResult, EventSummary
│   │   ├── value_object.go     # Topic, SubTopic, MergeAction(add/update/delete)
│   │   ├── event.go            # EngineCompletedEvent, EngineFailedEvent
│   │   └── errors.go           # ErrLLMUnavailable, ErrParseProfile
│   ├── usecase/
│   │   ├── process_blobs.go    # Orchestrate full pipeline
│   │   ├── extract_profile.go  # LLM Call #1+#2: entry_summary + extract
│   │   ├── merge_profile.go    # LLM Call #3: YOLO merge
│   │   ├── organize_profile.go # Reorganize subtopics (no LLM)
│   │   ├── re_summary.go       # Conditional re-summary for oversized slots
│   │   ├── summarize_event.go  # Event summary + gist generation
│   │   ├── tag_event.go        # Conditional event tagging
│   │   ├── port/
│   │   │   ├── input.go        # ProcessBlobsUseCase interface
│   │   │   └── output.go       # LLMClient, EmbedderClient, ProfileRepo,
│   │   │                       #   EventRepo, BlobReader, PromptProvider,
│   │   │                       #   EventPublisher
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go     # ProcessResult{add,update,delete,event_id}
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go      # memobase.engine.v1.EngineService impl
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── profile_repo.go   # user_profiles CRUD
│   │   │       └── event_repo.go     # user_events + user_event_gists
│   │   ├── client/
│   │   │   ├── llm_client.go         # Bifrost / OpenAI LLM calls
│   │   │   ├── embedder_client.go    # OpenAI / Jina / Ollama embeddings
│   │   │   └── ingestion_client.go   # gRPC → ingestion for blob fetch
│   │   ├── event/
│   │   │   ├── publisher.go    # NATS: memobase.engine.completed/failed
│   │   │   └── subscriber.go   # NATS: memobase.buffer.ready
│   │   └── prompt/
│   │       ├── registry.go     # Prompt template registry
│   │       ├── en/             # English prompts
│   │       │   ├── extract_profile.go
│   │       │   ├── merge_profile_yolo.go
│   │       │   └── summary_entry_chats.go
│   │       └── zh/             # Chinese prompts
│   │           ├── extract_profile.go
│   │           ├── merge_profile_yolo.go
│   │           └── summary_entry_chats.go
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Entities

```go
type MergeAction string
const (
    MergeAdd    MergeAction = "add"
    MergeUpdate MergeAction = "update"
    MergeDelete MergeAction = "delete"
)

type ProfileSlot struct {
    Topic    string
    SubTopic string
    Content  string
}

type MergeDecision struct {
    Action    MergeAction
    ProfileID *uuid.UUID     // for update/delete
    Slot      ProfileSlot    // for add/update
}

type MergeResult struct {
    Add    []ProfileSlot
    Update []struct{ ProfileID uuid.UUID; Slot ProfileSlot }
    Delete []uuid.UUID
}

type EventSummary struct {
    EventTip   string           // Full event summary
    EventGists []string         // Fine-grained lines
    EventTags  []EventTag       // Custom tags
    ProfileDelta []ProfileSlot  // Delta from this session
    Embedding  []float32        // Event embedding vector
}

type EventTag struct {
    Tag   string
    Value string
}
```

---

## 4. Pipeline Flow: ProcessBlobs

```
NATS: memobase.buffer.ready → subscriber
                │
                ▼
    ┌──── ProcessBlobsUseCase ─────────────────────────┐
    │                                                   │
    │ 1. Fetch blobs from ingestion (gRPC or direct DB) │
    │ 2. Truncate to max_process_token_size (16384)     │
    │ 3. Get project profile config                     │
    │ 4. Get current user profiles                      │
    │                                                   │
    │ ┌─── LLM Call #1 ──────────────────────────────┐ │
    │ │ entry_chat_summary(blobs, config, profiles)   │ │
    │ │ → user_memo_str                               │ │
    │ └───────────────────────────────────────────────┘ │
    │                                                   │
    │ ┌─── errgroup.Go (parallel) ──────────────────┐  │
    │ │                                              │  │
    │ │  goroutine 1: Profile Processing             │  │
    │ │  ┌── LLM Call #2 ─────────────────────────┐ │  │
    │ │  │ extract_topics(memo_str, schema)         │ │  │
    │ │  │ → extracted facts + attributes           │ │  │
    │ │  └─────────────────────────────────────────┘ │  │
    │ │  ┌── LLM Call #3 ─────────────────────────┐ │  │
    │ │  │ merge_yolo(facts, existing_profiles)    │ │  │
    │ │  │ → MergeResult{add, update, delete}      │ │  │
    │ │  └─────────────────────────────────────────┘ │  │
    │ │  organize_profiles (if subtopics > limit)   │  │
    │ │  re_summary (if content > token limit)      │  │
    │ │                                              │  │
    │ │  goroutine 2: Event Processing              │  │
    │ │  tag_event (if event_tags configured)       │  │
    │ │                                              │  │
    │ └──────────────────────────────────────────────┘  │
    │                                                   │
    │ 5. DB: Upsert profiles (add + update + delete)    │
    │ 6. DB: Store event + embeddings + gists           │
    │ 7. NATS: memobase.engine.completed                │
    └───────────────────────────────────────────────────┘
```

**Total LLM calls**: Cố định 3 (entry_summary + extract + merge_yolo)

---

## 5. YOLO Merge Algorithm

```go
// merge_profile.go
func (uc *MergeProfileUseCase) Execute(ctx context.Context,
    extractedFacts []ProfileSlot,
    existingProfiles []ProfileSlot,
) (*MergeResult, error) {

    // Build LLM prompt with existing profiles (indexed)
    // and new extracted facts
    prompt := uc.promptProvider.MergeProfileYOLO(existingProfiles, extractedFacts)

    // Single LLM call (JSON mode)
    response, err := uc.llmClient.Complete(ctx, prompt, llm.WithJSONMode())

    // Parse response → MergeResult
    // { "add": [...], "update": [{index, memo}], "delete": [index] }
    return parseMergeResponse(response, existingProfiles)
}
```

---

## 6. NATS Events

| Subject | Payload | Direction |
|---------|---------|-----------|
| `memobase.buffer.ready` | `{user_id, project_id, buffer_ids[], blob_type}` | Subscribe |
| `memobase.engine.completed` | `{user_id, project_id, buffer_ids[], event_id, merge_result}` | Publish |
| `memobase.engine.failed` | `{user_id, project_id, buffer_ids[], error}` | Publish |
| `memobase.profile.changed` | `{user_id, project_id, action}` | Publish |
| `memobase.event.created` | `{user_id, project_id, event_id}` | Publish |

---

## 7. LLM & Embedding Port Interfaces

```go
// port/output.go

type LLMClient interface {
    Complete(ctx context.Context, prompt string, opts ...llm.Option) (string, error)
    CompleteJSON(ctx context.Context, prompt string, opts ...llm.Option) (json.RawMessage, error)
}

type EmbedderClient interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
}

type PromptProvider interface {
    EntrySummary(blobs []Blob, config ProfileConfig) string
    ExtractProfile(memoStr string, schema ProfileSchema) string
    MergeProfileYOLO(existing []ProfileSlot, facts []ProfileSlot) string
    SummarizeEvent(memoStr string) string
    TagEvent(memoStr string, tagDefs []EventTagDef) string
    Language() string  // "en" | "zh"
}
```

---

## 8. Configuration

```yaml
engine:
  grpc:
    port: 9042
  health:
    port: 9092
  llm:
    provider: "bifrost"           # bifrost | openai | ollama
    base_url: "${LLM_BASE_URL}"
    api_key: "${LLM_API_KEY}"
    model: "gpt-4o-mini"
    thinking_model: "o4-mini"
    max_tokens: 1024
  embedding:
    provider: "openai"            # openai | jina | ollama
    model: "text-embedding-3-small"
    dimension: 1536
    enabled: true
  profile:
    max_subtopics: 15
    max_slot_token_size: 128
    strict_mode: false
    validate_mode: true
    language: "en"
  nats:
    url: "nats://nats:4222"
    stream: "memobase"
    consumer_group: "engine"
    max_concurrent: 10            # Bulkhead for LLM calls
  database:
    url: "${DATABASE_URL}"
    pool_size: 25
```
