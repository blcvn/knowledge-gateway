# Feature 05 — Profile Memory (Memobase)

> **Loại:** Memory Engine | **Priority:** P0 | **Status:** Implemented

## Mô tả

Memobase quản lý **Profile Memory** — bộ nhớ cấu trúc về người dùng. Thay vì lưu raw conversations, Memobase tự động trích xuất và duy trì **structured user profiles** bao gồm: preferences (sở thích), facts (sự thật về user), goals (mục tiêu), habits (thói quen).

Điểm nổi bật là **YOLO Engine** — một pipeline cố định 3 LLM calls per flush, đảm bảo chi phí token predictable và context retrieval `< 100ms`.

---

## Business Logic

### Blob → Buffer → Profile Pipeline

Memobase sử dụng mô hình **buffered processing**:

1. **Blob Insertion**: Developer insert từng blob (conversation turn, document, summary) vào user's buffer.
2. **Buffer Accumulation**: Blobs tích lũy trong buffer (in-memory + PostgreSQL).
3. **Auto-flush Trigger**: Khi buffer đạt `FlushThreshold` (mặc định 20 blobs), tự động trigger flush. Developer cũng có thể manual flush.
4. **YOLO Engine**: Flush chạy pipeline 3 LLM calls cố định.
5. **Profile Update**: Profiles được cập nhật, events được log.

### YOLO Engine (3 LLM Calls Fixed)

Tên "YOLO" (You Only LLM Once per stage) phản ánh triết lý: mỗi giai đoạn flush chỉ dùng 1 LLM call, tổng cộng 3 calls bất kể số lượng blobs.

**Call 1 — Extract**:
- Input: Tất cả blobs trong buffer
- Task: Trích xuất profile candidates từ blob content
- Output: List of `{key, value, category, confidence}` candidates

**Call 2 — Merge**:
- Input: New candidates + Existing profiles
- Task: Merge candidates vào existing profiles (resolve conflicts, update scores)
- Output: Updated profile set

**Call 3 — Events**:
- Input: Blob content + profile changes
- Task: Generate `GistText` (LLM-generated summary) cho each event
- Output: Event list for timeline

### Profile Categories

| Category | Ý nghĩa | Ví dụ |
|----------|---------|-------|
| `preference` | Sở thích của user | "Thích dark mode", "Prefer Python" |
| `fact` | Sự thật về user | "Làm việc tại Acme Corp", "Sống ở HCM" |
| `goal` | Mục tiêu của user | "Muốn học machine learning" |
| `habit` | Thói quen | "Code vào buổi sáng", "Dùng Vim" |

Mỗi profile attribute có **score** (float64, 0-1) biểu thị độ tin cậy/tần suất xuất hiện.

### Context Assembly

`GET /v1/memobase/users/{uid}/context` trả về một **prompt-ready string** trong `< 100ms`:
- Summary of user profiles
- Recent events
- Token count estimate
- Configurable token budget

---

## Dataflow

### Blob Insertion & Auto-flush

```
POST /v1/memobase/users/{uid}/blobs
        │
        ├── Input: {type: "conversation|fact|document|image", content: "..."}
        │
        ▼
memobase-ingestion service
        │
        ├── 1. Validate & parse blob
        ├── 2. Insert blob to user's Buffer (PostgreSQL: blobs table)
        ├── 3. Increment buffer counter
        └── 4. Check: buffer_count >= FlushThreshold?
                  │
                  ├── YES → Trigger async flush (NATS: memobase.flush.triggered)
                  └── NO  → Return 202 Accepted (accumulate more)


NATS: memobase.flush.triggered
        │
        ▼
memobase-engine service (YOLO Engine)
        │
        ├── Load all pending blobs for user
        │
        ├── LLM Call 1 — Extract
        │         Input:  blob contents (batch)
        │         Output: [{key, value, category, confidence}]
        │
        ├── LLM Call 2 — Merge
        │         Input:  new candidates + existing profiles from DB
        │         Output: merged profile set with updated scores
        │
        ├── LLM Call 3 — Events
        │         Input:  blob content + profile diff
        │         Output: [{event_type, gist_text, timestamp}]
        │
        ├── Write to ProfileRepository
        │         └── UPSERT profiles (key/value/category/score)
        │
        ├── Write to EventRepository
        │         └── INSERT events (for timeline)
        │
        └── Clear processed blobs from buffer
```

### Context Retrieval

```
GET /v1/memobase/users/{uid}/context
        │
        ▼
memobase-context service
        │
        ├── Load profiles from ProfileRepository (cache-first)
        ├── Load recent events from EventRepository
        ├── Format as prompt-ready string
        │         └── "User preferences: dark mode, Python..."
        │             "Recent activity: asked about ML on 2026-06-15..."
        └── Return {context_string, token_count, profiles, events}
        │
        (Target: < 100ms p95)
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/memobase/users/{uid}/blobs` | Insert blob vào buffer |
| `POST` | `/v1/memobase/users/{uid}/flush` | Manual flush trigger |
| `GET` | `/v1/memobase/users/{uid}/context` | Get prompt-ready context (`< 100ms`) |
| `GET` | `/v1/memobase/users/{uid}/profiles` | Get structured profiles |
| `GET` | `/v1/memobase/users/{uid}/events` | Get event timeline |

---

## Services

| Service | Vai trò |
|---------|---------|
| `memobase-ingestion` | Blob ingestion, buffer management |
| `memobase-engine` | YOLO engine (3 LLM calls), profile extraction |
| `memobase-context` | Context assembly, profile retrieval |
| `memobase-event` | Event timeline management |
| `memobase-pipeline` | Background jobs, flush orchestration |
| `memobase-admin` | Admin APIs, configuration |

---

## Performance Targets

| Metric | Target |
|--------|--------|
| Context retrieval (p95) | `< 100ms` |
| LLM calls per flush | Fixed 3 (predictable cost) |
| Buffer flush threshold | 20 blobs (configurable) |
