# SOL-MB-004 — Solution: Event Timeline & Semantic Search

| Field | Value |
|---|---|
| **Solution ID** | SOL-MB-004 |
| **CR** | CR-MB-004 |
| **TDD ref** | [04-memobase-services.md](../../../tdd/architecture/04-memobase-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/memobase-events` |

---

## 1. Giải pháp

Event timeline = ordered history of user events with semantic search capability.

### `services/memobase-events/internal/usecase/timeline.go` [NEW]

```go
func (u *TimelineUseCase) GetTimeline(ctx context.Context, req *TimelineRequest) ([]*UserEvent, error) {
    if req.Query != "" {
        // Semantic search over event content
        embedding, _ := u.embedder.Embed(ctx, req.Query)
        return u.eventRepo.SemanticSearch(ctx, req.TenantID, req.UserID, embedding, req.Limit)
    }
    return u.eventRepo.GetByTimeRange(ctx, req.TenantID, req.UserID, req.From, req.To, req.Limit)
}
```

## 2. Acceptance Criteria

- [ ] Timeline returns events in chronological order
- [ ] Semantic search uses pgvector cosine similarity
- [ ] Time range filtering works correctly

