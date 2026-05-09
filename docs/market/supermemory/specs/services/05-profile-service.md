# 05 — Profile Service

> **gRPC**: 9004 | **Health**: 9084

---

## 1. Purpose

Tự động xây dựng và duy trì User Profiles từ accumulated memories. Tách biệt Static Profile (long-term facts) và Dynamic Profile (recent context). Target latency: **< 100ms**.

---

## 2. Clean Architecture

```
services/profile-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # UserProfile, ProfileSnapshot
│   │   ├── value_object.go     # ProfileType (static|dynamic), ProfileEntry
│   │   └── errors.go           # ErrProfileNotFound
│   ├── usecase/
│   │   ├── get_profile.go      # Fetch profile + optional search
│   │   ├── build_profile.go    # Rebuild from memories (static + dynamic)
│   │   ├── update_profile.go   # Incremental update on memory events
│   │   ├── port/
│   │   │   ├── input.go        # GetProfileUC, BuildProfileUC
│   │   │   └── output.go       # ProfileRepo, ProfileCache, MemoryReader,
│   │   │                       # SearchClient, EventPublisher
│   │   └── dto/
│   │       └── profile.go      # ProfileOutput, ProfileWithSearchOutput
│   ├── adapter/
│   │   ├── grpc/handler.go     # ProfileServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       └── profile.go  # Profile snapshots, rebuild queries
│   │   ├── cache/
│   │   │   └── redis.go        # Profile cache (TTL 5min, invalidate on update)
│   │   ├── grpc_client/
│   │   │   ├── search.go       # Call Search Service for profile+search combo
│   │   │   └── memory.go       # Read memories for profile building
│   │   └── event/
│   │       └── subscriber.go   # memory.created, memory.forgotten → update profile
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   └── 001_create_profiles.up.sql
└── Dockerfile
```

---

## 3. Profile Model

```go
type UserProfile struct {
    OrgID         string
    ContainerTag  string
    Static        []string    // Long-term stable facts
    Dynamic       []string    // Recent context, activities
    UpdatedAt     time.Time
}

// Deduplication priority: Static > Dynamic > SearchResults
type ProfileWithSearch struct {
    Profile       UserProfile
    SearchResults []SearchResult  // Optional: query-specific results
}
```

---

## 4. Profile Build Algorithm

```go
func (uc *BuildProfileUseCase) Execute(ctx context.Context, orgID, containerTag string) (*UserProfile, error) {
    // 1. Fetch all non-forgotten, latest memories for containerTag
    memories, _ := uc.memoryReader.ListByContainerTag(ctx, orgID, containerTag, ListOptions{
        OnlyLatest:  true,
        NotForgotten: true,
    })

    // 2. Classify into static vs dynamic
    var static, dynamic []string
    for _, m := range memories {
        if m.IsStatic {
            static = append(static, m.Memory)
        } else {
            dynamic = append(dynamic, m.Memory)
        }
    }

    // 3. Deduplicate
    seen := make(map[string]struct{})
    static = dedup(static, seen)
    dynamic = dedup(dynamic, seen)

    // 4. Cache result (Redis, TTL 5min)
    profile := &UserProfile{OrgID: orgID, ContainerTag: containerTag, Static: static, Dynamic: dynamic}
    uc.cache.Set(ctx, cacheKey(orgID, containerTag), profile, 5*time.Minute)

    return profile, nil
}
```

---

## 5. gRPC Interface

```protobuf
service ProfileService {
  rpc GetProfile(GetProfileRequest) returns (ProfileResponse);
  rpc GetProfileWithSearch(GetProfileWithSearchRequest) returns (ProfileWithSearchResponse);
  rpc RebuildProfile(RebuildProfileRequest) returns (ProfileResponse);
}

message GetProfileRequest {
  string container_tag = 1;
}

message GetProfileWithSearchRequest {
  string container_tag = 1;
  string query = 2;           // Optional: search query for combined response
  double threshold = 3;       // Search similarity threshold
}
```
