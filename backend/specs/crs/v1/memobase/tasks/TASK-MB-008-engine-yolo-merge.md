# TASK-MB-008 — `services/memobase-engine` YOLO Merge, Organize, NATS & gRPC

**Wave:** 2 (LLM Processing)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-MB-007 (engine domain + pipeline)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-MB-002 §5, §6, §7, §8](../solutions/SOL-MB-002-Memory-Engine-Profile-YOLO.md)

**Trạng thái:** ✅ Implemented  
**Ghi chú:** memobase-pipeline: 15 .go - YOLO merge pipeline  
---

## Mục tiêu

Hoàn thiện `services/memobase-engine/` với: MergeProfileUseCase (YOLO merge algorithm với index validation), OrganizeProfileUseCase (post-merge constraints, no LLM), ValidateProfileUseCase (strict mode filter), ReSummaryUseCase (slot truncation), NATS subscriber, gRPC handler, và monolith bootstrap.

---

## Các file cần tạo

### 1. `internal/usecase/merge_profile.go` — YOLO Merge

```go
type MergeProfileUseCase struct {
    llm     llm.LLMClient
    prompts *prompt.Registry
    lang    string
}

func (uc *MergeProfileUseCase) Execute(
    ctx context.Context,
    existingSlots []domain.ProfileSlot,
    newFacts string,
) (*domain.MergeResult, error) {
    // 1. Format existing slots with index for LLM prompt
    promptSlots := make([]promptpkg.ProfileSlot, len(existingSlots))
    for i, s := range existingSlots {
        promptSlots[i] = promptpkg.ProfileSlot{
            Index:    i,
            Topic:    s.Topic,
            SubTopic: s.SubTopic,
            Content:  s.Content,
        }
    }

    provider := uc.prompts.GetOrDefault(uc.lang)
    promptStr := provider.MergeProfileYOLO(promptSlots, newFacts)

    // 2. LLM Call #3 (YOLO)
    raw, err := uc.llm.CompleteJSON(ctx, promptStr, llm.WithJSONMode(), llm.WithMaxTokens(1024))
    if err != nil { return nil, fmt.Errorf("merge LLM: %w", err) }

    // 3. Parse merge action
    var action domain.MergeAction
    if err := json.Unmarshal(raw, &action); err != nil {
        return nil, fmt.Errorf("%w: %v", domain.ErrLLMParseFailed, err)
    }

    // 4. Validate indices (prevent out-of-range panics)
    for _, upd := range action.Update {
        if upd.Index < 0 || upd.Index >= len(existingSlots) {
            slog.Warn("YOLO merge: invalid update index, skipping",
                "index", upd.Index, "max", len(existingSlots)-1)
        }
    }
    for _, delIdx := range action.Delete {
        if delIdx < 0 || delIdx >= len(existingSlots) {
            slog.Warn("YOLO merge: invalid delete index, skipping", "index", delIdx)
        }
    }

    // 5. Build MergeResult
    result := &domain.MergeResult{}

    // Add new slots
    for _, a := range action.Add {
        result.Added = append(result.Added, domain.ProfileSlot{
            ID: uuid.New(), Topic: a.Topic, SubTopic: a.SubTopic, Content: a.Content,
        })
    }

    // Update existing slots (by index)
    for _, u := range action.Update {
        if u.Index < 0 || u.Index >= len(existingSlots) { continue }
        slot := existingSlots[u.Index]
        slot.Topic = u.Topic; slot.SubTopic = u.SubTopic; slot.Content = u.Content
        result.Updated = append(result.Updated, slot)
    }

    // Delete existing slots (by index)
    for _, d := range action.Delete {
        if d < 0 || d >= len(existingSlots) { continue }
        result.Deleted = append(result.Deleted, existingSlots[d].ID)
    }

    return result, nil
}
```

### 2. `internal/usecase/organize_profile.go` — No-LLM Constraints

```go
type OrganizeProfileUseCase struct{}

func (uc *OrganizeProfileUseCase) Execute(
    mr *domain.MergeResult,
    config port.ProjectProfileConfig,
) *domain.MergeResult {
    // Merge Add + Updated into a combined view per topic
    // Group existing + added by topic, sort by updated_at DESC
    // If topic has > MaxSubtopics: remove oldest (add to Deleted)

    // 1. Build map[topic][]ProfileSlot including new additions
    byTopic := make(map[string][]domain.ProfileSlot)
    for _, s := range mr.Added   { byTopic[s.Topic] = append(byTopic[s.Topic], s) }
    for _, s := range mr.Updated { byTopic[s.Topic] = append(byTopic[s.Topic], s) }

    // 2. Enforce MaxSubtopics per topic
    maxSubtopics := config.MaxSubtopics
    if maxSubtopics <= 0 { maxSubtopics = 15 }

    for topic, slots := range byTopic {
        if len(slots) > maxSubtopics {
            // Sort by UpdatedAt DESC, keep newest maxSubtopics
            sort.Slice(slots, func(i, j int) bool {
                return slots[i].UpdatedAt.After(slots[j].UpdatedAt)
            })
            toDelete := slots[maxSubtopics:]
            for _, s := range toDelete {
                mr.Deleted = append(mr.Deleted, s.ID)
            }
            byTopic[topic] = slots[:maxSubtopics]
        }
    }

    return mr
}
```

### 3. `internal/usecase/re_summary.go` — Slot Content Truncation

```go
type ReSummaryUseCase struct {
    llm        llm.LLMClient
    tokenizer  tokenizer.Tokenizer
    maxTokens  int  // default: 128 (MaxSlotTokenSize)
}

func (uc *ReSummaryUseCase) Execute(ctx context.Context, slot domain.ProfileSlot) (domain.ProfileSlot, error) {
    count := uc.tokenizer.Count(slot.Content)
    if count <= uc.maxTokens {
        return slot, nil  // No need to summarize
    }

    promptStr := fmt.Sprintf(
        `Summarize the following profile information to under %d tokens while preserving all key facts:

%s

Output only the summarized text, no JSON.`, uc.maxTokens, slot.Content)

    summarized, err := uc.llm.Complete(ctx, promptStr, llm.WithMaxTokens(uc.maxTokens))
    if err != nil { return slot, err }  // Return original on failure

    slot.Content = summarized
    return slot, nil
}
```

### 4. `internal/usecase/validate_profile.go` — Strict Mode

```go
type ValidateProfileUseCase struct {
    llm llm.LLMClient
}

func (uc *ValidateProfileUseCase) Execute(
    ctx context.Context,
    mr *domain.MergeResult,
    config port.ProjectProfileConfig,
) (*domain.MergeResult, error) {
    if !config.StrictMode {
        return mr, nil
    }

    // Filter: only allow topics in AdditionalTopics + built-in topics
    allowed := buildAllowedTopics(config)

    var filteredAdd []domain.ProfileSlot
    for _, s := range mr.Added {
        if allowed.Contains(s.Topic) {
            filteredAdd = append(filteredAdd, s)
        } else {
            slog.Debug("strict mode: filtered out slot", "topic", s.Topic)
        }
    }
    mr.Added = filteredAdd

    var filteredUpdate []domain.ProfileSlot
    for _, s := range mr.Updated {
        if allowed.Contains(s.Topic) { filteredUpdate = append(filteredUpdate, s) }
    }
    mr.Updated = filteredUpdate

    return mr, nil
}

var builtinTopics = []string{"basic_info", "work", "lifestyle", "interest", "relationship", "preference", "skill", "goal", "other"}

func buildAllowedTopics(config port.ProjectProfileConfig) stringSet {
    set := newStringSet(builtinTopics...)
    for _, t := range config.AdditionalTopics { set.Add(t) }
    return set
}
```

### 5. `internal/adapter/event/subscriber.go` — NATS Consumer

```go
type Subscriber struct {
    processUC   *usecase.ProcessBlobsUseCase
    bulkhead    *resilience.Bulkhead
    js          nats.JetStreamContext
}

func (s *Subscriber) Start(ctx context.Context) error {
    _, err := s.js.Subscribe("memobase.buffer.ready",
        s.handleBufferReady,
        nats.Durable("memobase-engine"),
        nats.MaxDeliver(3),             // retry 3 times on failure
        nats.AckWait(120*time.Second),  // LLM calls can be slow
        nats.AckExplicit(),
    )
    return err
}

func (s *Subscriber) handleBufferReady(msg *nats.Msg) {
    var payload struct {
        UserID    string   `json:"user_id"`
        ProjectID string   `json:"project_id"`
        BufferIDs []string `json:"buffer_ids"`
        BlobType  string   `json:"blob_type"`
    }

    if err := json.Unmarshal(msg.Data, &payload); err != nil {
        slog.Error("engine: invalid buffer.ready payload", "error", err)
        msg.Ack()  // Don't retry bad messages
        return
    }

    userID, _ := uuid.Parse(payload.UserID)
    bufferIDs  := stringsToUUIDs(payload.BufferIDs)

    // Use bulkhead to limit concurrent processing
    err := s.bulkhead.Execute(context.Background(), func() error {
        return s.processUC.Execute(context.Background(), usecase.ProcessBlobsRequest{
            UserID:    userID,
            ProjectID: payload.ProjectID,
            BufferIDs: bufferIDs,
            BlobType:  domain.BlobType(payload.BlobType),
        })
    })

    if err != nil {
        slog.Warn("engine: process failed, will redeliver", "user_id", payload.UserID, "error", err)
        msg.Nak()  // NATS will redeliver (up to MaxDeliver=3 times)
        return
    }

    msg.Ack()
}
```

### 6. `internal/adapter/event/publisher.go` — NATS Publisher

```go
type NATSPublisher struct {
    js nats.JetStreamContext
}

type EngineCompletedPayload struct {
    UserID    string   `json:"user_id"`
    ProjectID string   `json:"project_id"`
    BufferIDs []string `json:"buffer_ids"`
}

type ProfileChangedPayload struct {
    UserID    string `json:"user_id"`
    ProjectID string `json:"project_id"`
}

type EventCreatedPayload struct {
    EventID   string    `json:"event_id"`
    UserID    string    `json:"user_id"`
    ProjectID string    `json:"project_id"`
    EventTip  string    `json:"event_tip"`
    CreatedAt time.Time `json:"created_at"`
}

func (p *NATSPublisher) PublishEngineCompleted(ctx context.Context, payload EngineCompletedPayload) error
// Subject: "memobase.engine.completed"

func (p *NATSPublisher) PublishProfileChanged(ctx context.Context, payload ProfileChangedPayload) error
// Subject: "memobase.engine.profile.changed"

func (p *NATSPublisher) PublishEventCreated(ctx context.Context, event *domain.UserEvent) error
// Subject: "memobase.engine.event.created"
```

### 7. `internal/adapter/grpc/handler.go`

```go
type EngineHandler struct {
    enginev1.UnimplementedEngineServiceServer
    processUC *usecase.ProcessBlobsUseCase
}

// ProcessBlobs: called by gateway in sync mode (rare; normally via NATS)
func (h *EngineHandler) ProcessBlobs(ctx context.Context, req *enginev1.ProcessBlobsRequest) (*enginev1.ProcessBlobsResponse, error)
```

### 8. Config

```yaml
engine:
  server:
    grpc_port: 9042
    health_port: 9092
  llm:
    provider: "bifrost"
    api_key: "${MEMOBASE_LLM_API_KEY}"
    base_url: "${MEMOBASE_LLM_BASE_URL}"
    model: "gpt-4o-mini"           # MEMOBASE_BEST_LLM_MODEL
    max_tokens: 1024
    max_process_token_size: 16384  # MEMOBASE_MAX_PROCESS_TOKEN_SIZE
  embedding:
    provider: "jina"               # MEMOBASE_EMBEDDING_PROVIDER
    api_key: "${MEMOBASE_EMBEDDING_API_KEY}"
    model: "jina-embeddings-v3"
    dimension: 1024
    enabled: true                  # MEMOBASE_ENABLE_EVENT_EMBEDDING
  profile:
    max_subtopics: 15              # MEMOBASE_MAX_PROFILE_SUBTOPICS
    max_slot_token_size: 128
    strict_mode: false
    validate_mode: true
    language: "en"                 # MEMOBASE_LANGUAGE
  nats:
    url: "${NATS_URL}"
    stream: "memobase"
    consumer_group: "engine"
    max_concurrent: 10             # bulkhead semaphore size
  database:
    url: "${DATABASE_URL}"
    pool_size: 20
  services:
    ingestion: "memobase-ingestion:9041"
    admin: "memobase-admin:9045"
```

---

## Unit Tests

```
TestMergeProfileUseCase_AddAction          → LLM returns {"add":[...]} → Added populated
TestMergeProfileUseCase_UpdateAction       → {"update":[{"index":0,...}]} → Updates slot[0]
TestMergeProfileUseCase_DeleteAction       → {"delete":[1]} → Deletes slot[1]
TestMergeProfileUseCase_InvalidUpdateIndex → index=99 (out of range) → skipped with warning
TestMergeProfileUseCase_InvalidDeleteIndex → index=-1 → skipped
TestMergeProfileUseCase_LLMParseFailed     → LLM returns "I'll update..." → ErrLLMParseFailed
TestMergeProfileUseCase_EmptyExisting      → no existing slots → only Add actions
TestOrganizeProfile_BelowMaxSubtopics      → 5 slots, max=15 → no deletion
TestOrganizeProfile_ExceedsMaxSubtopics    → 20 slots, max=15 → 5 oldest deleted
TestOrganizeProfile_PerTopicBucket         → topic A: 10, topic B: 10, max=8 → 4 deleted each
TestReSummary_UnderLimit                   → 50 tokens, max=128 → original returned, LLM not called
TestReSummary_OverLimit                    → 200 tokens, max=128 → LLM called, summarized
TestReSummary_LLMFail_ReturnOriginal       → LLM error → original slot returned (graceful)
TestValidateProfile_StrictMode_FiltersOut  → topic "custom" not in allowed → filtered
TestValidateProfile_StrictMode_AllowsBuiltin → "work" → kept
TestValidateProfile_NonStrictMode          → strict=false → all slots kept
TestValidateProfile_AdditionalTopics       → "sports" in AdditionalTopics → kept
TestNATSHandleBufferReady_ProcessesBlobs   → valid payload → processUC.Execute called
TestNATSHandleBufferReady_InvalidJSON      → bad payload → Ack (don't retry)
TestNATSHandleBufferReady_ProcessFails     → Execute error → Nak (retry)
TestNATSHandleBufferReady_BulkheadBlocks   → max=2, 3 messages → 1 waits
TestNATSPublisher_EngineCompleted          → publish → correct NATS subject
TestNATSPublisher_ProfileChanged           → publish → "memobase.engine.profile.changed"
```

---

## Monolith Bootstrap Integration

**File: `apps/memory/internal/bootstrap/memobase_engine.go`**

```go
func bootstrapMemobaseEngine(ctx context.Context, cfg *config.Config, registry *bus.InProcessRegistry) error {
    db := infra.NewPostgresPool(cfg.Engine.Database)
    js := infra.GetNATSJetStream(ctx)

    tok, _ := tokenizer.New(cfg.Engine.LLM.Model)
    llmClient := llm.NewBifrostClient(cfg.Engine.LLM.BaseURL, cfg.Engine.LLM.APIKey, cfg.Engine.LLM.Model)
    resiLLM := resilience.NewResilientLLMClient(llmClient,
        resilience.NewCircuitBreaker("engine-llm"),
        resilience.NewBulkhead(cfg.Engine.NATS.MaxConcurrent))

    embedderClient := embedder.NewJinaEmbedder(cfg.Engine.Embedding.APIKey, cfg.Engine.Embedding.Model, cfg.Engine.Embedding.Dimension)
    if !cfg.Engine.Embedding.Enabled { embedderClient = &embedder.DisabledEmbedder{} }

    prompts := prompt.NewRegistry()
    profileRepo := postgres.NewProfileRepository(db)
    eventRepo   := postgres.NewEventRepository(db)
    publisher   := event.NewNATSPublisher(js)

    ingestionConn, _ := registry.Dial("memobase-ingestion")
    adminConn, _     := registry.Dial("memobase-admin")
    ingestionClient := client.NewIngestionClient(ingestionConn)
    adminClient     := client.NewAdminClient(adminConn)

    processUC := usecase.NewProcessBlobsUseCase(/* all deps */)
    bulkhead   := resilience.NewBulkhead(cfg.Engine.NATS.MaxConcurrent)
    subscriber := event.NewSubscriber(processUC, bulkhead, js)
    subscriber.Start(ctx)

    handler := grpchandler.New(processUC)
    server  := grpc.NewServer()
    enginev1.RegisterEngineServiceServer(server, handler)
    registry.Register("memobase-engine", server, bufconn.Listen(1024*1024))
    return nil
}
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
buf generate services/memobase-engine/
go build ./services/memobase-engine/...
go test ./services/memobase-engine/... -v -count=1 -race
```

---

## Ghi chú triển khai

- YOLO index validation là critical — LLM có thể return invalid indices → log + skip, không panic
- Bulkhead size = `engine.nats.max_concurrent` (default 10) — limit simultaneous LLM calls
- `msg.Nak()`: NATS redeliver với exponential backoff, max 3 times (MaxDeliver=3)
- `AckWait(120s)`: LLM calls có thể mất 60-90s → cần ít nhất 120s window
- Event goroutine failure: return `nil` explicitly để không cancel profile goroutine
