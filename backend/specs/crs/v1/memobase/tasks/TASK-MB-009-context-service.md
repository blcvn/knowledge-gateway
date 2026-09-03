# TASK-MB-009 — `services/memobase-context` Profile Read, Redis Cache & Context Assembly

**Wave:** 3 (Read Path)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-MB-001 (pkg/tokenizer), TASK-MB-005 (pkg/adapters), TASK-MB-007/008 (engine writes profiles)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-MB-003](../solutions/SOL-MB-003-Context-Service.md)  
**Port gRPC:** 9043  
**Latency target:** < 100ms P99 (cache hit path)

**Trạng thái:** 🔄 Partial  
**Ghi chú:** memobase-context: 0 internal .go - service scaffolded, logic missing  
---

## Mục tiêu

Tạo `services/memobase-context/` — hot-path read service với Redis cache, parallel fetch (profiles + events), token-budget truncation, context string assembly, manual profile CRUD, và NATS-driven cache invalidation.

---

## Cấu trúc thư mục

```
services/memobase-context/
├── cmd/server/main.go
├── api/proto/memobase/context/v1/context.proto
├── internal/
│   ├── domain/
│   │   ├── profile.go          # Profile, ProfileAttributes, TruncationConfig, ContextResult
│   │   └── errors.go           # ErrUserNotFound, ErrProfileNotFound
│   ├── usecase/
│   │   ├── get_context.go      # Orchestrator: parallel fetch + assemble
│   │   ├── get_profiles.go     # Cache-first profile retrieval
│   │   ├── truncate_profiles.go
│   │   ├── assemble_context.go
│   │   ├── add_profile.go      # Manual CRUD + cache invalidation
│   │   ├── update_profile.go
│   │   ├── delete_profile.go
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go       # ProfileRepository, ProfileCache, EventClient, EventPublisher
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── postgres/
│   │   │   │   └── profile_repo.go   # Read-only PG access (write owned by engine)
│   │   │   └── redis/
│   │   │       └── profile_cache.go  # Redis cache
│   │   ├── client/
│   │   │   └── event_client.go       # gRPC → memobase-event
│   │   └── event/
│   │       └── subscriber.go         # NATS: profile.changed → invalidate
│   └── infra/
│       ├── config/config.go
│       └── migrations/               # None — reads from engine's tables
```

---

## 1. Domain Models

**File: `internal/domain/profile.go`**

```go
type ProfileAttributes struct {
    Topic    string `json:"topic"`
    SubTopic string `json:"sub_topic"`
}

type Profile struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    ProjectID  string
    Content    string
    Attributes ProfileAttributes
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type TruncationConfig struct {
    MaxTokenSize    int
    PreferTopics    []string
    OnlyTopics      []string
    MaxSubtopicSize int
    TopicLimits     map[string]int  // per-topic max subtopics
}

type ContextResult struct {
    ContextString string
    ProfileCount  int
    EventCount    int
    TokensUsed    int
}
```

---

## 2. Use Cases

### `internal/usecase/get_profiles.go` — Cache-First

```go
type GetProfilesUseCase struct {
    cache       port.ProfileCache
    profileRepo port.ProfileRepository
    config      ContextConfig
}

func (uc *GetProfilesUseCase) Execute(ctx context.Context, req GetProfilesRequest) ([]domain.Profile, error) {
    // 1. Try Redis cache
    cached, found, err := uc.cache.Get(ctx, req.ProjectID, req.UserID)
    if err == nil && found {
        return cached, nil  // Cache HIT (sub-3ms)
    }
    // Graceful: Redis error → fallback to DB (don't fail request)

    // 2. Cache MISS → PostgreSQL
    profiles, err := uc.profileRepo.GetByUser(ctx, req.UserID, req.ProjectID)
    if err != nil { return nil, err }

    // 3. Warm cache (non-blocking)
    go func() {
        warmCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        uc.cache.Set(warmCtx, req.ProjectID, req.UserID, profiles, uc.config.ProfileTTL)
    }()

    return profiles, nil
}
```

### `internal/usecase/truncate_profiles.go`

```go
func TruncateProfiles(profiles []domain.Profile, config domain.TruncationConfig, tok tokenizer.Tokenizer) []domain.Profile {
    // Step 1: Sort by UpdatedAt DESC (freshest first)
    sort.Slice(profiles, func(i, j int) bool {
        return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt)
    })

    // Step 2: filter only_topics
    if len(config.OnlyTopics) > 0 {
        profiles = filterByTopics(profiles, config.OnlyTopics)
    }

    // Step 3: Prioritize prefer_topics (move to front)
    if len(config.PreferTopics) > 0 {
        var preferred, rest []domain.Profile
        for _, p := range profiles {
            if slices.Contains(config.PreferTopics, p.Attributes.Topic) {
                preferred = append(preferred, p)
            } else {
                rest = append(rest, p)
            }
        }
        profiles = append(preferred, rest...)
    }

    // Step 4: per-topic subtopic limits
    if len(config.TopicLimits) > 0 || config.MaxSubtopicSize > 0 {
        profiles = applyTopicLimits(profiles, config.TopicLimits, config.MaxSubtopicSize)
    }

    // Step 5: Token budget enforcement
    var result []domain.Profile
    tokenCount := 0
    for _, p := range profiles {
        line := fmt.Sprintf("- %s::%s: %s", p.Attributes.Topic, p.Attributes.SubTopic, p.Content)
        tokens := tok.Count(line)
        if config.MaxTokenSize > 0 && tokenCount+tokens > config.MaxTokenSize {
            break
        }
        tokenCount += tokens
        result = append(result, p)
    }
    return result
}
```

### `internal/usecase/assemble_context.go`

```go
type EventGist struct {
    GistContent string
}

func AssembleContext(profiles []domain.Profile, events []EventGist, customTemplate string) string {
    if customTemplate != "" {
        profileSection := formatProfileSection(profiles)
        eventSection   := formatEventSection(events)
        result := strings.ReplaceAll(customTemplate, "{profile_section}", profileSection)
        result  = strings.ReplaceAll(result, "{event_section}", eventSection)
        return result
    }

    // Default output format
    var sb strings.Builder
    sb.WriteString("# Memory\n")
    sb.WriteString("Unless the user has relevant queries, do not actively mention those memories.\n")

    if len(profiles) > 0 {
        sb.WriteString("\n## User Background:\n")
        for _, p := range profiles {
            sb.WriteString(fmt.Sprintf("- %s::%s: %s\n",
                p.Attributes.Topic, p.Attributes.SubTopic, p.Content))
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
```

### `internal/usecase/get_context.go` — Parallel Orchestrator

```go
type ContextRequest struct {
    UserID                  uuid.UUID
    ProjectID               string
    MaxTokenSize            int       // default: 500
    PreferTopics            []string
    OnlyTopics              []string
    ProfileEventRatio       float64   // default: 0.7
    Chats                   []ChatMessage  // for semantic search embedding
    CustomTemplate          string
    EventSimilarityThreshold float64  // default: 0.2
    TimeRangeDays           int       // default: 21
    EventTopK               int       // default: 5
    TopicLimits             map[string]int
    MaxSubtopicSize         int
}

func (uc *GetContextUseCase) Execute(ctx context.Context, req ContextRequest) (*domain.ContextResult, error) {
    // 1. Pre-compute embedding (sequential, before parallel)
    var queryEmbedding []float32
    if len(req.Chats) > 0 && uc.embedder.IsEnabled() {
        latestMsg := extractLatestUserMessage(req.Chats)
        queryEmbedding, _ = uc.embedder.EmbedQuery(ctx, latestMsg)
        // Non-fatal: if embedding fails, fallback to recent events
    }

    // 2. Parallel fetch
    var profiles []domain.Profile
    var events   []EventGist

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
            events, err = uc.eventClient.SearchEventGists(gCtx, SearchGistsRequest{
                UserID:              req.UserID,
                ProjectID:           req.ProjectID,
                Embedding:           queryEmbedding,
                TopK:                req.EventTopK,
                TimeRangeDays:       req.TimeRangeDays,
                SimilarityThreshold: req.EventSimilarityThreshold,
            })
        } else {
            events, err = uc.eventClient.GetRecentEventGists(gCtx, req.UserID, req.ProjectID, req.EventTopK)
        }
        if err != nil {
            slog.Warn("get context: event fetch failed, continuing without events", "error", err)
            return nil  // Non-fatal: return profiles without events
        }
        return nil
    })

    if err := g.Wait(); err != nil { return nil, err }

    // 3. Token budget split
    maxTokens := req.MaxTokenSize
    if maxTokens <= 0 { maxTokens = 500 }
    ratio := req.ProfileEventRatio
    if ratio <= 0 { ratio = 0.7 }

    profileBudget := int(float64(maxTokens) * ratio)
    truncated := TruncateProfiles(profiles, domain.TruncationConfig{
        MaxTokenSize:    profileBudget,
        PreferTopics:    req.PreferTopics,
        OnlyTopics:      req.OnlyTopics,
        TopicLimits:     req.TopicLimits,
        MaxSubtopicSize: req.MaxSubtopicSize,
    }, uc.tokenizer)

    // 4. Truncate events to remaining budget
    usedProfileTokens := countProfileTokens(truncated, uc.tokenizer)
    eventBudget := maxTokens - usedProfileTokens
    truncatedEvents := truncateEventsByTokens(events, eventBudget, uc.tokenizer)

    // 5. Assemble context string
    contextStr := AssembleContext(truncated, truncatedEvents, req.CustomTemplate)
    tokensUsed := uc.tokenizer.Count(contextStr)

    return &domain.ContextResult{
        ContextString: contextStr,
        ProfileCount:  len(truncated),
        EventCount:    len(truncatedEvents),
        TokensUsed:    tokensUsed,
    }, nil
}
```

### Manual Profile CRUD

```go
// add_profile.go
func (uc *AddProfileUseCase) Execute(ctx context.Context, req AddProfileRequest) (*domain.Profile, error) {
    profile, err := uc.profileRepo.Save(ctx, &domain.Profile{
        ID: uuid.New(), UserID: req.UserID, ProjectID: req.ProjectID,
        Content: req.Content, Attributes: domain.ProfileAttributes{Topic: req.Topic, SubTopic: req.SubTopic},
    })
    if err != nil { return nil, err }
    uc.cache.Invalidate(ctx, req.ProjectID, req.UserID.String())
    return profile, nil
}

// update_profile.go
func (uc *UpdateProfileUseCase) Execute(ctx context.Context, req UpdateProfileRequest) (*domain.Profile, error) {
    // UPDATE user_profiles SET content=$1, updated_at=NOW() WHERE id=$2 AND project_id=$3
    updated, err := uc.profileRepo.Update(ctx, req)
    if err != nil { return nil, err }
    uc.cache.Invalidate(ctx, req.ProjectID, req.UserID.String())
    return updated, nil
}

// delete_profile.go
func (uc *DeleteProfileUseCase) Execute(ctx context.Context, req DeleteProfileRequest) error {
    if err := uc.profileRepo.Delete(ctx, req.ProfileID, req.ProjectID); err != nil { return err }
    uc.cache.Invalidate(ctx, req.ProjectID, req.UserID.String())
    return nil
}
```

---

## 3. Redis Cache Adapter

**File: `internal/adapter/repository/redis/profile_cache.go`**

```go
// Key format: "profiles::{project_id}::{user_id}"
// TTL: 1200s (configurable)

type RedisProfileCache struct {
    client *redis.Client
    ttl    time.Duration
}

func (c *RedisProfileCache) Get(ctx context.Context, projectID, userID string) ([]domain.Profile, bool, error) {
    key := fmt.Sprintf("profiles::%s::%s", projectID, userID)
    data, err := c.client.Get(ctx, key).Bytes()
    if err == redis.Nil { return nil, false, nil }  // Cache miss
    if err != nil { return nil, false, err }         // Redis error

    var profiles []domain.Profile
    if err := json.Unmarshal(data, &profiles); err != nil { return nil, false, err }
    return profiles, true, nil
}

func (c *RedisProfileCache) Set(ctx context.Context, projectID, userID string, profiles []domain.Profile, ttl time.Duration) error {
    key := fmt.Sprintf("profiles::%s::%s", projectID, userID)
    data, err := json.Marshal(profiles)
    if err != nil { return err }
    return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisProfileCache) Invalidate(ctx context.Context, projectID, userID string) error {
    key := fmt.Sprintf("profiles::%s::%s", projectID, userID)
    return c.client.Del(ctx, key).Err()
}
```

---

## 4. NATS Subscriber

**File: `internal/adapter/event/subscriber.go`**

```go
func (s *Subscriber) Start(ctx context.Context) {
    // Profile changed (by engine) → invalidate cache immediately
    s.js.Subscribe("memobase.engine.profile.changed", func(msg *nats.Msg) {
        var p struct{ UserID, ProjectID string }
        json.Unmarshal(msg.Data, &p)
        s.cache.Invalidate(ctx, p.ProjectID, p.UserID)
        msg.Ack()
    }, nats.Durable("memobase-context-profile-changed"))

    // User deleted → invalidate cache
    s.js.Subscribe("memobase.admin.user.deleted", func(msg *nats.Msg) {
        var p struct{ UserID, ProjectID string }
        json.Unmarshal(msg.Data, &p)
        s.cache.Invalidate(ctx, p.ProjectID, p.UserID)
        msg.Ack()
    }, nats.Durable("memobase-context-user-deleted"))

    // Project config updated → reload in-memory config cache
    s.js.Subscribe("memobase.admin.project.updated", func(msg *nats.Msg) {
        var p struct{ ProjectID, ConfigYAML string }
        json.Unmarshal(msg.Data, &p)
        s.configCache.Reload(p.ProjectID, p.ConfigYAML)
        msg.Ack()
    }, nats.Durable("memobase-context-project-updated"))
}
```

---

## 5. gRPC Proto

**File: `api/proto/memobase/context/v1/context.proto`**

```protobuf
syntax = "proto3";
package memobase.context.v1;
option go_package = "vnp-memory/services/memobase-context/api/gen/context/v1;contextv1";

service ContextService {
  rpc GetContext(GetContextRequest) returns (GetContextResponse);
  rpc GetProfiles(GetProfilesRequest) returns (GetProfilesResponse);
  rpc AddProfile(AddProfileRequest) returns (AddProfileResponse);
  rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileResponse);
  rpc DeleteProfile(DeleteProfileRequest) returns (DeleteProfileResponse);
  rpc TruncateProfiles(TruncateProfilesRequest) returns (TruncateProfilesResponse);
}

message GetContextRequest {
  string user_id                    = 1;
  string project_id                 = 2;
  int32  max_token_size             = 3;   // default: 500
  repeated string prefer_topics     = 4;
  repeated string only_topics       = 5;
  double profile_event_ratio        = 6;   // default: 0.7
  repeated ChatMessage chats        = 7;   // for semantic event search
  string customize_context_prompt   = 8;   // template with {profile_section}/{event_section}
  double event_similarity_threshold = 9;   // default: 0.2
  int32  time_range_in_days         = 10;  // default: 21
  int32  event_topk                 = 11;  // default: 5
  map<string, int32> topic_limits   = 12;
  int32  max_subtopic_size          = 13;
}

message GetContextResponse {
  string context_str    = 1;
  int32  profile_count  = 2;
  int32  event_count    = 3;
  int32  tokens_used    = 4;
}

message AddProfileRequest {
  string user_id    = 1;
  string project_id = 2;
  string content    = 3;
  string topic      = 4;
  string sub_topic  = 5;
}
```

---

## 6. Config

```yaml
context:
  server:
    grpc_port: 9043
    health_port: 9093
  cache:
    redis_url: "${REDIS_URL}"
    profile_ttl: 1200s    # MEMOBASE_CACHE_USER_PROFILES_TTL
  defaults:
    max_token_size: 500
    profile_event_ratio: 0.7
    event_time_range_days: 21
    event_topk: 5
    similarity_threshold: 0.2
  embedding:
    provider: "openai"
    model: "text-embedding-3-small"
    dimension: 1536
    enabled: true
  database:
    url: "${DATABASE_URL}"
    pool_size: 30
    max_overflow: 20
  nats:
    url: "${NATS_URL}"
    stream: "memobase"
  services:
    event: "memobase-event:9044"
    admin: "memobase-admin:9045"
```

---

## Unit Tests

```
TestGetProfilesUseCase_CacheHit            → Redis returns data → DB NOT called
TestGetProfilesUseCase_CacheMiss           → Redis empty → DB called → cache warmed async
TestGetProfilesUseCase_RedisError_Fallback → Redis error → DB called (graceful degradation)
TestTruncateProfiles_TokenBudget           → max=100 tokens → result ≤ 100 tokens
TestTruncateProfiles_SortedByUpdatedAt     → newer profiles first in output
TestTruncateProfiles_PreferTopics          → prefer=["work"] → work profiles first
TestTruncateProfiles_OnlyTopics            → only=["basic_info"] → only basic_info returned
TestTruncateProfiles_TopicLimits           → limit work=2, 5 work slots → 2 kept
TestTruncateProfiles_EmptyProfiles         → [] → []
TestAssembleContext_DefaultFormat          → includes "# Memory" header
TestAssembleContext_ProfileSection         → profile lines with "topic::sub: content"
TestAssembleContext_EventSection           → event lines with "- gist"
TestAssembleContext_EmptyProfiles          → no "## User Background" section
TestAssembleContext_CustomTemplate         → {profile_section} replaced
TestGetContextUseCase_ParallelFetch        → profiles + events fetched concurrently (errgroup)
TestGetContextUseCase_TokenBudgetSplit     → 70% profiles, 30% events (ratio=0.7)
TestGetContextUseCase_EventFetchFails      → event error → profiles returned without events
TestGetContextUseCase_NoChats_RecentEvents → no chats → GetRecentEventGists called
TestGetContextUseCase_WithChats_Semantic   → chats → EmbedQuery → SearchEventGists
TestAddProfile_InvalidatesCache            → save → cache.Invalidate called
TestUpdateProfile_InvalidatesCache         → update → cache.Invalidate called
TestDeleteProfile_InvalidatesCache         → delete → cache.Invalidate called
TestRedisCache_Get_Hit                     → set then get → same data
TestRedisCache_Get_Miss                    → empty → found=false
TestRedisCache_Invalidate                  → set then invalidate → next get = miss
TestRedisCache_TTL                         → set with 1s TTL → after 1s → miss
TestNATSSubscriber_ProfileChanged          → NATS event → cache.Invalidate called
TestNATSSubscriber_UserDeleted             → NATS event → cache.Invalidate called
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
buf generate services/memobase-context/
go build ./services/memobase-context/...
go test ./services/memobase-context/... -v -count=1 -race

# Benchmark (optional)
go test ./services/memobase-context/... -bench=BenchmarkGetContext -benchmem
```

---

## Ghi chú triển khai

- **Cache miss path**: khi Redis down → fallback to PG, không trả error → graceful degradation
- **Event fetch failure**: trả về profiles + empty events (không fail request) — event là "nice to have"
- **Profile write**: KHÔNG trong memobase-context (context chỉ đọc từ engine tables); manual CRUD là exception thông qua `profileRepo.Write()`
- **Profile repo** cần write access chỉ cho manual CRUD — dùng cùng PG instance với engine
- `countProfileTokens`: dùng format string giống `AssembleContext` để đếm token chính xác
- Redis `go-redis/v9`: `client.Get(ctx, key).Bytes()` trả `redis.Nil` khi key không tồn tại
