# Change Request: CR-MB-002 — Memory Engine: Profile Extraction & YOLO Merge

**CR ID:** CR-MB-002  
**Component:** `services/memobase-engine` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** memobase PRD §5.3 (F-3), SRS §3.5, specs/services/03-memobase-engine.md  
**Maps to Python:** `controllers/modal/chat/`, `llms/`, `prompts/`

---

## 1. Mô tả

Xây dựng **memobase-engine** service — LLM processing pipeline thực hiện:
1. **Profile Extraction** (3 fixed LLM calls per flush): entry summary → extract topics → YOLO merge.
2. **Event Summary & Gist Generation**: tóm tắt sự kiện + fine-grained descriptions.
3. **Event Tagging**: custom temporal attributes (emotion, goal, etc.).
4. **Profile Organization**: tự động tổ chức subtopics vượt giới hạn.
5. **Profile Validation**: loại bỏ meaningless profile slots.

**Key innovation: YOLO Merge** — merge profiles mới với profiles hiện có trong **1 LLM call duy nhất**, giảm từ 3-10 calls xuống còn cố định 3 calls.

---

## 2. Vấn đề hiện tại

VNP Memory hiện tại:
- ✅ Có basic LLM extraction.
- ❌ Không có **3-call fixed pipeline** (entry_summary + extract_topics + YOLO_merge).
- ❌ Không có **YOLO merge algorithm** (merge với 1 LLM call, add/update/delete decisions).
- ❌ Không có **event gist generation** (fine-grained descriptions từ event_tip).
- ❌ Không có **event tagging** (custom tags: emotion, goal, etc.).
- ❌ Không có **profile organization** (auto-reorganize khi subtopics > max_profile_subtopics).
- ❌ Không có **profile validation** (remove meaningless slots).
- ❌ Không có **profile strict mode** (chỉ collect defined schema).
- ❌ Không có **multilingual prompts** (EN/ZH template registry).
- ❌ Không có **max_process_token_size** truncation (16384 tokens).
- ❌ Không có **parallel processing** (errgroup cho profile + event goroutines).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/memobase-engine/`

**Port:** `9042` (gRPC internal), **Health:** `9092`

```
services/memobase-engine/
├── internal/
│   ├── domain/
│   │   ├── entity.go          # Profile, ProfileSlot, MergeResult, EventSummary
│   │   ├── value_object.go    # MergeAction (add|update|delete), Topic, SubTopic
│   │   ├── event.go           # EngineCompletedEvent, EngineFailedEvent
│   │   └── errors.go          # ErrLLMUnavailable, ErrParseProfile
│   ├── usecase/
│   │   ├── process_blobs.go   # Pipeline orchestrator (3 LLM calls)
│   │   ├── extract_profile.go # LLM Call #1 (entry_summary) + LLM Call #2 (extract_topics)
│   │   ├── merge_profile.go   # LLM Call #3 (YOLO merge)
│   │   ├── organize_profile.go # Reorganize subtopics (no LLM)
│   │   ├── validate_profile.go # Remove meaningless slots (optional LLM)
│   │   ├── re_summary.go      # Re-summarize oversized slots
│   │   ├── summarize_event.go # Event tip + gist generation
│   │   ├── tag_event.go       # Conditional event tagging
│   │   └── port/
│   │       ├── input.go       # ProcessBlobsUseCase, ExtractProfileUseCase interfaces
│   │       └── output.go      # LLMClient, EmbedderClient, ProfileRepo, EventRepo,
│   │                          #   BlobReader, PromptProvider, EventPublisher
│   ├── adapter/
│   │   ├── grpc/handler.go    # memobase.engine.v1.EngineService gRPC impl
│   │   ├── repository/postgres/
│   │   │   ├── profile_repo.go  # user_profiles CRUD + upsert
│   │   │   └── event_repo.go    # user_events + user_event_gists + embeddings
│   │   ├── client/
│   │   │   ├── llm_client.go    # Bifrost/OpenAI LLM calls with JSON mode
│   │   │   ├── embedder_client.go # OpenAI/Jina/Ollama embeddings
│   │   │   └── ingestion_client.go # Fetch blobs from ingestion service
│   │   ├── event/
│   │   │   ├── publisher.go   # NATS: engine.completed/failed/profile.changed/event.created
│   │   │   └── subscriber.go  # NATS: memobase.buffer.ready
│   │   └── prompt/
│   │       ├── registry.go    # Prompt template registry (language-aware)
│   │       ├── en/
│   │       │   ├── entry_chat_summary.go
│   │       │   ├── extract_profile.go
│   │       │   └── merge_profile_yolo.go
│   │       └── zh/
│   │           ├── entry_chat_summary.go
│   │           ├── extract_profile.go
│   │           └── merge_profile_yolo.go
```

### 3.2. Domain Models

```go
// internal/domain/entity.go

type ProfileSlot struct {
    Topic    string `json:"topic"`
    SubTopic string `json:"sub_topic"`
    Content  string `json:"content"`
}

type MergeAction string
const (
    MergeAdd    MergeAction = "add"
    MergeUpdate MergeAction = "update"
    MergeDelete MergeAction = "delete"
)

type MergeDecision struct {
    Action    MergeAction
    ProfileID *uuid.UUID   // for update/delete (index into existing)
    Slot      ProfileSlot  // for add/update
}

type MergeResult struct {
    Add    []ProfileSlot
    Update []struct{ ProfileID uuid.UUID; Slot ProfileSlot }
    Delete []uuid.UUID
    Stats  struct{ Added, Updated, Deleted int }
}

type EventSummary struct {
    EventTip     string       // Full event summary string
    EventGists   []string     // Fine-grained lines (split by "-")
    EventTags    []EventTag   // Custom temporal attributes
    ProfileDelta []ProfileSlot
    Embedding    []float32    // Event embedding vector
}

type EventTag struct {
    Tag   string `json:"tag"`
    Value string `json:"value"`
}
```

### 3.3. 3-Step Fixed LLM Pipeline

```go
// internal/usecase/process_blobs.go

func (uc *ProcessBlobsUseCase) Execute(ctx, req ProcessBlobsRequest) (*ProcessResult, error) {
    // --- Setup ---
    // 1. Fetch blobs from ingestion service (gRPC)
    blobs, _ := uc.blobReader.GetBlobsForProcessing(ctx, req.BufferIDs)
    
    // 2. Truncate to max_process_token_size (16384 tokens)
    blobs = uc.tokenizer.TruncateBlobs(blobs, uc.config.MaxProcessTokenSize)
    
    // 3. Get project profile config (schema, strict_mode, language)
    profileConfig, _ := uc.adminClient.GetProfileConfig(ctx, req.ProjectID)
    
    // 4. Get current user profiles (for merge)
    existingProfiles, _ := uc.profileRepo.GetByUser(ctx, req.UserID, req.ProjectID)

    // --- LLM Call #1: Entry Summary ---
    memoStr, _ := uc.llmClient.CompleteJSON(ctx,
        uc.promptProvider.EntrySummary(blobs, profileConfig),
    )
    // → user_memo_str: synthesized representation of conversations

    // --- Parallel Processing (errgroup) ---
    var mergeResult *MergeResult
    var eventSummary *EventSummary

    g, ctx := errgroup.WithContext(ctx)

    // goroutine 1: Profile Pipeline
    g.Go(func() error {
        // LLM Call #2: extract_topics
        extractedFacts, _ := uc.llmClient.CompleteJSON(ctx,
            uc.promptProvider.ExtractProfile(memoStr, profileConfig.Schema),
        )

        // LLM Call #3: YOLO merge
        mergeResult, _ = uc.mergeProfile.Execute(ctx, extractedFacts, existingProfiles)

        // Post-processing (no LLM)
        mergeResult = uc.organizeProfile.Execute(mergeResult, profileConfig)
        mergeResult = uc.validateProfile.Execute(mergeResult, profileConfig)  // if validate_mode
        return nil
    })

    // goroutine 2: Event Pipeline
    g.Go(func() error {
        eventSummary, _ = uc.summarizeEvent.Execute(ctx, memoStr)
        if len(profileConfig.EventTagDefs) > 0 {
            eventSummary.EventTags, _ = uc.tagEvent.Execute(ctx, memoStr, profileConfig.EventTagDefs)
        }
        if uc.embedder.IsEnabled() {
            eventSummary.Embedding, _ = uc.embedder.Embed(ctx, []string{eventSummary.EventTip})
        }
        return nil
    })

    g.Wait()

    // --- Persist ---
    // 5. DB: Upsert profiles (add + update + delete)
    uc.profileRepo.ApplyMergeResult(ctx, req.UserID, req.ProjectID, mergeResult)

    // 6. DB: Store event + gists + embeddings
    eventID, _ := uc.eventRepo.SaveEvent(ctx, req.UserID, req.ProjectID, eventSummary)

    // 7. NATS: engine.completed, profile.changed, event.created
    uc.eventPublisher.PublishEngineCompleted(ctx, req, eventID, mergeResult)
    uc.eventPublisher.PublishProfileChanged(ctx, req.UserID, req.ProjectID)
    uc.eventPublisher.PublishEventCreated(ctx, req.UserID, req.ProjectID, eventID)

    return &ProcessResult{
        Added:   len(mergeResult.Add),
        Updated: len(mergeResult.Update),
        Deleted: len(mergeResult.Delete),
        EventID: eventID,
    }, nil
}
```

### 3.4. YOLO Merge Algorithm

```go
// internal/usecase/merge_profile.go

func (uc *MergeProfileUseCase) Execute(ctx context.Context,
    extractedFacts []ProfileSlot,
    existingProfiles []ProfileSlot,  // indexed 0..N-1
) (*MergeResult, error) {
    // Build LLM prompt:
    // - Existing profiles WITH index numbers: [0] basic_info::name: Alice, [1] ...
    // - New extracted facts
    prompt := uc.promptProvider.MergeProfileYOLO(existingProfiles, extractedFacts)

    // Single LLM call (JSON mode)
    // Response format:
    // {
    //   "add": [{topic, sub_topic, content}],       // new facts to add
    //   "update": [{index: 0, topic, sub_topic, content}],  // update existing
    //   "delete": [1, 3, 5]                         // indices to delete
    // }
    raw, err := uc.llmClient.CompleteJSON(ctx, prompt, llm.WithJSONMode())

    return parseMergeResponse(raw, existingProfiles)
}

// Profile constraints after merge:
// - Per-topic subtopics: max max_profile_subtopics (default 15)
//   → auto organize_profile (no LLM) if exceeded
// - Per-slot content: max max_slot_token_size (default 128 tokens)
//   → re_summary (LLM) if exceeded
// - Strict mode: only keep slots with topic in schema
// - Validate mode: remove "meaningless" slots (LLM check)
```

### 3.5. Event Summary & Gist Generation

```go
// internal/usecase/summarize_event.go

// LLM extracts event_tip from memoStr
// event_tip is split into gists by lines starting with "-"
// e.g. event_tip = "- Discussed project deadline\n- Mentioned feeling stressed"
// → gists = ["Discussed project deadline", "Mentioned feeling stressed"]

type EventData struct {
    EventTip     string     `json:"event_tip"`
    EventTags    []EventTag `json:"event_tags"`
    ProfileDelta []any      `json:"profile_delta"`
}
```

### 3.6. Profile Schema (Configurable)

```go
// pkg/profile/schema.go

type ProfileSchema struct {
    Language         string              // "en" | "zh"
    StrictMode       bool                // Only collect defined profiles
    ValidateMode     bool                // Remove meaningless slots
    MaxSubtopics     int                 // per-topic limit (default 15)
    MaxSlotTokenSize int                 // per-slot limit (default 128)
    EventTagDefs     []EventTagDef       // Custom event tags to extract
    Additional       []ProfileTopicDef   // Developer-defined topics
    Overwrite        []ProfileTopicDef   // Override built-in topics
}

type ProfileTopicDef struct {
    Topic       string   `yaml:"topic"`
    Description string   `yaml:"description"`
    SubTopics   []string `yaml:"sub_topics,omitempty"`
}

type EventTagDef struct {
    Tag         string   `yaml:"tag"`
    Description string   `yaml:"description"`
    Values      []string `yaml:"values,omitempty"`  // allowed values
}

// Default built-in profile topics (from memobase):
// basic_info, interest, lifestyle, attitude, work, social, health
```

### 3.7. Prompt Templates (EN/ZH)

```go
// adapter/prompt/registry.go

type PromptProvider interface {
    // LLM Call #1: Synthesize conversations into user memo
    EntrySummary(blobs []Blob, config ProfileSchema) string

    // LLM Call #2: Extract structured profile facts
    ExtractProfile(memoStr string, schema ProfileSchema) string

    // LLM Call #3: YOLO merge with existing profiles
    MergeProfileYOLO(existing []ProfileSlot, facts []ProfileSlot) string

    // Event summary extraction
    SummarizeEvent(memoStr string) string

    // Conditional event tagging
    TagEvent(memoStr string, tagDefs []EventTagDef) string

    Language() string  // "en" | "zh"
}

// Registry: auto-select EN or ZH based on config.language
var PromptRegistry = map[string]PromptProvider{
    "en": &ENPromptProvider{},
    "zh": &ZHPromptProvider{},
}
```

### 3.8. gRPC API

```protobuf
service EngineService {
    rpc ProcessBlobs(ProcessBlobsRequest) returns (ProcessBlobsResponse);
    rpc ExtractProfile(ExtractProfileRequest) returns (ExtractProfileResponse);
    rpc GetEngineStatus(GetEngineStatusRequest) returns (GetEngineStatusResponse);
}

message ProcessBlobsRequest {
    string user_id = 1;
    string project_id = 2;
    repeated string buffer_ids = 3;
    string blob_type = 4;
}

message ProcessBlobsResponse {
    int32 profiles_added = 1;
    int32 profiles_updated = 2;
    int32 profiles_deleted = 3;
    string event_id = 4;
    int32 llm_calls_used = 5;     // always 3
}
```

### 3.9. NATS Events

| Subject | Direction | Payload |
|---|---|---|
| `memobase.buffer.ready` | Subscribe | `{user_id, project_id, buffer_ids[], blob_type}` |
| `memobase.engine.completed` | Publish | `{user_id, project_id, buffer_ids[], event_id, merge_result}` |
| `memobase.engine.failed` | Publish | `{user_id, project_id, buffer_ids[], error}` |
| `memobase.profile.changed` | Publish | `{user_id, project_id, action: "merge"}` |
| `memobase.event.created` | Publish | `{user_id, project_id, event_id}` |

---

## 4. Configuration

```yaml
engine:
  grpc:
    port: 9042
  health:
    port: 9092
  llm:
    provider: "bifrost"            # bifrost | openai | ollama
    model: "gpt-4o-mini"
    thinking_model: "o4-mini"
    max_tokens: 1024
    max_process_token_size: 16384  # truncate blobs before LLM
  embedding:
    provider: "openai"
    model: "text-embedding-3-small"
    dimension: 1536
    enabled: true
  profile:
    max_subtopics: 15
    max_slot_token_size: 128
    strict_mode: false
    validate_mode: true
    language: "en"                 # en | zh
  nats:
    url: "nats://nats:4222"
    stream: "memobase"
    consumer_group: "engine"
    max_concurrent: 10             # Bulkhead for LLM calls
```

---

## 5. Acceptance Criteria

- [ ] ProcessBlobs với ChatBlob "My name is Alice, I work at Acme" → `profiles_added ≥ 1` với `topic: basic_info, sub_topic: name, content: Alice`.
- [ ] LLM calls per flush = cố định 3 (entry_summary + extract_topics + merge_yolo), không phụ thuộc số profiles.
- [ ] Merge update: ingest "Alice now works at Beta Corp" sau khi đã có "Alice works at Acme" → profile `work::company` được UPDATE (không tạo duplicate).
- [ ] Merge delete: ingest "Alice is no longer vegetarian" → profile `lifestyle::diet: vegetarian` bị DELETE.
- [ ] `max_profile_subtopics = 15` enforced: topic có >15 subtopics → auto-organized (giảm về ≤15).
- [ ] `profile_strict_mode = true`: chỉ extract profiles thuộc schema defined → `basic_info::unrelated_topic` bị bỏ qua.
- [ ] Event gist generation: event_tip "- discussed project\n- mentioned stress" → `event_gists = ["discussed project", "mentioned stress"]`.
- [ ] Event tagging với `tag_defs = [{"tag": "emotion"}]` → `event_tags = [{tag: "emotion", value: "stressed"}]`.
- [ ] Profile embedding: nếu `enable_event_embedding = true` → event.embedding vector tồn tại trong DB.
- [ ] Chinese prompt: `language = "zh"` → LLM prompt và response được dùng template ZH.
- [ ] `max_concurrent = 10` bulkhead: chỉ tối đa 10 concurrent LLM calls.
