# S5 — Automatic User Profiling

> **Giải quyết Pain Points:** PP-P1-05, PP-P7-01, PP-P7-02, PP-P8-01
> **Actor chính:** P1 (AI Agent Developer), P7 (AI Power User), P8 (Product Manager)
> **Features:** F05 (Memobase)

---

## Vấn đề cần giải quyết

Sau hàng trăm conversations, developer không có structured data về user. Không biết user thích gì, dùng gì, mục tiêu là gì. AI không thể cá nhân hóa response. Product Manager không có insights từ conversations.

---

## Giải pháp: Memobase YOLO Engine — Auto Profile Extraction

### Pipeline: Conversation → Structured Profile

```
Conversation messages
        │
        ▼
POST /v1/memobase/users/{uid}/blobs
(nhiều messages → nhiều blobs)
        │
        ▼
Buffer (tích lũy tối thiểu 20 blobs)
        │
        ▼
Auto-flush trigger (hoặc manual flush)
        │
        ▼ ─────────── YOLO Engine: 3 LLM Calls ───────────
        │
        ├── Call 1: EXTRACT
        │     Prompt: "Từ các conversations này, extract thông tin về user"
        │     Output: raw profile attributes
        │
        ├── Call 2: MERGE
        │     Prompt: "Merge với profile hiện có, tránh duplicate, resolve conflicts"
        │     Output: clean profile attributes
        │
        └── Call 3: EVENTS
              Prompt: "Generate timeline events từ conversations"
              Output: user event log
        │
        ▼
Structured Profile: {key, value, category, score}
```

**Fixed 3 LLM calls** bất kể bao nhiêu blobs → **Predictable cost**.

---

### Profile Structure

```json
GET /v1/memobase/users/{uid}/profiles

{
  "profiles": [
    {
      "key": "preferred_language",
      "value": "TypeScript",
      "category": "preference",
      "score": 0.95,
      "updated_at": "2026-09-03T08:00:00Z"
    },
    {
      "key": "experience_level",
      "value": "Senior (8 năm)",
      "category": "fact",
      "score": 0.99
    },
    {
      "key": "current_project",
      "value": "Microservices platform cho fintech",
      "category": "goal",
      "score": 0.87
    },
    {
      "key": "code_review_habit",
      "value": "Review PR vào buổi chiều, prefer small PRs",
      "category": "habit",
      "score": 0.75
    }
  ]
}
```

**4 Profile Categories:**
| Category | Ý nghĩa | Persistence | Score range |
|---|---|---|---|
| `preference` | Sở thích, style | Cao | 0.7-1.0 |
| `fact` | Thông tin khách quan | Rất cao | 0.9-1.0 |
| `goal` | Mục tiêu, dự án | Trung bình | 0.5-1.0 |
| `habit` | Thói quen, pattern | Cao | 0.6-1.0 |

---

### Context Assembly — Prompt-ready < 100ms

```http
GET /v1/memobase/users/{uid}/context

Response:
{
  "context": "User là Senior TypeScript developer (8 năm). Đang xây microservices
              platform cho fintech. Prefer small PRs, review buổi chiều. Không muốn
              upgrade PostgreSQL. Dùng macOS, VSCode, prefer dark mode.",
  "token_count": 87,
  "profiles_included": 12,
  "events_included": 5
}
```

AI inject string này vào system prompt → **personalized từ message đầu tiên**.

---

### User Event Timeline

```http
GET /v1/memobase/users/{uid}/events

[
  {
    "event_type": "profile_update",
    "description": "Cập nhật: chuyển từ React sang Vue",
    "timestamp": "2026-08-15T10:30:00Z"
  },
  {
    "event_type": "goal_added",
    "description": "Bắt đầu dự án microservices fintech",
    "timestamp": "2026-07-01T09:00:00Z"
  }
]
```

**Dùng cho P8 (Product Manager):** Aggregate events across users → insights về user behavior.

---

## Kết quả

| Metric | Trước | Sau |
|---|---|---|
| Structured profile từ conversations | Không có | Tự động extract |
| Context retrieval latency p95 | N/A | < 100ms |
| LLM cost per profile update | Unpredictable | Fixed 3 calls |
| Profile categories | Flat text | 4 structured categories + score |
| Product insights từ conversations | Không có | Aggregated profiles |
| AI personalization từ session đầu | Không | Có (profile inject vào prompt) |
