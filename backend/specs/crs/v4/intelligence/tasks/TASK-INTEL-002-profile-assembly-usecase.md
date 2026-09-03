# TASK-INTEL-002 — Implement ProfileAssemblyUseCase: aggregate from blobs + facts + entities

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-002 |
| **Wave** | 1 |
| **Solution** | [SOL-INTEL-001](../solutions/SOL-INTEL-001-User-Profile-Assembly.md) §1.1 |
| **Component** | `services/memobase-engine/internal/usecase/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-INTEL-001 |
| **Estimated** | 5h |

---

## Mục tiêu

Implement ProfileAssemblyUseCase: aggregate from blobs + facts + entities

---

## Công việc cụ thể

### `services/memobase-engine/internal/usecase/profile_assembly.go` [NEW]

```go
type ProfileAssemblyUseCase struct {
    blobRepo   port.BlobRepository
    factRepo   port.FactRepository
    entityRepo port.EntityRepository
    llm        port.LLMClient
    profileRepo port.ProfileRepository
}

func (u *ProfileAssemblyUseCase) AssembleProfile(ctx context.Context, tenantID, userID string) (*UserProfile, error) {
    var wg sync.WaitGroup
    var blobs []Blob; var facts []Fact; var entities []Entity

    wg.Add(3)
    go func() { defer wg.Done(); blobs, _ = u.blobRepo.GetUserBlobs(ctx, tenantID, userID) }()
    go func() { defer wg.Done(); facts, _ = u.factRepo.GetUserFacts(ctx, tenantID, userID) }()
    go func() { defer wg.Done(); entities, _ = u.entityRepo.GetUserEntities(ctx, tenantID, userID) }()
    wg.Wait()

    // LLM: extract structured profile categories
    combined := formatForLLM(blobs, facts, entities)
    categories, err := u.llm.ExtractProfileCategories(ctx, combined)
    if err != nil { return nil, err }

    // LLM: generate summary
    summary, _ := u.llm.GenerateProfileSummary(ctx, categories)

    // Save or update
    existing, _ := u.profileRepo.Get(ctx, tenantID, userID)
    if existing == nil {
        return u.profileRepo.Create(ctx, &UserProfile{
            TenantID: tenantID, UserID: userID,
            Categories: categories, Summary: summary, Version: 1,
        })
    }

    existing.Categories = categories
    existing.Summary = summary
    existing.Version++
    return u.profileRepo.Update(ctx, existing)
}
```

### Profile categories LLM prompt

```go
const extractProfilePrompt = `From the following user activity data, extract a structured user profile.
Return JSON with these keys: preferences, skills, work_context, communication_style, goals, technical_level.

Data:
%s

Return ONLY valid JSON.`
```

---

## Acceptance Criteria

- [ ] Parallel fetch from 3 sources (blobs, facts, entities)
- [ ] LLM extracts structured categories
- [ ] Profile version increments on update
- [ ] Profile history saved in user_profile_versions
- [ ] `go test ./services/memobase-engine/...` passes

## Files

```
services/memobase-engine/internal/usecase/profile_assembly.go  [NEW]
services/memobase-engine/internal/port/assembly.go             [NEW]
services/memobase-engine/internal/adapter/pg/profile_repo.go   [NEW]
```
