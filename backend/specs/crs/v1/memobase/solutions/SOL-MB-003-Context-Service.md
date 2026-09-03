# Solution: SOL-MB-003 — Context Service (Profile Read & Context Assembly)

**CR:** [CR-MB-003](../CR-MB-003-Context-Service.md)  
**Wave:** 3 (Read Path)  
**Priority:** Critical  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/memobase-context` — read-path service với **latency target < 100ms P99**. Đây là service được gọi nhiều nhất trong hệ thống (mỗi LLM inference → 1 context call).

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| Không có Redis caching | Redis cache với key `profiles::{project_id}::{user_id}`, TTL 1200s |
| Cache invalidation | Subscribe `memobase.profile.changed` → immediate invalidate |
| Không có context assembly | `AssembleContext()` — format prompt-ready string từ profiles + events |
| Không có token budget | `TruncateProfiles()` với `profile_event_ratio` split |
| Không có parallel fetch | `errgroup.WithContext` — profiles + events đồng thời |
| Manual profile CRUD | Add/Update/Delete profile slots → immediate cache invalidation |

---

## 2. Latency Budget Analysis

```
Target: < 100ms P99 (excluding embedding API call)

Breakdown:
  Redis GET (cache hit):    ~1-3ms
  Redis GET (cache miss) + PG SELECT: ~5-15ms
  Event service gRPC (memobase-event): ~5-20ms (parallel với profile)
  Context assembly (CPU-bound):        ~1-2ms
  Tokenizer count:                     ~1ms
  ─────────────────────────────────────────
  Cache HIT path:  3 + 20 + 2 + 1 ≈ 26ms  ✅
  Cache MISS path: 15 + 20 + 2 + 1 ≈ 38ms ✅
  
  Note: Embedding call (nếu có chats) được gọi TRƯỚC parallel fetch,
  nhưng latency budget tính KHÔNG bao gồm embedding.
```

---

## 3. Vị trí trong Codebase

```
vnp-memory/
└── services/
    └── memobase-context/              ← [NEW]
        ├── cmd/server/main.go
        ├── internal/
        │   ├── domain/
        │   │   ├── entity.go          # Profile, TruncationConfig, ContextResult
        │   │   └── errors.go
        │   ├── usecase/
        │   │   ├── get_context.go     # Orchestrator (parallel fetch + assemble)
        │   │   ├── get_profiles.go    # Cache-first profile retrieval
        │   │   ├── truncate_profiles.go
        │   │   ├── assemble_context.go
        │   │   ├── add_profile.go
        │   │   ├── update_profile.go
        │   │   ├── delete_profile.go
        │   │   └── port/
        │   ├── adapter/
        │   │   ├── grpc/handler.go
        │   │   ├── repository/
        │   │   │   ├── postgres/profile_repo.go  # read-only PG access
        │   │   │   └── redis/profile_cache.go    # Redis cache
        │   │   ├── client/
        │   │   │   └── event_client.go           # gRPC → memobase-event
        │   │   └── event/
        │   │       └── subscriber.go             # profile.changed → invalidate
        │   └── infra/
```

---

## 4. Redis Cache Design

### 4.1 Cache Key Strategy

```
Key format: "profiles::{project_id}::{user_id}"
Value: JSON-encoded []Profile
TTL: 1200s (20 phút) — configurable via MEMOBASE_CACHE_USER_PROFILES_TTL

Ví dụ:
  Key:   "profiles::proj-abc::550e8400-e29b-41d4-a716-446655440000"
  Value: [{"id":"...","content":"Alice","attributes":{"topic":"basic_info","sub_topic":"name"}},...]
  TTL:   1200s
```

### 4.2 Cache Invalidation Triggers

```go
// adapter/repository/redis/profile_cache.go

func (c *ProfileCache) Invalidate(ctx context.Context, projectID, userID string) error {
    key := fmt.Sprintf("profiles::%s::%s", projectID, userID)
    return c.redis.Del(ctx, key).Err()
}

// Triggers:
// 1. NATS memobase.profile.changed → subscriber.go → Invalidate()
// 2. NATS memobase.admin.user.deleted → subscriber.go → Invalidate()
// 3. POST /api/v1/users/profile/{user_id} → AddProfile → Invalidate()
// 4. PUT /api/v1/users/profile/{user_id}/{profile_id} → UpdateProfile → Invalidate()
// 5. DELETE /api/v1/users/profile/{user_id}/{profile_id} → DeleteProfile → Invalidate()
```

### 4.3 Cache-First Get Pattern

```go
// usecase/get_profiles.go

func (uc *GetProfilesUseCase) Execute(ctx context.Context, req GetProfilesRequest) ([]Profile, error) {
    // 1. Try Redis cache
    cached, found, err := uc.cache.Get(ctx, req.ProjectID, req.UserID)
    if err == nil && found {
        return cached, nil  // Cache HIT → sub-5ms
    }

    // 2. Cache MISS → query PostgreSQL
    profiles, err := uc.profileRepo.GetByUser(ctx, req.UserID, req.ProjectID)
    if err != nil {
        return nil, err
    }

    // 3. Warm cache (fire-and-forget, don't block response)
    go func() {
        warmCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        uc.cache.Set(warmCtx, req.ProjectID, req.UserID, profiles, uc.config.ProfileTTL)
    }()

    return profiles, nil
}
```

---

## 5. Context Assembly Algorithm

### 5.1 Profile Truncation

```go
// usecase/truncate_profiles.go

func TruncateProfiles(profiles []Profile, config TruncationConfig) []Profile {
    // Step 1: Sort by updated_at DESC (most recent first)
    sort.Slice(profiles, func(i, j int) bool {
        return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt)
    })

    // Step 2: filter only_topics
    if len(config.OnlyTopics) > 0 {
        profiles = filterByTopics(profiles, config.OnlyTopics)
    }

    // Step 3: Prioritize prefer_topics (move to front, keep order)
    if len(config.PreferTopics) > 0 {
        var preferred, rest []Profile
        for _, p := range profiles {
            if slices.Contains(config.PreferTopics, p.Attributes.Topic) {
                preferred = append(preferred, p)
            } else {
                rest = append(rest, p)
            }
        }
        profiles = append(preferred, rest...)
    }

    // Step 4: Per-topic subtopic limits
    if len(config.TopicLimits) > 0 || config.MaxSubtopicSize > 0 {
        profiles = applyTopicLimits(profiles, config.TopicLimits, config.MaxSubtopicSize)
    }

    // Step 5: Token budget enforcement
    var result []Profile
    tokenCount := 0
    for _, p := range profiles {
        line := fmt.Sprintf("- %s::%s: %s", p.Attributes.Topic, p.Attributes.SubTopic, p.Content)
        tokens := tokenizer.Count(line)
        if tokenCount+tokens > config.MaxTokenSize {
            break
        }
        tokenCount += tokens
        result = append(result, p)
    }
    return result
}
```

### 5.2 Context String Output

```go
// usecase/assemble_context.go

func AssembleContext(profiles []Profile, events []EventGist, template string) string {
    if template != "" {
        return assembleWithTemplate(profiles, events, template)
    }
    // Default format:
    var sb strings.Builder
    sb.WriteString("# Memory\n")
    sb.WriteString("Unless the user has relevant queries, do not actively mention those memories.\n")

    if len(profiles) > 0 {
        sb.WriteString("## User Background:\n")
        for _, p := range profiles {
            sb.WriteString(fmt.Sprintf("- %s::%s: %s\n", p.Attributes.Topic, p.Attributes.SubTopic, p.Content))
        }
    }

    if len(events) > 0 {
        sb.WriteString("\n## Latest Events:\n")
        for _, e := range events {
            sb.WriteString(fmt.Sprintf("- %s\n", e.GistContent))
        }
    }

    return sb.String()
}

func assembleWithTemplate(profiles []Profile, events []EventGist, tmpl string) string {
    profileSection := formatProfiles(profiles)
    eventSection := formatEvents(events)
    result := strings.ReplaceAll(tmpl, "{profile_section}", profileSection)
    result = strings.ReplaceAll(result, "{event_section}", eventSection)
    return result
}
```

---

## 6. GetContext Use Case — Parallel Fetch

```go
// usecase/get_context.go

func (uc *GetContextUseCase) Execute(ctx context.Context, req ContextRequest) (*ContextResult, error) {
    // Pre-compute query embedding nếu có chats (sequential — cần trước parallel)
    var queryEmbedding []float32
    if len(req.Chats) > 0 && uc.embedder.IsEnabled() {
        latestMsg := extractLatestUserMessage(req.Chats)
        queryEmbedding, _ = uc.embedder.EmbedQuery(ctx, latestMsg)
    }

    // Parallel fetch
    var (
        profiles []Profile
        events   []EventGist
    )
    g, gCtx := errgroup.WithContext(ctx)

    g.Go(func() error {
        var err error
        profiles, err = uc.getProfiles.Execute(gCtx, GetProfilesRequest{
            UserID:    req.UserID,
            ProjectID: req.ProjectID,
        })
        return err
    })

    g.Go(func() error {
        var err error
        if queryEmbedding != nil {
            // Semantic event search
            events, err = uc.eventClient.SearchEventGists(gCtx, SearchGistsRequest{
                UserID:              req.UserID,
                ProjectID:           req.ProjectID,
                Embedding:           queryEmbedding,
                TopK:                req.EventTopK,
                TimeRangeDays:       req.TimeRangeDays,
                SimilarityThreshold: req.EventSimilarityThreshold,
            })
        } else {
            // Fallback: recent events
            events, err = uc.eventClient.GetRecentEventGists(gCtx, req.UserID, req.ProjectID, req.EventTopK)
        }
        return err
    })

    if err := g.Wait(); err != nil {
        return nil, err
    }

    // Token budget split
    profileTokenBudget := int(float64(req.MaxTokenSize) * req.ProfileEventRatio)
    truncatedProfiles := TruncateProfiles(profiles, TruncationConfig{
        MaxTokenSize:    profileTokenBudget,
        PreferTopics:    req.PreferTopics,
        OnlyTopics:      req.OnlyTopics,
        MaxSubtopicSize: req.MaxSubtopicSize,
        TopicLimits:     req.TopicLimits,
    })

    profileTokensUsed := uc.tokenizer.CountProfiles(truncatedProfiles)
    eventTokenBudget := req.MaxTokenSize - profileTokensUsed
    truncatedEvents := truncateEvents(events, eventTokenBudget, uc.tokenizer)

    contextStr := AssembleContext(truncatedProfiles, truncatedEvents, req.CustomTemplate)

    return &ContextResult{
        ContextString: contextStr,
        ProfileCount:  len(truncatedProfiles),
        EventCount:    len(truncatedEvents),
        TokensUsed:    uc.tokenizer.CountString(contextStr),
    }, nil
}
```

---

## 7. Manual Profile CRUD

```go
// usecase/add_profile.go
func (uc *AddProfileUseCase) Execute(ctx context.Context, req AddProfileRequest) (*Profile, error) {
    profile, err := uc.profileRepo.Save(ctx, Profile{
        UserID:    req.UserID,
        ProjectID: req.ProjectID,
        Content:   req.Content,
        Attributes: ProfileAttributes{
            Topic:    req.Topic,
            SubTopic: req.SubTopic,
        },
    })
    if err != nil {
        return nil, err
    }
    // Immediate cache invalidation
    uc.cache.Invalidate(ctx, req.ProjectID, req.UserID)
    return profile, nil
}

// usecase/update_profile.go — UPDATE user_profiles SET content=..., updated_at=NOW()
// usecase/delete_profile.go — DELETE FROM user_profiles WHERE id=$1 AND project_id=$2
// Cả hai đều gọi cache.Invalidate() sau khi DB operation thành công
```

---

## 8. gRPC Service

```protobuf
syntax = "proto3";
package memobase.context.v1;

service ContextService {
    // Context assembly (primary API)
    rpc GetContext(GetContextRequest) returns (GetContextResponse);

    // Profile CRUD
    rpc GetProfiles(GetProfilesRequest) returns (GetProfilesResponse);
    rpc AddProfile(AddProfileRequest) returns (AddProfileResponse);
    rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileResponse);
    rpc DeleteProfile(DeleteProfileRequest) returns (DeleteProfileResponse);

    // Internal utility
    rpc TruncateProfiles(TruncateProfilesRequest) returns (TruncateProfilesResponse);
}

message GetContextRequest {
    string user_id                    = 1;
    string project_id                 = 2;
    int32  max_token_size             = 3;  // default: 500
    repeated string prefer_topics     = 4;
    repeated string only_topics       = 5;
    double profile_event_ratio        = 6;  // default: 0.7
    repeated ChatMessage chats        = 7;  // latest chats for semantic search
    string customize_context_prompt   = 8;
    double event_similarity_threshold = 9;  // default: 0.2
    int32  time_range_in_days         = 10; // default: 21
    int32  event_topk                 = 11; // default: 5
    map<string, int32> topic_limits   = 12;
    int32  max_subtopic_size          = 13;
}
```

---

## 9. NATS Subscriber

```go
// adapter/event/subscriber.go

func (s *Subscriber) Start(ctx context.Context) {
    // Profile changed → invalidate cache
    s.js.Subscribe("memobase.profile.changed", func(msg *nats.Msg) {
        var p struct{ UserID, ProjectID string }
        json.Unmarshal(msg.Data, &p)
        s.cache.Invalidate(ctx, p.ProjectID, p.UserID)
        msg.Ack()
    }, nats.Durable("memobase-context-profile"))

    // User deleted → delete from cache
    s.js.Subscribe("memobase.admin.user.deleted", func(msg *nats.Msg) {
        var p struct{ UserID, ProjectID string }
        json.Unmarshal(msg.Data, &p)
        s.cache.Invalidate(ctx, p.ProjectID, p.UserID)
        msg.Ack()
    }, nats.Durable("memobase-context-user-deleted"))

    // Project config updated → reload config (không ảnh hưởng cache)
    s.js.Subscribe("memobase.admin.project.updated", func(msg *nats.Msg) {
        var p struct{ ProjectID, ConfigYAML string }
        json.Unmarshal(msg.Data, &p)
        s.configCache.Reload(p.ProjectID, p.ConfigYAML)
        msg.Ack()
    }, nats.Durable("memobase-context-project"))
}
```

---

## 10. Configuration

```yaml
context:
  server:
    grpc_port: 9043
    health_port: 9093

  cache:
    redis_url: "${REDIS_URL}"
    profile_ttl: 1200s                      # MEMOBASE_CACHE_USER_PROFILES_TTL

  defaults:
    max_token_size: 500
    profile_event_ratio: 0.7
    event_time_range_days: 21
    event_topk: 5
    similarity_threshold: 0.2

  embedding:
    provider: "openai"
    model: "text-embedding-3-small"
    enabled: true

  database:
    url: "${DATABASE_URL}"
    pool_size: 30
    max_overflow: 20

  services:
    event:
      address: "memobase-event:9044"
      timeout: 10s
    admin:
      address: "memobase-admin:9045"
      timeout: 5s
```

---

## 11. Testing Strategy

### Unit Tests
- `TestGetProfilesUseCase_CacheHit` — mock Redis → verify DB not called
- `TestGetProfilesUseCase_CacheMiss` — Redis empty → DB called → cache warmed
- `TestTruncateProfiles_TokenBudget` — `max_token_size=100` → output ≤ 100 tokens
- `TestTruncateProfiles_PreferTopics` — prefer_topics=[basic_info] → appears first
- `TestTruncateProfiles_OnlyTopics` — only work profiles returned
- `TestAssembleContext_DefaultFormat` — verify output string format
- `TestAssembleContext_CustomTemplate` — `{profile_section}` replaced correctly
- `TestCacheInvalidation_OnProfileChanged` — NATS event → cache.Invalidate called

### Performance Tests
- `BenchmarkGetContext_CacheHit` — < 5ms
- `BenchmarkGetContext_CacheMiss` — < 50ms

---

## 12. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| Redis unavailable | Trung bình | Graceful degradation: fallback thẳng xuống PostgreSQL |
| Cache stampede (nhiều requests cùng miss) | Thấp | Singleflight pattern hoặc probabilistic early expiration |
| Event service timeout trong parallel fetch | Thấp | `errgroup` với context timeout 10s; trả về profiles mà không có events |
| Token count sai khác so với thực tế | Thấp | tiktoken gpt-4o ≈ 95% accurate; buffer 5% |
| Custom template injection risk | Thấp | Chỉ replace `{profile_section}` / `{event_section}`, không eval code |
