# Change Request: CR-CORE-004 — Persistent Session Context

**CR ID:** CR-CORE-004
**Component:** `backend/services/zep-memory`, `backend/services/memobase-context`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Core Memory & Integration
**Solution:** [S1 — Persistent Memory Layer](../../../bussiness/solutions/S1-persistent-memory.md)
**Features:** [F04](../../../features/04-conversational-memory-zep/README.md), [F05](../../../features/05-profile-memory-memobase/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-01 | AI Agent Developer | Agent mất context sau mỗi session — user phải repeat mình |
| PP-P5-01 | IDE Plugin User | AI coding assistant quên project context — 10 phút brief mỗi sáng |
| PP-P7-01 | AI Power User | AI không nhớ preferences — mỗi session như lần đầu gặp |

**Before:** Context chỉ tồn tại trong 1 session window.
**After:** Context persist cross-session — agent nhớ từ session đầu tiên.

---

## 2. API Contract

```http
# Lưu session messages (Zep)
POST /v1/zep/sessions/{session_id}/messages
{
  "messages": [
    {"role": "user", "content": "Tôi tên Bình"},
    {"role": "assistant", "content": "Chào Bình!"}
  ]
}

# Lấy context cho session mới (cross-session)
POST /v1/memory/recall
{
  "user_id": "u_123",
  "query": "Người dùng là ai?",
  "types": ["conversational", "profile"]
}
→ Returns Zep session history + Memobase profile facts
```

---

## 3. Thay đổi đề xuất

### 3.1 Cross-session memory assembly

```go
// backend/services/zep-memory/internal/usecase/context_assembly.go [NEW]
func (u *ContextUseCase) AssembleCrossSession(ctx context.Context, userID, query string) (*ContextResult, error) {
    // 1. Recent messages từ active session
    recent := u.sessionRepo.GetRecentMessages(ctx, userID, limit=20)
    
    // 2. Long-term facts từ Memobase profile
    profile := u.memobaseClient.GetContext(ctx, userID)
    
    // 3. Relevant past sessions (Zep graph search)
    pastContext := u.graphClient.SearchRelevant(ctx, userID, query)
    
    return &ContextResult{
        RecentMessages: recent,
        UserProfile:    profile,
        PastContext:    pastContext,
    }, nil
}
```

---

## 4. Acceptance Criteria

- [ ] Session messages persist sau khi session ends (not lost)
- [ ] Cross-session recall trả kết quả từ lịch sử session
- [ ] Memobase profile được update sau mỗi flush (20 blobs default)
- [ ] Context assembly `< 200ms` (combined Zep + Memobase)
- [ ] User profile có ít nhất 4 categories: preference, fact, goal, habit
