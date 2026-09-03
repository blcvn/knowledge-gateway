# S4 — Adaptive Knowledge Evolution

> **Giải quyết Pain Points:** PP-P1-04, PP-P7-02
> **Actor chính:** P1 (AI Agent Developer), P7 (AI Power User)
> **Features:** F07 (Supermemory), F09 (Memory Lifecycle)

---

## Vấn đề cần giải quyết

Knowledge trong AI memory không tự cập nhật khi thực tế thay đổi. User chuyển từ React sang Vue — AI vẫn suggest React solutions. "Budget tăng lên" — AI vẫn nhắc số cũ. Không có contradiction resolution tự động.

---

## Giải pháp: Living Knowledge Graph + Auto-versioning

### Supermemory — Adaptive KG (F07)

Supermemory là "living knowledge graph" — memory tự động evolve:

**Version Chain:**
```
Memory v1: "User dùng React" (id: mem-001)
        │ parent_id: null, isLatest: false ← sau khi update
        │
        ▼ (relation: updates)
Memory v2: "User chuyển sang Vue" (id: mem-002)
        │ parent_id: mem-001, isLatest: true
        │ root_id: mem-001
```

**Khi store memory mới:**
```http
POST /v1/sm/memories
{
  "content": "User chuyển từ React sang Vue",
  "user_id": "user-123",
  "relation_type": "updates"   // updates | extends | derives
}
```

Supermemory tự động:
1. Detect memory liên quan (semantic similarity)
2. Xác định relation type (updates/extends/derives)
3. Mark memory cũ `isLatest=false`
4. Build version chain (parent → root)

**3 relation types:**

| Type | Ý nghĩa | Ví dụ |
|---|---|---|
| `updates` | Thay thế hoàn toàn | "Vue" thay thế "React" |
| `extends` | Bổ sung thêm | Thêm skill mới vào profile |
| `derives` | Suy ra từ | Insight từ nhiều facts |

---

### Auto-forgetting — Memory không tồn tại mãi mãi

Không phải mọi memory đều cần giữ mãi. `forgetAfter` định nghĩa vòng đời:

```http
POST /v1/sm/memories
{
  "content": "User đang debug issue X",
  "forget_after": "7d"    // Tự động xóa sau 7 ngày
}

POST /v1/sm/memories
{
  "content": "User là senior engineer",
  "forget_after": "never" // Permanent
}
```

**Retention policy theo loại:**
| Memory Type | forget_after mặc định |
|---|---|
| Working task context | 7 ngày |
| Project facts | 1 năm |
| User preferences | never |
| Session summaries | 90 ngày |

---

### Memory Lifecycle — Decay & Eviction (F09)

AgentMemory layer thêm **salience scoring** để quyết định memory nào giữ khi storage đầy:

```
eviction_score = importance × recency_factor × access_frequency

importance:        manual score hoặc inferred từ memory type
recency_factor:    1 / (1 + days_since_last_access)
access_frequency:  số lần được recalled
```

Memory có `eviction_score` thấp nhất sẽ bị evict trước — giống "forgetting curve" của não người.

**Jaccard-based Deduplication:**
```
New memory: "User thích TypeScript"
Existing:   "User prefer TypeScript over JavaScript"
                │
                ▼ Jaccard similarity > 0.8
                → Merge thay vì tạo duplicate
```

---

## Luồng Evolution hoàn chỉnh

```
User nói: "Tôi vừa chuyển team, giờ dùng Python thay Go"
        │
        ▼
POST /v1/memory/store {"content": "...", "type": "adaptive"}
        │
        ▼
Supermemory:
  1. Semantic search: tìm memories liên quan đến "programming language"
  2. Tìm: "User dùng Go" (similarity > threshold)
  3. Create new memory với relation_type="updates"
  4. Mark "User dùng Go" → isLatest=false
  5. Publish NATS: "memory.updated" event
        │
        ▼
Memobase (YOLO flush nếu đủ blobs):
  1. Extract: fact category="fact", key="language", value="Python"
  2. Merge với profile: replace previous value="Go"
  3. Update profile score
        │
        ▼
Lần recall tiếp theo:
  memory_recall("programming language preference")
  → Trả về "Python" (isLatest=true) ✓
  → KHÔNG trả về "Go" (isLatest=false) ✓
```

---

## Kết quả

| Scenario | Trước | Sau |
|---|---|---|
| User đổi framework | AI vẫn suggest framework cũ | Tự update, isLatest tracking |
| Mâu thuẫn trong memory | Cả 2 facts cùng tồn tại | Contradiction resolution tự động |
| Memory hết hạn | Phải tự viết cleanup job | forgetAfter tự động |
| Storage đầy | Không biết xóa gì | Eviction score tự động |
| Duplicate memories | Phải tự detect | Jaccard dedup tự động |
