# Feature 09 — Agent Memory Lifecycle

> **Loại:** AgentMemory | **Priority:** P0 | **Status:** Implemented (CR-AM-002)

## Mô tả

Agent Memory Lifecycle quản lý vòng đời của memories trong AgentMemory layer — bao gồm 6 loại memory chuyên biệt, **Jaccard-based versioning** để tránh duplicate, **memory decay** theo thời gian, **TTL auto-forget**, và **eviction policy** thông minh dựa trên importance × recency × frequency.

Đây là layer memory thứ 2 (sau 6 engines chính), được thiết kế đặc biệt cho AI Agent operations.

---

## Business Logic

### 6 Memory Types

AgentMemory phân loại memory theo 6 types chuyên biệt:

| Type | Mô tả | Ví dụ |
|------|-------|-------|
| `pattern` | Coding patterns, problem-solving patterns | "Luôn validate input trước khi xử lý" |
| `preference` | User/agent preferences | "User thích TypeScript over JavaScript" |
| `bug` | Bugs và cách fix | "Bug: nil pointer khi array empty → check len() trước" |
| `knowledge` | Domain knowledge | "Go generics cú pháp: func[T any]()" |
| `context` | Project/session context | "Project dùng Clean Architecture" |
| `procedure` | Step-by-step procedures | "Deploy: build → test → docker push → helm upgrade" |

### Jaccard-based Versioning

Khi lưu memory mới, hệ thống kiểm tra similarity với existing memories:

1. Tính **Jaccard similarity** giữa new memory và candidates (dựa trên token overlap).
2. Nếu `similarity > 0.7`: memory được coi là **version mới** của memory cũ.
   - Memory cũ: `isLatest = false`
   - Memory mới: `parentID = old.ID`, `isLatest = true`
3. Nếu `similarity <= 0.7`: memory là **độc lập**, không có parent.

Threshold 0.7 đảm bảo: chỉ những memories thực sự là "cùng topic" mới bị group thành version chain.

### Memory Strength & Time Decay

Mỗi memory có `strength` score (0.0 - 1.0) được tính theo công thức:

```
strength = base_strength * e^(-decay_rate * time_since_last_access)
```

- `base_strength`: Khởi tạo từ creation confidence (0.0 - 1.0).
- `decay_rate`: Configurable per memory type. `bug` decay chậm hơn `context`.
- Memory không được access sẽ decay dần theo thời gian.

### TTL Auto-forget

Developer có thể set `forgetAfter` duration khi tạo memory:

- Background scheduler chạy định kỳ.
- Memory expired: set `is_active = false` → không xuất hiện trong search.
- Memory không bị xóa physical (để maintain audit trail).

### Eviction Policy

Khi storage đạt limit, eviction policy chọn memory ít quan trọng nhất để deactivate:

```
eviction_score = importance × recency × frequency
```

- `importance`: Manual score hoặc inferred từ memory type.
- `recency`: Thời gian kể từ lần access cuối.
- `frequency`: Số lần được accessed.

Memory có `eviction_score` thấp nhất bị evict trước.

### Memory Slots

**Memory Slots** là named, editable blocks — tương tự environment variables nhưng cho AI memory. Slot là vị trí cố định với `scope` và `label`, có thể được ghi đè:

- `scope`: `agent` | `session` | `project` | `global`
- `label`: Tên của slot (e.g., "current-task", "user-preferences")

Slots hữu ích cho structured working memory như: "current sprint goal", "active error context", "user's communication style".

---

## Dataflow

### Memory Store with Versioning

```
POST /v1/memory/agent/remember
        │
        ├── Input: {content: "...", type: "pattern|bug|...", forgetAfter: "30d"}
        │
        ▼
memory-service (AgentMemory domain)
        │
        ├── 1. Generate token set from content
        │
        ├── 2. Load candidate memories (same type, same agent/session)
        │
        ├── 3. Jaccard Similarity Check
        │         For each candidate:
        │             similarity = |intersection| / |union| (token sets)
        │             if similarity > 0.7 → version candidate found
        │
        ├── 4a. Version found:
        │         ├── Set candidate.isLatest = false
        │         └── Create new memory: {parentID: candidate.ID, isLatest: true}
        │
        ├── 4b. No version found:
        │         └── Create new memory: {parentID: nil, isLatest: true}
        │
        ├── 5. Set TTL schedule (if forgetAfter provided)
        │
        └── 6. Store to: agent_memories table + search indexes
```

### Memory Decay Scheduler

```
Background Scheduler (runs every hour)
        │
        ├── Query: all active memories where last_accessed < threshold
        │
        ├── For each memory:
        │         new_strength = base_strength * e^(-decay_rate * elapsed_hours)
        │         UPDATE agent_memories SET strength = new_strength
        │
        └── Memories with strength < 0.1 → schedule for eviction review
```

### Eviction Flow

```
POST /v1/memory/agent/evict
        │
        ├── Trigger: manual OR storage limit reached
        │
        ▼
memory-service
        │
        ├── Query: all active memories ordered by eviction_score ASC
        │         eviction_score = importance × recency_factor × access_frequency
        │
        ├── Select bottom N memories for eviction
        │
        └── Set is_active = false (soft delete, preserves audit trail)
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/memory/agent/remember` | Store agent memory |
| `GET` | `/v1/memory/agent/list` | List memories (filterable) |
| `GET` | `/v1/memory/agent/{id}` | Get memory detail |
| `DELETE` | `/v1/memory/agent/{id}` | Delete memory |
| `GET` | `/v1/memory/agent/{id}/retention` | Get retention/strength score |
| `POST` | `/v1/memory/agent/evict` | Manual eviction |
| `POST` | `/v1/memory/agent/auto-forget` | Run TTL sweep |
| `GET` | `/v1/memory/slots` | List all slots |
| `GET` | `/v1/memory/slots/{scope}/{label}` | Read slot |
| `POST` | `/v1/memory/slots/{scope}/{label}` | Write slot |
| `DELETE` | `/v1/memory/slots/{scope}/{label}` | Delete slot |

---

## Database Tables

| Table | Nội dung |
|-------|---------|
| `agent_memories` | Memory records với versioning, strength, TTL |
| `memory_slots` | Named memory slots (scope + label) |
