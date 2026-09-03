# SOL-CORE-004 — Solution: Persistent Session Context

| Field | Value |
|---|---|
| **Solution ID** | SOL-CORE-004 |
| **CR** | [CR-CORE-004](../../../../docs/crs/v3/core-memory/CR-CORE-004-Persistent-Session-Context.md) |
| **TDD ref** | [06-zep-services.md](../../../tdd/architecture/06-zep-services.md) · [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

---

## 1. Giải pháp

Persistent session context = lưu conversation history + user profile context across sessions.
Implemented via **Zep** (conversational memory) + **Memobase** (profile context).

### 1.1 `services/zep-memory/internal/usecase/context.go` [MODIFY]

```go
// GetContext — assemble context từ multiple sources
func (u *ContextUseCase) GetContext(ctx context.Context, req *GetContextRequest) (*ContextBundle, error) {
    // 1. Last N messages từ thread
    messages, _ := u.threadRepo.GetLastMessages(ctx, req.ThreadID, req.MessageLimit)
    
    // 2. Summary của session trước (nếu có)
    prevSummary, _ := u.summaryRepo.GetLatest(ctx, req.UserID, req.TenantID)
    
    // 3. User facts từ knowledge graph
    facts, _ := u.graphRepo.GetUserFacts(ctx, req.UserID)
    
    return &ContextBundle{
        Messages:        messages,
        PreviousSummary: prevSummary,
        UserFacts:       facts,
        TokenCount:      u.countTokens(messages, prevSummary, facts),
    }, nil
}
```

### 1.2 `services/memobase-context/internal/usecase/session.go` [MODIFY]

```go
// PersistSession — save session context khi end session
func (u *SessionUseCase) PersistSession(ctx context.Context, req *PersistRequest) error {
    // 1. Extract key facts từ conversation
    facts, err := u.llm.ExtractFacts(ctx, req.Messages)
    
    // 2. Merge vào user profile (YOLO merge)
    profile, _ := u.profileRepo.GetOrCreate(ctx, req.UserID, req.TenantID)
    merged := yoloMerge(profile, facts)
    
    // 3. Store updated profile
    return u.profileRepo.Update(ctx, merged)
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/zep-memory/internal/usecase/context.go` | MODIFY — multi-source context assembly |
| `services/memobase-context/internal/usecase/session.go` | MODIFY — persist on session end |
| `deployment/dev/migrations/0XX_session_summaries.sql` | NEW |

---

## 3. Acceptance Criteria

- [ ] `GET /v1/memory/context` trả về assembled context trong < 200ms
- [ ] Session history persist sau `POST /v1/observe/sessions/{id}/end`
- [ ] Context token count reported in response
- [ ] Cross-session facts accumulate correctly
