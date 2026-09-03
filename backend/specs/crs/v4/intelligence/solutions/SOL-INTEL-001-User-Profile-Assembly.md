# SOL-INTEL-001 — Solution: User Profile Assembly

| Field | Value |
|---|---|
| **Solution ID** | SOL-INTEL-001 |
| **CR** | [CR-INTEL-001](../../../../docs/crs/v4/intelligence/CR-INTEL-001-User-Profile-Assembly.md) |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | 🔄 Partial |
| **Priority** | 🔴 Critical |

---

## 1. Giải pháp

User Profile Assembly = aggregate profile data từ nhiều sources (Memobase blobs, Zep facts, Cognee entities) thành một unified user profile.

### 1.1 `services/memobase-engine/internal/usecase/profile_assembly.go` [NEW]

```go
type ProfileAssemblyUseCase struct {
    blobRepo    port.BlobRepository       // memobase blobs
    factRepo    port.FactRepository       // zep knowledge graph facts
    entityRepo  port.EntityRepository     // cognee graph entities
    llm         port.LLMClient
}

type UserProfile struct {
    TenantID    string
    UserID      string
    Categories  map[string][]ProfileFact  // "preferences", "skills", "work", etc.
    Summary     string                     // LLM-generated summary
    UpdatedAt   time.Time
    Version     int
}

func (u *ProfileAssemblyUseCase) AssembleProfile(ctx context.Context, tenantID, userID string) (*UserProfile, error) {
    // 1. Fetch data từ all sources (parallel)
    var wg sync.WaitGroup
    var blobs []Blob; var facts []Fact; var entities []Entity

    wg.Add(3)
    go func() { defer wg.Done(); blobs, _ = u.blobRepo.GetUserBlobs(ctx, tenantID, userID) }()
    go func() { defer wg.Done(); facts, _ = u.factRepo.GetUserFacts(ctx, tenantID, userID) }()
    go func() { defer wg.Done(); entities, _ = u.entityRepo.GetUserEntities(ctx, tenantID, userID) }()
    wg.Wait()

    // 2. LLM: classify and extract profile facts
    combined := combineData(blobs, facts, entities)
    categories, _ := u.llm.ExtractProfileCategories(ctx, combined)

    // 3. Generate summary
    summary, _ := u.llm.GenerateProfileSummary(ctx, categories)

    return &UserProfile{
        TenantID: tenantID, UserID: userID,
        Categories: categories, Summary: summary,
        UpdatedAt: time.Now(), Version: 1,
    }, nil
}
```

### 1.2 API endpoint

```go
// GET /v1/profiles/{user_id}
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "user_id")
    tenant := tenant.FromContext(r.Context())

    profile, err := h.assembly.AssembleProfile(r.Context(), tenant, userID)
    if err != nil { writeError(w, 500, "assembly_failed", err.Error()); return }
    writeJSON(w, 200, profile)
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/memobase-engine/internal/usecase/profile_assembly.go` | NEW |
| `services/memobase-engine/internal/port/assembly.go` | NEW — port interfaces |
| `gateway/adapter/handler/profile_handler.go` | MODIFY — add GET /profiles/{id} |
| `deployment/dev/migrations/0XX_user_profiles.sql` | NEW |

---

## 3. Acceptance Criteria

- [ ] `GET /v1/profiles/{user_id}` returns profile trong < 500ms (cached)
- [ ] Profile categories: preferences, skills, work_context, communication_style, goals
- [ ] Profile version tracked, history queryable
- [ ] RBAC: user can only read own profile; admin reads any

---

**Ghi chú audit:** user_profiles migration done; extract_profile + merge_profile usecase stubs exist; cross-engine aggregation (Zep/Cognee) not wired
