# SOL-MB-003 — Solution: Context Service (Profile Read & Assembly)

| Field | Value |
|---|---|
| **Solution ID** | SOL-MB-003 |
| **CR** | CR-MB-003 |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/memobase-context` |

---

## 1. Giải pháp

Read user profile + recent blobs → assemble context for LLM injection.

### `services/memobase-context/internal/usecase/context.go` [MODIFY]

```go
func (u *ContextUseCase) GetContext(ctx context.Context, req *GetContextRequest) (*ContextBundle, error) {
    profile, _ := u.profileRepo.Get(ctx, req.TenantID, req.UserID)
    recentBlobs, _ := u.blobRepo.GetRecent(ctx, req.TenantID, req.UserID, 5)
    
    // Format for LLM injection
    return &ContextBundle{
        ProfileSummary: formatProfile(profile),
        RecentActivity: formatBlobs(recentBlobs),
        TokenCount:     u.tokenizer.Count(profile, recentBlobs),
    }, nil
}
```

## 2. Acceptance Criteria

- [ ] Context assembly < 100ms (profile cached in Redis)
- [ ] Token count always within budget
- [ ] MCP tool: profile_get_context works end-to-end

