# TASK-SM-008 — services/profile-service: User Profile with Redis Cache

**Task ID:** TASK-SM-008  
**Wave:** 3 (Intelligence)  
**Solution:** [SOL-SM-004](../solutions/SOL-SM-004-User-Profile-Service.md)  
**Depends on:** TASK-SM-006 (memory_entries with IsStatic field)  
**Ước tính:** 3h  
**Priority:** High

---

## Mục tiêu

Tạo `services/profile-service/` với Profile Build + Redis Cache + NATS invalidation:
1. `UserProfile` entity (Static facts + Dynamic recent activities)
2. Build algorithm: `isStatic` separation, dedup, top-20 dynamic
3. Redis cache TTL 5 phút + event-driven invalidation
4. `ProfileWithSearch` combo (parallel profile + search)
5. `ToSystemPrompt()` helper cho LLM injection

---

## Công việc cụ thể

### 1. Tạo Domain Model

**`services/profile-service/internal/domain/profile.go`**

```go
type UserProfile struct {
    OrgID       string
    SpaceID     string
    Static      []string     // isStatic=true facts (long-term)
    Dynamic     []string     // isStatic=false, recent (max 20, sorted by UpdatedAt DESC)
    MemoryCount int
    UpdatedAt   time.Time
    CacheHit    bool
}

type ProfileWithSearch struct {
    Profile       UserProfile
    SearchResults []SearchResult
}

// ToSystemPrompt formats for LLM system prompt injection
// Format:
// "About the user:
// Long-term facts:
// - ...
// Current context:
// - ..."
func (p *UserProfile) ToSystemPrompt() string
```

### 2. Implement Build Profile Algorithm

**`services/profile-service/internal/usecase/build_profile.go`**

```go
// BuildProfileUseCase:
// 1. ListLatest(orgID, spaceID, isForgotten=false, isLatest=true) từ MemoryService gRPC
// 2. Sort by UpdatedAt DESC
// 3. Dedup: normalize (lowercase + trim) → seenFacts map
// 4. Split: isStatic=true → Static; isStatic=false → Dynamic (max 20)
// 5. Cache in Redis "profile:{orgID}:{spaceID}" TTL 5 phút
func (uc *BuildProfileUseCase) Execute(ctx, orgID, spaceID string) (*UserProfile, error)

func normalizeContent(s string) string {
    return strings.ToLower(strings.TrimSpace(s))
}
```

### 3. Implement Get Profile (Cache-First)

**`services/profile-service/internal/usecase/get_profile.go`**

```go
// 1. Redis GET → cache HIT: return immediately (< 100ms SLA)
// 2. Cache MISS → BuildProfile (< 500ms SLA)
func (uc *GetProfileUseCase) Execute(ctx, orgID, spaceID string) (*UserProfile, error)
```

### 4. Implement Profile + Search Combo

**`services/profile-service/internal/usecase/search_combo.go`**

```go
// Parallel execution với sync.WaitGroup:
// G1: GetProfile (from cache, ~1ms)
// G2: SearchService.Search (if query provided)
// Wait for both → dedup: remove search results already in profile
// dedupAgainstProfile: profile facts có priority cao hơn search results
func (uc *ProfileSearchComboUseCase) Execute(ctx, req ProfileSearchRequest) (*ProfileWithSearch, error)
```

### 5. Implement NATS Event Subscriber (Cache Invalidation)

**`services/profile-service/internal/adapter/subscriber/memory_events.go`**

```go
// Subscribe:
// "memory.created"   → Delete cache + async rebuild
// "memory.forgotten" → Delete cache + async rebuild
// Pattern: invalidate-then-async-rebuild (không block event processing)
func (s *MemoryEventSubscriber) Start(ctx context.Context)
```

### 6. Redis Cache Implementation

**`services/profile-service/internal/infra/redis/profile_cache.go`**

```go
// Key: "profile:{orgID}:{spaceID}"
// TTL: 5 phút (configurable)
// Serialize: JSON marshal/unmarshal
func (c *RedisProfileCache) Get(ctx, key string) (*UserProfile, error)
func (c *RedisProfileCache) Set(ctx, key string, profile *UserProfile, ttl time.Duration) error
func (c *RedisProfileCache) Delete(ctx, key string) error
```

### 7. REST Endpoints

```
GET  /api/v1/profiles              → GetProfile (query: ?spaceId=xxx)
POST /api/v1/profiles/search       → ProfileSearch combo
POST /api/v1/profiles/rebuild      → Force rebuild (admin only)
```

### 8. Bootstrap Integration

**`apps/memory/internal/bootstrap/profile.go`**:
- Init profile service + Redis cache + NATS subscriber
- Register gRPC service with InProcessRegistry

### 9. Tests

- `TestGetProfile_CacheHit_Under100ms`: Redis cache HIT → latency < 100ms
- `TestGetProfile_CacheMiss_BuildsAndCaches`: MISS → builds + stores in cache
- `TestBuildProfile_StaticSeparation`: isStatic=true memories → Static list
- `TestBuildProfile_DynamicMax20`: 30 dynamic memories → only 20 in profile
- `TestBuildProfile_Dedup`: same content (different case) → appears once
- `TestMemoryCreated_InvalidatesCache`: NATS event → cache deleted
- `TestProfileSearch_Parallel`: G1+G2 run concurrently
- `TestDedupAgainstProfile`: search result already in profile → removed
- `TestToSystemPrompt_Format`: profile → starts with "About the user:"

---

## Acceptance Criteria

- [ ] `go build ./services/profile-service/...` không lỗi
- [ ] 10 memories (5 static, 5 dynamic) → profile.Static has 5, profile.Dynamic has 5
- [ ] 30 dynamic memories → profile.Dynamic has max 20
- [ ] "User is a developer" + "user is a Developer" → dedup (appears once)
- [ ] Cache HIT → p95 < 100ms
- [ ] NATS memory.created → cache invalidated within 1s
- [ ] ProfileSearch: profile.CacheHit=true AND search results (without profile duplicates)
- [ ] ToSystemPrompt() starts with "About the user:"
- [ ] `go test ./services/profile-service/...` pass

---

## Files tạo ra

```
services/profile-service/
├── internal/
│   ├── domain/
│   │   └── profile.go
│   ├── usecase/
│   │   ├── get_profile.go
│   │   ├── get_profile_test.go
│   │   ├── build_profile.go
│   │   ├── search_combo.go
│   │   └── rebuild_profile.go
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── profile_server.go
│   │   └── subscriber/
│   │       └── memory_events.go
│   └── infra/
│       ├── redis/
│       │   └── profile_cache.go
│       └── memory_client/
│           └── memory_grpc.go

apps/memory/internal/bootstrap/
└── profile.go

gateway/adapter/handler/
└── profile_handler.go
```

## Sau khi hoàn thành

Chạy: `go build ./... && go test ./services/profile-service/...`
