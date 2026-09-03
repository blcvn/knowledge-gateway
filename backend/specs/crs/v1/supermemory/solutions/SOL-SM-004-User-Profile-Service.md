# Solution: SOL-SM-004 — User Profile Service

**CR ID:** CR-SM-004  
**Solution ID:** SOL-SM-004  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/profile-service/` với thuật toán Profile Build tự động từ `memory_entries`, cache Redis TTL 5 phút, và event-driven invalidation. Mục tiêu latency p95 < 100ms với cache hit, < 500ms với cache miss.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `SMProfile` entity | `services/memory-service/internal/domain/sm/` | Có: UserID, TenantID, Memories[], Stats |
| `memobase-context` service | `apps/memory/internal/bootstrap/` | Có: UserContext assembly cho Memobase |
| `memory_profile` MCP tool | `gateway/adapter/mcp/` | Có: user profile từ memobase-context |
| Redis | Infrastructure | Đã có, dùng cho cache + rate limit |

### Gap phân tích

- `SMProfile` thiếu phân biệt Static/Dynamic profile
- Chưa có event-driven cache invalidation (`memory.created` → rebuild)
- Thiếu `ProfileWithSearch` combo endpoint
- Latency chưa được optimize (không có Redis cache layer)

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service Mới

```
services/profile-service/
├── internal/
│   ├── domain/
│   │   ├── profile.go         # UserProfile, ProfileWithSearch entities
│   │   └── repository.go      # ProfileCacheRepository port
│   ├── usecase/
│   │   ├── get_profile.go     # Cache-first profile retrieval
│   │   ├── build_profile.go   # Build from memory_entries
│   │   ├── search_combo.go    # Profile + Search trong 1 call
│   │   └── rebuild_profile.go # Force rebuild
│   ├── adapter/
│   │   ├── grpc/              # ProfileService gRPC server
│   │   └── subscriber/
│   │       └── memory_events.go  # NATS: memory.created, memory.forgotten
│   └── infra/
│       ├── redis/
│       │   └── profile_cache.go  # Redis cache với TTL 5 phút
│       └── memory_client/
│           └── memory_grpc.go    # gRPC client → memory-service
```

### 3.2. Domain Model

```go
// services/profile-service/internal/domain/profile.go

package domain

import "time"

type UserProfile struct {
    OrgID        string
    SpaceID      string      // Container tag / project space
    Static       []string    // isStatic=true memories (long-term facts)
    Dynamic      []string    // isStatic=false, recent activities (sorted by UpdatedAt DESC)
    MemoryCount  int
    UpdatedAt    time.Time
    CacheHit     bool        // Debug info
}

type ProfileWithSearch struct {
    Profile       UserProfile
    SearchResults []SearchResult  // Optional, nếu query được cung cấp
}

type SearchResult struct {
    ID      string
    Content string
    Score   float64
    Type    string // "chunk" | "memory"
}
```

### 3.3. Profile Build Algorithm

```go
// services/profile-service/internal/usecase/build_profile.go

type BuildProfileUseCase struct {
    memClient  MemoryServiceClient    // gRPC client → memory-service
    cacheRepo  ProfileCacheRepository // Redis
}

func (uc *BuildProfileUseCase) Execute(ctx context.Context, orgID, spaceID string) (*UserProfile, error) {
    // Step 1: Fetch all non-forgotten, isLatest=true memories
    memories, err := uc.memClient.ListLatest(ctx, ListLatestRequest{
        OrgID:       orgID,
        SpaceID:     spaceID,
        IsForgotten: false,
        IsLatest:    true,
    })
    if err != nil { return nil, err }

    // Step 2: Separate Static vs Dynamic
    staticFacts := make([]string, 0)
    dynamicFacts := make([]string, 0)

    // Sort by UpdatedAt DESC để dynamic có thông tin gần nhất
    sort.Slice(memories, func(i, j int) bool {
        return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
    })

    seenFacts := make(map[string]bool) // Dedup

    for _, m := range memories {
        // Normalize content cho dedup (lowercase + trim)
        normalized := normalizeContent(m.Content)
        if seenFacts[normalized] { continue }
        seenFacts[normalized] = true

        if m.IsStatic {
            staticFacts = append(staticFacts, m.Content)
        } else {
            // Dynamic: chỉ lấy 20 gần nhất để tránh noise
            if len(dynamicFacts) < 20 {
                dynamicFacts = append(dynamicFacts, m.Content)
            }
        }
    }

    profile := &UserProfile{
        OrgID:       orgID,
        SpaceID:     spaceID,
        Static:      staticFacts,
        Dynamic:     dynamicFacts,
        MemoryCount: len(memories),
        UpdatedAt:   time.Now(),
    }

    // Step 3: Cache in Redis với TTL 5 phút
    uc.cacheRepo.Set(ctx, cacheKey(orgID, spaceID), profile, 5*time.Minute)

    return profile, nil
}

func cacheKey(orgID, spaceID string) string {
    return fmt.Sprintf("profile:%s:%s", orgID, spaceID)
}

func normalizeContent(s string) string {
    return strings.ToLower(strings.TrimSpace(s))
}
```

### 3.4. Get Profile (Cache-First)

```go
// services/profile-service/internal/usecase/get_profile.go

type GetProfileUseCase struct {
    cacheRepo    ProfileCacheRepository
    buildProfile *BuildProfileUseCase
}

func (uc *GetProfileUseCase) Execute(ctx context.Context, orgID, spaceID string) (*UserProfile, error) {
    // Step 1: Cache check (target: < 100ms với cache HIT)
    cached, err := uc.cacheRepo.Get(ctx, cacheKey(orgID, spaceID))
    if err == nil && cached != nil {
        cached.CacheHit = true
        return cached, nil
    }

    // Step 2: Cache miss → build from memory_entries
    // (target: < 500ms với cache MISS)
    profile, err := uc.buildProfile.Execute(ctx, orgID, spaceID)
    if err != nil { return nil, err }

    return profile, nil
}
```

### 3.5. Profile + Search Combo

```go
// services/profile-service/internal/usecase/search_combo.go

type ProfileSearchComboUseCase struct {
    getProfile   *GetProfileUseCase
    searchClient SearchServiceClient  // gRPC client → search-service
}

// Parallel execution: Profile + Search đồng thời
func (uc *ProfileSearchComboUseCase) Execute(ctx context.Context, req ProfileSearchRequest) (*ProfileWithSearch, error) {
    var (
        profile *UserProfile
        results []SearchResult
        profileErr, searchErr error
    )

    var wg sync.WaitGroup
    wg.Add(2)

    // G1: Get Profile (từ cache)
    go func() {
        defer wg.Done()
        profile, profileErr = uc.getProfile.Execute(ctx, req.OrgID, req.SpaceID)
    }()

    // G2: Search (nếu có query)
    go func() {
        defer wg.Done()
        if req.Query == "" {
            return
        }
        searchResp, err := uc.searchClient.Search(ctx, SearchRequest{
            Query:   req.Query,
            OrgID:   req.OrgID,
            SpaceID: req.SpaceID,
            Limit:   req.Limit,
        })
        if err == nil {
            searchErr = nil
            results = convertResults(searchResp)
        } else {
            searchErr = err
        }
    }()

    wg.Wait()

    if profileErr != nil { return nil, profileErr }

    // Dedup: loại bỏ search results đã có trong profile
    filteredResults := dedupAgainstProfile(profile, results)

    return &ProfileWithSearch{
        Profile:       *profile,
        SearchResults: filteredResults,
    }, nil
}

// dedupAgainstProfile: profile facts có priority cao hơn search results
func dedupAgainstProfile(profile *UserProfile, results []SearchResult) []SearchResult {
    profileSet := make(map[string]bool)
    for _, s := range profile.Static {
        profileSet[normalizeContent(s)] = true
    }
    for _, d := range profile.Dynamic {
        profileSet[normalizeContent(d)] = true
    }

    filtered := make([]SearchResult, 0)
    for _, r := range results {
        if !profileSet[normalizeContent(r.Content)] {
            filtered = append(filtered, r)
        }
    }
    return filtered
}
```

### 3.6. Event-Driven Cache Invalidation

```go
// services/profile-service/internal/adapter/subscriber/memory_events.go

type MemoryEventSubscriber struct {
    nats      NATSClient
    cacheRepo ProfileCacheRepository
    builder   *BuildProfileUseCase
}

func (s *MemoryEventSubscriber) Start(ctx context.Context) {
    // Subscribe memory.created → invalidate + rebuild
    s.nats.Subscribe(ctx, "memory.created", func(msg MemoryCreatedEvent) {
        key := cacheKey(msg.OrgID, msg.SpaceID)
        s.cacheRepo.Delete(ctx, key) // Invalidate
        // Async rebuild (không block event processing)
        go s.builder.Execute(context.Background(), msg.OrgID, msg.SpaceID)
    })

    // Subscribe memory.forgotten → invalidate + rebuild
    s.nats.Subscribe(ctx, "memory.forgotten", func(msg MemoryForgottenEvent) {
        key := cacheKey(msg.OrgID, msg.SpaceID)
        s.cacheRepo.Delete(ctx, key) // Invalidate
        go s.builder.Execute(context.Background(), msg.OrgID, msg.SpaceID)
    })
}
```

### 3.7. Redis Cache Implementation

```go
// services/profile-service/internal/infra/redis/profile_cache.go

type RedisProfileCache struct {
    client *redis.Client
}

func (c *RedisProfileCache) Get(ctx context.Context, key string) (*UserProfile, error) {
    data, err := c.client.Get(ctx, key).Bytes()
    if err == redis.Nil { return nil, ErrCacheMiss }
    if err != nil { return nil, err }

    var profile UserProfile
    return &profile, json.Unmarshal(data, &profile)
}

func (c *RedisProfileCache) Set(ctx context.Context, key string, profile *UserProfile, ttl time.Duration) error {
    data, err := json.Marshal(profile)
    if err != nil { return err }
    return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisProfileCache) Delete(ctx context.Context, key string) error {
    return c.client.Del(ctx, key).Err()
}
```

---

## 4. API Endpoints (Gateway)

```go
// gateway/adapter/handler/profile_handler.go

func (h *ProfileHandler) Register(mux *http.ServeMux) {
    // GET /api/v1/profiles?spaceId=sm_project_backend
    mux.HandleFunc("GET /api/v1/profiles", h.GetProfile)

    // POST /api/v1/profiles/search
    // Body: { "spaceId": "...", "q": "optional query", "limit": 10 }
    mux.HandleFunc("POST /api/v1/profiles/search", h.ProfileSearch)

    // POST /api/v1/profiles/rebuild
    // Body: { "spaceId": "..." }
    mux.HandleFunc("POST /api/v1/profiles/rebuild", h.RebuildProfile)
}
```

**Response:**
```json
{
  "static": [
    "User is a backend developer who prefers Go",
    "User works at ACME Corp as a senior engineer"
  ],
  "dynamic": [
    "Currently working on VNP Memory microservices",
    "Recently exploring pgvector for vector search"
  ],
  "memoryCount": 47,
  "updatedAt": "2026-06-17T00:30:00Z",
  "cacheHit": true,
  "searchResults": [...]
}
```

---

## 5. System Prompt Helper

```go
// packages/sdk/go/profile.go

func (p *UserProfile) ToSystemPrompt() string {
    var sb strings.Builder
    sb.WriteString("About the user:\n")

    if len(p.Static) > 0 {
        sb.WriteString("Long-term facts:\n")
        for _, f := range p.Static {
            sb.WriteString("- " + f + "\n")
        }
    }

    if len(p.Dynamic) > 0 {
        sb.WriteString("\nCurrent context:\n")
        for _, d := range p.Dynamic {
            sb.WriteString("- " + d + "\n")
        }
    }
    return sb.String()
}
```

---

## 6. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + Redis cache infra | 1 ngày |
| **P2** | Build profile algorithm (static/dynamic split + dedup) | 2 ngày |
| **P3** | Cache-first Get profile + Miss rebuild | 1 ngày |
| **P4** | Profile + Search combo (parallel) | 1 ngày |
| **P5** | NATS event subscriber (invalidation) | 1 ngày |
| **P6** | Gateway integration + REST handlers | 1 ngày |
| **P7** | Tests + Latency benchmarks | 1 ngày |

**Tổng:** ~8 ngày (Wave 3 — song song với CR-SM-003)

---

## 7. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| 10 memories → GET /profiles trả về Static/Dynamic đúng | Build algorithm: isStatic flag phân tách |
| Profile p95 < 100ms | Redis cache GET = ~1-5ms (HIT) |
| Memory mới → cache auto-invalidate | NATS subscriber memory.created → Delete + async rebuild |
| POST /profiles/search → profile + search results | ProfileSearchComboUseCase: parallel G1+G2 |
| Không duplicate giữa static và dynamic | `seenFacts` map dedup + `dedupAgainstProfile` |
