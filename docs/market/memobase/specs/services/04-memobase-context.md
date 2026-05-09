# 04 — Memobase Context Service

> **gRPC**: 9043 | **Health**: 9093

---

## 1. Purpose

Read-path service: quản lý user profiles (CRUD + caching), assembly context string cho LLM prompt injection. Tối ưu cho latency thấp (< 100ms) với Redis cache layer.

---

## 2. Clean Architecture

```
services/memobase-context/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Profile, ContextResult, TruncationConfig
│   │   ├── value_object.go     # Topic, SubTopic, TokenBudget
│   │   └── errors.go           # ErrProfileNotFound, ErrContextTooLarge
│   ├── usecase/
│   │   ├── get_context.go      # Assemble profiles + events → context string
│   │   ├── get_profiles.go     # Read profiles (cache → DB)
│   │   ├── add_profile.go      # Manual profile creation
│   │   ├── update_profile.go
│   │   ├── delete_profile.go
│   │   ├── truncate_profiles.go # Token-budget truncation algorithm
│   │   ├── filter_profiles.go  # LLM-based relevance filtering
│   │   ├── port/
│   │   │   ├── input.go        # GetContextUseCase, GetProfilesUseCase
│   │   │   └── output.go       # ProfileRepository, ProfileCache,
│   │   │                       #   EventSearchClient, LLMClient, PromptProvider
│   │   └── dto/
│   │       ├── request.go      # ContextRequest{max_token,prefer_topics,...}
│   │       └── response.go     # ContextResponse{context_string}
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go      # memobase.context.v1.ContextService impl
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── postgres/
│   │   │   │   └── profile_repo.go  # user_profiles table
│   │   │   └── redis/
│   │   │       └── profile_cache.go # Profile caching (TTL 20min)
│   │   ├── client/
│   │   │   └── event_client.go      # gRPC → memobase-event.SearchEventGists
│   │   └── event/
│   │       └── subscriber.go   # NATS: memobase.profile.changed → invalidate cache
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Entities

```go
type Profile struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    ProjectID  string
    Content    string
    Attributes ProfileAttributes
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type ProfileAttributes struct {
    Topic    string `json:"topic"`
    SubTopic string `json:"sub_topic"`
}

type TruncationConfig struct {
    MaxTokenSize    int
    PreferTopics    []string
    OnlyTopics      []string
    MaxSubtopicSize int
    TopicLimits     map[string]int
    TopK            int
}

type ContextResult struct {
    ContextString  string
    ProfileCount   int
    EventCount     int
    TokensUsed     int
}
```

---

## 4. Use Case Flow: GetContext

```
Client → Gateway → gRPC GetContext(user_id, params)
                        │
                        ▼
        ┌──── GetContextUseCase ──────────────────────────┐
        │                                                  │
        │  errgroup.Go (parallel):                         │
        │                                                  │
        │  goroutine 1: Get Profiles                       │
        │  ┌─────────────────────────────────────────────┐│
        │  │ 1. Check Redis cache                        ││
        │  │    key: "profiles::{project}::{user}"       ││
        │  │ 2. Cache miss → query PostgreSQL            ││
        │  │ 3. Cache result (TTL 1200s)                 ││
        │  │ 4. Apply truncation:                        ││
        │  │    a. Sort by updated_at DESC               ││
        │  │    b. Apply prefer_topics priority          ││
        │  │    c. Apply only_topics filter              ││
        │  │    d. Apply per-topic limits                ││
        │  │    e. Token budget enforcement              ││
        │  └─────────────────────────────────────────────┘│
        │                                                  │
        │  goroutine 2: Get Events                         │
        │  ┌─────────────────────────────────────────────┐│
        │  │ gRPC → memobase-event.SearchEventGists()    ││
        │  │ (if chats provided → semantic search)       ││
        │  │ (else → recent events by time)              ││
        │  └─────────────────────────────────────────────┘│
        │                                                  │
        │  Assembly:                                       │
        │  1. profile_section = format profiles            │
        │  2. remaining_tokens = budget - profile_tokens   │
        │  3. event_section = truncate events to remaining │
        │  4. context = prompt_template(profile, event)    │
        │                                                  │
        │  Return: ContextResult{context_string, counts}   │
        └──────────────────────────────────────────────────┘
```

---

## 5. Caching Strategy

```go
// ProfileCache interface (port/output.go)
type ProfileCache interface {
    Get(ctx context.Context, projectID, userID string) ([]Profile, error)
    Set(ctx context.Context, projectID, userID string, profiles []Profile) error
    Invalidate(ctx context.Context, projectID, userID string) error
}

// Redis implementation
// Key: "profiles::{project_id}::{user_id}"
// Value: JSON-serialized []Profile
// TTL: 1200s (20 minutes)
// Invalidation: on NATS memobase.profile.changed event
```

---

## 6. Profile Truncation Algorithm

```go
func (uc *TruncateProfilesUseCase) Execute(
    profiles []Profile, config TruncationConfig,
) []Profile {
    // 1. Sort by updated_at DESC (most recent first)
    sort.Slice(profiles, func(i, j int) bool {
        return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt)
    })

    // 2. Apply only_topics filter
    if len(config.OnlyTopics) > 0 {
        profiles = filterByTopics(profiles, config.OnlyTopics)
    }

    // 3. Priority reorder (prefer_topics first, then rest)
    if len(config.PreferTopics) > 0 {
        profiles = prioritizeTopics(profiles, config.PreferTopics)
    }

    // 4. Per-topic subtopic limits
    profiles = applyTopicLimits(profiles, config.TopicLimits, config.MaxSubtopicSize)

    // 5. Token budget enforcement
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

---

## 7. Context Output Format

```
# Memory
Unless the user has relevant queries, do not actively mention those memories.
## User Background:
- basic_info::name: Gus
- basic_info::age: 25
- interest::food: Mexican cuisine, Thai food

## Latest Events:
- Discussed project deadline for Q3
- Mentioned feeling stressed about workload
```

Custom template support: user provides template with `{profile_section}` and `{event_section}` placeholders.

---

## 8. NATS Events

| Subject | Direction | Handler |
|---------|-----------|---------|
| `memobase.profile.changed` | Subscribe | Invalidate Redis cache |
| `memobase.admin.user.deleted` | Subscribe | Delete cached profiles |

---

## 9. Configuration

```yaml
context:
  grpc:
    port: 9043
  health:
    port: 9093
  cache:
    redis_url: "redis://redis:6379/1"
    profile_ttl: 1200s              # 20 minutes
  context:
    default_max_tokens: 500
    default_profile_event_ratio: 0.7
    default_event_time_range_days: 21
    default_event_topk: 5
    default_similarity_threshold: 0.2
  database:
    url: "${DATABASE_URL}"
    pool_size: 30                   # Read-heavy, larger pool
    max_overflow: 20
```
