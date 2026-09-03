# TASK-CORE-006 — Persistent Session Context Assembly

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-006 |
| **Wave** | 2 |
| **Solution** | [SOL-CORE-004](../solutions/SOL-CORE-004-Persistent-Session-Context.md) |
| **Component** | `services/zep-memory/`, `services/memobase-context/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-CORE-001 |
| **Estimated** | 4h |

---

## Mục tiêu

Persistent context = assemble context từ multiple sources và persist session facts on end.

---

## Công việc cụ thể

### 1. `services/zep-memory/internal/usecase/context.go` [MODIFY]

```go
type GetContextRequest struct {
    TenantID    string
    UserID      string
    ThreadID    string
    MessageLimit int // default: 20
}

type ContextBundle struct {
    Messages        []Message
    PreviousSummary string
    UserFacts       []string
    TokenCount      int
}

func (u *ContextUseCase) GetContext(ctx context.Context, req *GetContextRequest) (*ContextBundle, error) {
    limit := req.MessageLimit; if limit == 0 { limit = 20 }

    var wg sync.WaitGroup
    var messages []Message; var summary string; var facts []string
    wg.Add(3)

    go func() { defer wg.Done()
        messages, _ = u.threadRepo.GetLastMessages(ctx, req.ThreadID, limit)
    }()
    go func() { defer wg.Done()
        summary, _ = u.summaryRepo.GetLatest(ctx, req.UserID, req.TenantID)
    }()
    go func() { defer wg.Done()
        facts, _ = u.graphRepo.GetUserFacts(ctx, req.UserID)
    }()
    wg.Wait()

    return &ContextBundle{
        Messages: messages, PreviousSummary: summary,
        UserFacts: facts, TokenCount: u.tokenizer.Count(messages, summary, facts),
    }, nil
}
```

### 2. `services/memobase-context/internal/usecase/session.go` [MODIFY]

```go
// Called when session ends → persist key facts to user profile
func (u *SessionUseCase) PersistSession(ctx context.Context, req *PersistRequest) error {
    facts, err := u.llm.ExtractFacts(ctx, &port.ExtractFactsRequest{
        Messages: req.Messages, MaxFacts: 10,
    })
    if err != nil { return err }

    profile, _ := u.profileRepo.GetOrCreate(ctx, req.UserID, req.TenantID)
    for k, v := range facts { profile.Categories[k] = v }
    profile.Version++
    return u.profileRepo.Update(ctx, profile)
}
```

### 3. `deployment/dev/migrations/` [NEW]

```sql
-- session_summaries table
CREATE TABLE session_summaries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    session_id TEXT NOT NULL,
    summary    TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX ON session_summaries(tenant_id, user_id);
```

---

## Acceptance Criteria

- [ ] `GetContext` parallel fetches messages + summary + facts
- [ ] TokenCount returned in response
- [ ] `PersistSession` called on session end (via NATS or direct)
- [ ] Session summary stored in `session_summaries` table

## Files

```
services/zep-memory/internal/usecase/context.go          [MODIFY]
services/memobase-context/internal/usecase/session.go    [MODIFY]
deployment/dev/migrations/0XX_session_summaries.sql      [NEW]
```
