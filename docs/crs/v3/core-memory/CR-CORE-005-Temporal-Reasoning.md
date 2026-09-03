# Change Request: CR-CORE-005 — Temporal Reasoning Pipeline

**CR ID:** CR-CORE-005
**Component:** `backend/services/graphiti-ingestion`, `backend/services/graphiti-search`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Core Memory & Integration
**Solution:** [S3 — Temporal Reasoning](../../../bussiness/solutions/S3-temporal-reasoning.md)
**Features:** [F02](../../../features/02-episodic-memory-graphiti/README.md), [F09](../../../features/09-agent-memory-lifecycle/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-03 | AI Agent Developer | RAG không hiểu thời gian — recall stale facts từ 6 tháng trước |
| PP-P3-02 | ML/AI Engineer | Không thể hỏi "Khi nào X thay đổi?" — không có temporal index |

**Before:** Recall trả về facts cũ lẫn mới, không phân biệt.
**After:** `isLatest=true` filter đảm bảo chỉ trả facts còn hiệu lực.

---

## 2. Core Mechanism

```
Khi lưu episodic memory:
  Episode mới về "X = Y" (valid_at = now)
  → Tìm episode cũ về X
  → Set old.invalid_at = now, old.is_latest = false
  → New episode: is_latest = true, valid_at = now

Search filter:
  WHERE is_latest = true   (chỉ lấy facts còn hiệu lực)
  OR valid_at BETWEEN $from AND $to  (temporal range query)
```

---

## 3. API Contract

```http
# Store episodic fact
POST /v1/memory/store
{
  "type": "episodic",
  "content": "Project deadline changed to 2026-10-01",
  "metadata": {
    "subject": "project_deadline",
    "valid_at": "2026-09-03T00:00:00Z"
  }
}

# Temporal search
POST /v1/memory/recall
{
  "query": "project deadline",
  "types": ["episodic"],
  "time_range": {"from": "2026-09-01", "to": "2026-09-03"}
}
```

---

## 4. Thay đổi đề xuất

### 4.1 `backend/services/graphiti-ingestion/internal/domain/episode.go` [MODIFY]

```go
type Episode struct {
    ID          string
    TenantID    string
    UserID      string
    Content     string
    ValidAt     time.Time
    InvalidAt   *time.Time  // null = still valid
    IsLatest    bool        // true = current fact
    Subject     string      // entity being described
    CreatedAt   time.Time
}

// Invalidation pipeline:
func (r *EpisodeRepo) UpsertEpisode(ctx context.Context, ep *Episode) error {
    // Invalidate old facts about same subject
    r.db.ExecContext(ctx, `
        UPDATE episodes
        SET invalid_at = NOW(), is_latest = false
        WHERE tenant_id = $1 AND user_id = $2 AND subject = $3 AND is_latest = true
    `, ep.TenantID, ep.UserID, ep.Subject)
    
    // Insert new fact
    return r.db.InsertEpisode(ctx, ep)
}
```

---

## 5. Acceptance Criteria

- [ ] Contradiction detection: lưu fact mới về subject X → old fact về X bị set `is_latest=false`
- [ ] Default recall filter: `is_latest=true` (chỉ facts còn hiệu lực)
- [ ] Temporal range query: `valid_at BETWEEN from AND to` hoạt động đúng
- [ ] Timeline API: `GET /v1/memory/timeline?user_id=X` trả danh sách changes theo thời gian
- [ ] Query "Khi nào X thay đổi?" trả epoch đúng
