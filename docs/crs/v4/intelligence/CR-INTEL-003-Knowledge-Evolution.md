# Change Request: CR-INTEL-003 — Knowledge Evolution & Contradiction Resolution

**CR ID:** CR-INTEL-003
**Component:** `backend/services/sm-memory`, `backend/services/sm-engine`
**Priority:** 🟡 High
**Status:** Open
**Version:** v4 / Intelligence Layer
**Solution:** [S4 — Knowledge Evolution](../../../bussiness/solutions/S4-knowledge-evolution.md)
**Features:** [F07](../../../features/07-adaptive-memory-supermemory/README.md), [F09](../../../features/09-agent-memory-lifecycle/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-04 | AI Agent Developer | Knowledge graph không tự update — stale facts |
| PP-P7-02 | AI Power User | AI nhớ thông tin cũ — "tôi đã đổi project rồi" |

**Before:** Update fact X → 2 versions cùng tồn tại, conflict.
**After:** Living KG — isLatest=true chỉ cho version mới nhất, auto-resolve contradictions.

---

## 2. Living Knowledge Graph

```
Version chain example:
  v1: "Project deadline = Sept 1" (created Aug 1, is_latest=false)
  v2: "Project deadline = Oct 1"  (created Sep 3, is_latest=true, parent_id=v1)

Contradiction resolution:
  New fact: "Project deadline = Nov 1"
  → Detect: same subject "project_deadline" exists
  → Set v2.is_latest = false
  → Create v3 with parent_id = v2
```

---

## 3. API Contract

```http
# Store adaptive memory (auto-versions)
POST /v1/memory/store
{
  "type": "adaptive",
  "content": "Project deadline moved to November 1",
  "metadata": {"subject": "project_deadline"}
}
→ {
    "id": "sm_v3",
    "version": 3,
    "superseded": "sm_v2",
    "is_latest": true
  }

# Get version chain
GET /v1/sm/memories/{id}/history
→ [
    {"id": "sm_v1", "content": "...Sept 1", "created_at": "...", "is_latest": false},
    {"id": "sm_v2", "content": "...Oct 1",  "created_at": "...", "is_latest": false},
    {"id": "sm_v3", "content": "...Nov 1",  "created_at": "...", "is_latest": true}
  ]
```

---

## 4. Thay đổi đề xuất

### 4.1 `backend/services/sm-memory/internal/domain/entity.go` [MODIFY]

```go
type SMMemory struct {
    ID          string
    TenantID    string
    ParentID    *string    // version chain
    RootID      string     // first version
    IsLatest    bool       // true = current version
    Subject     string     // for contradiction detection
    Content     string
    ForgetAfter *time.Time // optional TTL
    CreatedAt   time.Time
}
```

---

## 5. Acceptance Criteria

- [ ] Version chain: parent → root linkage đúng
- [ ] `is_latest=true` chỉ có 1 version per subject per user
- [ ] Contradiction auto-detected khi cùng subject
- [ ] `GET /v1/sm/memories/{id}/history` trả full version chain
- [ ] `forgetAfter` TTL: memory tự evict sau deadline
