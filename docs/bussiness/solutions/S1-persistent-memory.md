# S1 — Persistent Memory Layer

> **Giải quyết Pain Points:** PP-P1-01, PP-P5-01, PP-P7-01
> **Actor chính:** P1 (AI Agent Developer), P5 (IDE Plugin User), P7 (AI Power User)
> **Features:** F01, F04, F05, F07

---

## Vấn đề cần giải quyết

AI Agent và AI Assistant không có bộ nhớ giữa các phiên. Mỗi conversation là tờ giấy trắng. User phải giải thích lại mọi thứ từ đầu — ngữ cảnh, sở thích, project, ràng buộc.

**Hậu quả đo được:**
- User mất 5-10 phút "brief" AI mỗi session
- AI đưa ra suggestions mâu thuẫn với thông tin đã nói trước
- User frustration → churn

---

## Giải pháp: Multi-Layer Persistent Memory

VNP Memory cung cấp **4 lớp lưu trữ khác nhau**, mỗi lớp tối ưu cho 1 loại thông tin:

```
┌─────────────────────────────────────────────────────┐
│              Unified Memory API (F01)                │
│          POST /v1/memory/store  (auto-route)         │
└──────────┬──────────┬──────────┬────────────────────┘
           │          │          │
    ┌──────▼──┐ ┌─────▼────┐ ┌──▼──────────┐ ┌──────────────┐
    │   Zep   │ │ Memobase │ │  Supermemory│ │  OpenViking  │
    │  (F04)  │ │  (F05)   │ │    (F07)    │ │    (F06)     │
    │Session  │ │  Profile │ │  Adaptive   │ │  Procedural  │
    │ memory  │ │  memory  │ │    KG       │ │  filesystem  │
    └─────────┘ └──────────┘ └─────────────┘ └──────────────┘
```

### Layer 1 — Conversational Memory (Zep, F04)

**Dùng cho:** Lịch sử hội thoại, context của session hiện tại và sessions trước.

```http
PUT /v1/zep/sessions/{session_id}/memory
{
  "messages": [
    {"role": "user", "content": "Tôi đang dùng PostgreSQL 15"},
    {"role": "assistant", "content": "OK, tôi sẽ không suggest upgrade"}
  ]
}
```

**Cơ chế:** Zep xây dựng knowledge graph từ conversation — entities, relationships, facts được extract tự động. Session tiếp theo query graph để retrieve relevant context.

**Kết quả:** Context persist cross-session, không mất sau `session_end`.

---

### Layer 2 — Profile Memory (Memobase YOLO Engine, F05)

**Dùng cho:** Thông tin về user — sở thích, thói quen, mục tiêu, facts.

```http
POST /v1/memobase/users/{user_id}/blobs
{
  "blob_type": "conversation",
  "content": "User nói: tôi prefer TypeScript, không dùng Python"
}
```

**Cơ chế YOLO Engine — 3 LLM calls cố định:**
```
Blob → Buffer (tích lũy đến 20 blobs)
    → Flush:
        Call 1: Extract profiles từ blobs
        Call 2: Merge với existing profile (avoid duplication)
        Call 3: Generate event timeline
    → Structured profile: {key, value, category, score}
```

**Profile categories:**
| Category | Ví dụ |
|---|---|
| `preference` | "Thích TypeScript hơn JavaScript" |
| `fact` | "Senior developer, 8 năm kinh nghiệm" |
| `goal` | "Đang xây microservices platform" |
| `habit` | "Code vào buổi sáng, review PR buổi chiều" |

**Truy xuất nhanh:**
```http
GET /v1/memobase/users/{user_id}/context
→ Prompt-ready string < 100ms
→ Gồm: Summary + Profiles + Events + TokenCount
```

---

### Layer 3 — Adaptive Memory (Supermemory, F07)

**Dùng cho:** Knowledge về domain, projects, long-term facts cần persist vĩnh viễn.

```http
POST /v1/sm/memories
{
  "content": "Project Alpha dùng microservices, deployment qua Kubernetes",
  "user_id": "user-123",
  "forget_after": "never"  // hoặc "30d", "1y"
}
```

**Kết quả:** Knowledge persist vĩnh viễn, self-update khi có thông tin mới (xem S4).

---

### Layer 4 — Procedural Memory (OpenViking, F06)

**Dùng cho:** Files, code patterns, project structure — dữ liệu có cấu trúc dạng filesystem.

```
VikingFS:
/projects/alpha/
    context.md          ← Project overview (L0: 100 tokens)
    architecture.md     ← Architecture details (L1: 2K tokens)
    src/                ← Source files (L2: full)
```

---

## Luồng hoạt động — Session tiếp theo

```
User bắt đầu session mới
        │
        ▼
AI Agent gọi: POST /v1/memory/recall
{
  "query": "user preferences và project context",
  "user_id": "user-123"
}
        │
        ▼
vnp-search-hub: parallel fan-out tới tất cả engines
        ├── Memobase context → profile (< 100ms)
        ├── Zep session history → recent conversations
        ├── Supermemory → project facts
        └── OpenViking → project files (L0 abstracts)
        │
        ▼
Merge + rerank kết quả
        │
        ▼
AI nhận context: "User là senior TypeScript developer,
                  đang làm Project Alpha (microservices + K8s),
                  không muốn upgrade PostgreSQL..."
        │
        ▼
AI respond ngay với đầy đủ context — không cần user brief lại
```

---

## Kết quả đo được

| Metric | Trước | Sau |
|---|---|---|
| Thời gian "brief" AI mỗi session | 5-10 phút | 0 phút |
| Context accuracy (relevant facts recalled) | ~30% | >85% |
| User phải nhắc lại preferences | Mỗi session | 0 lần |
| Memobase context retrieval p95 | N/A | < 100ms |
| Cross-engine recall p95 | N/A | < 500ms |

---

## Code example — Agent tích hợp Persistent Memory

```python
from vnp_memory import MemoryClient

client = MemoryClient(api_key="vnp_...", base_url="http://localhost:8080")

# Lưu memory sau mỗi interaction
async def after_interaction(user_id: str, messages: list):
    await client.memory.store(
        content=messages,
        type="conversational",
        user_id=user_id
    )

# Recall trước mỗi LLM call
async def before_llm_call(user_id: str, query: str) -> str:
    context = await client.memory.recall(
        query=query,
        user_id=user_id,
        token_budget=2000
    )
    return context.to_prompt_string()
```

Hoặc qua **MCP** (Claude Code, AutoGen):
```
memory_store(content="...", type="profile", user_id="u123")
memory_recall(query="user preferences", user_id="u123")
```
