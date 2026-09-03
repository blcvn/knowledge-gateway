# Change Request: CR-INTEL-001 — User Profile Assembly

**CR ID:** CR-INTEL-001
**Component:** `backend/services/memobase-context`, `backend/services/memobase-engine`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v4 / Intelligence Layer
**Solution:** [S5 — User Profiling](../../../bussiness/solutions/S5-user-profiling.md)
**Features:** [F05](../../../features/05-profile-memory-memobase/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-05 | AI Agent Developer | Agent không biết user preferences — phải hỏi lại mỗi session |
| PP-P7-01 | AI Power User | AI không nhớ tên, sở thích, mục tiêu — personalization = 0 |
| PP-P8-01 | Product Manager | Không có analytics về user behavior patterns |

**Business Impact:**
- Before: AI hỏi "Bạn muốn code style nào?" mỗi session
- After: Profile facts persist → AI biết ngay "Bình thích Clean Architecture, Go"

---

## 2. User Profile Structure

```json
{
  "user_id": "u_123",
  "profiles": {
    "preference": [
      {"key": "coding_style", "value": "Clean Architecture", "score": 0.95},
      {"key": "language", "value": "Go, TypeScript", "score": 0.99}
    ],
    "fact": [
      {"key": "name", "value": "Bình", "score": 1.0},
      {"key": "role", "value": "Backend Engineer", "score": 0.92}
    ],
    "goal": [
      {"key": "current_project", "value": "VNP Memory", "score": 0.88}
    ],
    "habit": [
      {"key": "work_hours", "value": "9am-11pm", "score": 0.75}
    ]
  },
  "context_string": "Tên: Bình | Role: Backend Engineer | Thích: Go, Clean Architecture | Project: VNP Memory"
}
```

---

## 3. API Contract

```http
# Get prompt-ready context string (< 100ms)
GET /v1/memobase/users/{user_id}/context
→ {
    "context": "Tên: Bình | Role: Backend Engineer | Thích: Go, Clean Architecture",
    "token_count": 24,
    "generated_at": "2026-09-03T12:00:00Z"
  }

# Get full profile details
GET /v1/memobase/users/{user_id}/profile
→ { "profiles": { "preference": [...], "fact": [...], "goal": [...], "habit": [...] } }

# Manual flush (trigger YOLO extraction immediately)
POST /v1/memobase/users/{user_id}/flush
→ { "processed_blobs": 23, "updated_profiles": 7, "duration_ms": 850 }
```

---

## 4. YOLO Engine Implementation

```go
// backend/services/memobase-engine/internal/usecase/process_blobs.go [MODIFY]
func (u *ProcessBlobsUseCase) Execute(ctx context.Context, userID string) error {
    blobs := u.blobRepo.GetPending(ctx, userID, limit=20)
    
    // Call 1: Entry summary (batch compress blobs)
    summary := u.llm.Complete(ctx, entrySummaryPrompt(blobs))
    
    // Call 2: Extract profile topics
    candidates := u.llm.Complete(ctx, extractTopicsPrompt(summary))
    
    // Call 3: YOLO Merge (merge với existing profiles — 1 LLM call)
    existing := u.profileRepo.GetAll(ctx, userID)
    merged := u.llm.Complete(ctx, yoloMergePrompt(candidates, existing))
    
    return u.profileRepo.Save(ctx, userID, merged)
}
```

---

## 5. Acceptance Criteria

- [ ] `GET /v1/memobase/users/{uid}/context` trả về `< 100ms` (SQL only, no LLM)
- [ ] 4 profile categories: preference, fact, goal, habit
- [ ] YOLO flush: cố định 3 LLM calls bất kể số blobs
- [ ] Auto-flush khi buffer đạt 20 blobs (configurable)
- [ ] `profile_score` float64 mỗi attribute (confidence)
- [ ] Context string sẵn sàng inject vào LLM system prompt
