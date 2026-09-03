# Change Request: CR-MB-003 — Context Service (Profile Read & Context Assembly)

**CR ID:** CR-MB-003  
**Component:** `services/memobase-context` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** memobase PRD §5.5 (F-5), SRS §3.8, specs/services/04-memobase-context.md  
**Maps to Python:** `controllers/context.py`, `controllers/profile.py` (read path)

---

## 1. Mô tả

Xây dựng **memobase-context** service — read-path service với **< 100ms P99 latency**:
1. **Profile CRUD** (get/add/update/delete user profiles với Redis caching).
2. **Context Assembly** — tạo prompt-ready string từ profiles + events với token budget control.
3. **Profile Truncation** — thông minh cắt profiles theo `prefer_topics`, `only_topics`, `topic_limits`.
4. **Custom Template** — developer customize context output format.
5. **Semantic Event Integration** — parallel fetch events liên quan tới conversation hiện tại.

---

## 2. Vấn đề hiện tại

VNP Memory hiện tại:
- ✅ Có basic profile read.
- ❌ Không có **Redis profile caching** với TTL 20 phút + NATS invalidation.
- ❌ Không có **Context Assembly** (prompt-ready string từ profiles + events).
- ❌ Không có **token budget truncation** (`max_token_size`, `profile_event_ratio`).
- ❌ Không có **topic priority/filtering** (`prefer_topics`, `only_topics`, `topic_limits`).
- ❌ Không có **custom context template** (`customize_context_prompt`).
- ❌ Không có **parallel fetch** (profiles + events concurrently via errgroup).
- ❌ Không có **manual profile CRUD** (add/update/delete individual profiles).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/memobase-context/`

**Port:** `9043` (gRPC internal), **Health:** `9093`

### 3.2. Domain Models

```go
// internal/domain/entity.go

type Profile struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    ProjectID  string
    Content    string
    Attributes ProfileAttributes  // topic + sub_topic
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type ProfileAttributes struct {
    Topic    string `json:"topic"`
    SubTopic string `json:"sub_topic"`
}

type TruncationConfig struct {
    MaxTokenSize       int
    ProfileEventRatio  float64         // profiles token share (default 0.7)
    PreferTopics       []string
    OnlyTopics         []string
    MaxSubtopicSize    int
    TopicLimits        map[string]int  // per-topic max profiles
    TopK               int
}

type ContextResult struct {
    ContextString string
    ProfileCount  int
    EventCount    int
    TokensUsed    int
}
```

### 3.3. GetContext Use Case (Parallel, < 100ms)

```go
// internal/usecase/get_context.go

func (uc *GetContextUseCase) Execute(ctx context.Context, req ContextRequest) (*ContextResult, error) {
    // --- Parallel fetch (errgroup) ---
    var (
        profiles []Profile
        events   []EventGist
    )

    g, ctx := errgroup.WithContext(ctx)

    // goroutine 1: Get Profiles (Redis cache-first)
    g.Go(func() error {
        profiles, _ = uc.getProfiles.Execute(ctx, GetProfilesRequest{
            UserID:    req.UserID,
            ProjectID: req.ProjectID,
        })
        return nil
    })

    // goroutine 2: Get Events (semantic search or recent)
    g.Go(func() error {
        if len(req.Chats) > 0 {
            // Semantic search: embed latest chats, find relevant events
            events, _ = uc.eventClient.SearchEventGists(ctx, SearchGistsRequest{
                UserID:             req.UserID,
                ProjectID:          req.ProjectID,
                QueryChats:         req.Chats,
                TopK:               req.EventTopK,
                TimeRangeDays:      req.TimeRangeDays,
                SimilarityThreshold: req.EventSimilarityThreshold,
            })
        } else {
            // Fallback: recent events by time
            events, _ = uc.eventClient.GetRecentEventGists(ctx, req.UserID, req.ProjectID, req.EventTopK)
        }
        return nil
    })

    g.Wait()

    // --- Truncation ---
    // Profile token budget = max_token_size * profile_event_ratio
    profileTokenBudget := int(float64(req.MaxTokenSize) * req.ProfileEventRatio)
    truncatedProfiles := uc.truncateProfiles.Execute(profiles, TruncationConfig{
        MaxTokenSize:      profileTokenBudget,
        PreferTopics:      req.PreferTopics,
        OnlyTopics:        req.OnlyTopics,
        MaxSubtopicSize:   req.MaxSubtopicSize,
        TopicLimits:       req.TopicLimits,
    })

    // Event token budget = remaining after profiles
    profileTokensUsed := uc.tokenizer.CountProfiles(truncatedProfiles)
    eventTokenBudget := req.MaxTokenSize - profileTokensUsed
    truncatedEvents := uc.truncateEvents(events, eventTokenBudget)

    // --- Assembly ---
    contextStr := uc.assembleContext(truncatedProfiles, truncatedEvents, req.CustomTemplate)

    return &ContextResult{
        ContextString: contextStr,
        ProfileCount:  len(truncatedProfiles),
        EventCount:    len(truncatedEvents),
        TokensUsed:    uc.tokenizer.CountString(contextStr),
    }, nil
}
```

### 3.4. Profile Truncation Algorithm

```go
// internal/usecase/truncate_profiles.go

func Truncate(profiles []Profile, config TruncationConfig) []Profile {
    // Step 1: Sort by updated_at DESC (most recent first)
    sort.Slice(profiles, func(i, j int) bool {
        return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt)
    })

    // Step 2: Filter by only_topics
    if len(config.OnlyTopics) > 0 {
        profiles = filterByTopics(profiles, config.OnlyTopics)
    }

    // Step 3: Priority reorder (prefer_topics first, then rest)
    if len(config.PreferTopics) > 0 {
        profiles = prioritizeTopics(profiles, config.PreferTopics)
    }

    // Step 4: Per-topic subtopic limits
    profiles = applyTopicLimits(profiles, config.TopicLimits, config.MaxSubtopicSize)

    // Step 5: Token budget enforcement (stop when exceeds budget)
    var result []Profile
    tokenCount := 0
    for _, p := range profiles {
        tokens := tokenizer.Count(formatProfileLine(p))
        if tokenCount + tokens > config.MaxTokenSize {
            break
        }
        tokenCount += tokens
        result = append(result, p)
    }
    return result
}
```

### 3.5. Redis Profile Caching

```go
// internal/adapter/repository/redis/profile_cache.go

type ProfileCache interface {
    Get(ctx context.Context, projectID, userID string) ([]Profile, bool, error)
    Set(ctx context.Context, projectID, userID string, profiles []Profile, ttl time.Duration) error
    Invalidate(ctx context.Context, projectID, userID string) error
}

// Redis key: "profiles::{project_id}::{user_id}"
// Value: JSON-serialized []Profile
// TTL: 1200s (20 minutes) — configurable
// Invalidation triggers:
//   NATS memobase.profile.changed → Invalidate(userID, projectID)
//   NATS memobase.admin.user.deleted → Invalidate(userID, projectID)
//   POST/PUT/DELETE /profile/{user_id} → immediate invalidate
```

### 3.6. Context Output Format

```
# Memory
Unless the user has relevant queries, do not actively mention those memories.
## User Background:
- basic_info::name: Alice
- basic_info::age: 28
- interest::food: Mexican cuisine, Thai food
- work::company: Acme Corp

## Latest Events:
- Discussed Q3 project deadline, feeling stressed about workload
- Mentioned starting a new fitness routine
```

Custom template support:
```
// customize_context_prompt = "Context:\n{profile_section}\nTimeline:\n{event_section}"
// Template placeholders: {profile_section}, {event_section}
```

### 3.7. Manual Profile CRUD

```go
// POST /api/v1/users/profile/{user_id} — Add manual profile slot
// Input: { content, attributes: {topic, sub_topic} }
// → Insert user_profiles record
// → Invalidate Redis cache

// PUT /api/v1/users/profile/{user_id}/{profile_id} — Update profile slot
// Input: { content, attributes: {topic, sub_topic} }
// → Update user_profiles record
// → Invalidate Redis cache

// DELETE /api/v1/users/profile/{user_id}/{profile_id} — Delete profile slot
// → Delete user_profiles record
// → Invalidate Redis cache

// GET /api/v1/users/profile/{user_id} — Get all profiles (cache-first)
// Parameters: prefer_topics[], only_topics[], max_token_size, topk, max_subtopic_size
```

### 3.8. gRPC API

```protobuf
service ContextService {
    // Context assembly
    rpc GetContext(GetContextRequest) returns (GetContextResponse);

    // Profile CRUD
    rpc GetProfiles(GetProfilesRequest) returns (GetProfilesResponse);
    rpc AddProfile(AddProfileRequest) returns (AddProfileResponse);
    rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileResponse);
    rpc DeleteProfile(DeleteProfileRequest) returns (DeleteProfileResponse);

    // Profile truncation (utility)
    rpc TruncateProfiles(TruncateProfilesRequest) returns (TruncateProfilesResponse);
}

message GetContextRequest {
    string user_id = 1;
    string project_id = 2;
    int32 max_token_size = 3;             // default: 500
    repeated string prefer_topics = 4;
    repeated string only_topics = 5;
    double profile_event_ratio = 6;       // default: 0.7
    repeated ChatMessage chats = 7;       // for semantic event search
    string customize_context_prompt = 8;
    double event_similarity_threshold = 9; // default: 0.2
    int32 time_range_in_days = 10;        // default: 21
    map<string, int32> topic_limits = 11;
}

message GetContextResponse {
    string context_string = 1;
    int32 profile_count = 2;
    int32 event_count = 3;
    int32 tokens_used = 4;
}
```

### 3.9. NATS Events

| Subject | Direction | Handler |
|---|---|---|
| `memobase.profile.changed` | Subscribe | Invalidate Redis cache |
| `memobase.admin.user.deleted` | Subscribe | Delete all cached profiles |
| `memobase.admin.project.updated` | Subscribe | Reload project config |

---

## 4. Configuration

```yaml
context:
  grpc:
    port: 9043
  health:
    port: 9093
  cache:
    redis_url: "redis://redis:6379/1"
    profile_ttl: 1200s                    # 20 minutes
  context:
    default_max_tokens: 500
    default_profile_event_ratio: 0.7
    default_event_time_range_days: 21
    default_event_topk: 5
    default_similarity_threshold: 0.2
  database:
    url: "${DATABASE_URL}"
    pool_size: 30
    max_overflow: 20
  services:
    event: { address: "memobase-event:9044", timeout: 10s }
```

---

## 5. Acceptance Criteria

- [ ] `GET /api/v1/users/context/{user_id}` → trả về context string trong < 100ms (P99, trừ embedding call).
- [ ] Context string format: `# Memory\n## User Background:\n- topic::sub_topic: content\n\n## Latest Events:\n- ...`.
- [ ] `max_token_size=200` → `tokens_used ≤ 200`.
- [ ] `prefer_topics=["basic_info"]` → basic_info profiles xuất hiện đầu tiên trong context.
- [ ] `only_topics=["work"]` → context chỉ chứa work profiles, không có basic_info.
- [ ] `profile_event_ratio=0.8` → profiles chiếm ~80% token budget, events ~20%.
- [ ] Redis cache hit: 2nd request cùng user → sub-5ms (cache hit), không query DB.
- [ ] Sau engine flush → NATS `memobase.profile.changed` → cache invalidated → next request queries DB.
- [ ] `POST /api/v1/users/profile/{user_id}` manual add → profile visible trong next GET.
- [ ] `DELETE /api/v1/users/profile/{user_id}/{profile_id}` → profile removed, cache invalidated.
- [ ] `chats` parameter provided → event search sử dụng semantic similarity với latest chats.
- [ ] `customize_context_prompt = "Context:\n{profile_section}\nTimeline:\n{event_section}"` → output theo template.
