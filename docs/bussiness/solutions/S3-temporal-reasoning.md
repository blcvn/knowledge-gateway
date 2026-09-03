# S3 — Temporal Reasoning & Graph Memory

> **Giải quyết Pain Points:** PP-P1-03, PP-P3-02
> **Actor chính:** P1 (AI Agent Developer), P3 (ML/AI Engineer)
> **Features:** F02 (Graphiti), F04 (Zep), F09 (Memory Lifecycle)

---

## Vấn đề cần giải quyết

Vector similarity search không hiểu thời gian. Fact "Budget là $50K" (tháng 1) và "Budget là $80K" (tháng 6) được treat như nhau — AI recall cái cũ và trả lời sai. Không query được "khi nào X thay đổi", "lúc đó user nói gì".

---

## Giải pháp: Temporal Knowledge Graph + Versioned Facts

### Graphiti — Episodic Memory với thời gian (F02)

Mỗi fact trong Graphiti có **validity window**:

```
Fact: "Budget của project Alpha là $50,000"
  valid_at:   2026-01-15T00:00:00Z
  invalid_at: 2026-06-20T00:00:00Z   ← Ngày bị thay thế
  status:     INVALIDATED

Fact: "Budget của project Alpha là $80,000"
  valid_at:   2026-06-20T00:00:00Z
  invalid_at: null                    ← Vẫn còn hiệu lực
  status:     ACTIVE
```

**Khi AI query:**
```http
POST /v1/graphiti/search
{
  "query": "project Alpha budget",
  "reference_time": "2026-09-01"     ← Query tại thời điểm này
}
→ Trả về fact ACTIVE tại reference_time → "$80,000" ✓
```

**Episode Ingestion — Graph tự extract:**
```http
POST /v1/graphiti/episodes
{
  "content": "Budget của Alpha tăng lên $80K do scope mở rộng",
  "type": "text",
  "source": "meeting_notes"
}
```

Graphiti tự động:
1. Extract entities: `Project Alpha`, `Budget`, `$80,000`
2. Extract relationships: `Alpha --[has_budget]--> $80K`
3. Detect contradiction với fact cũ → mark `invalid_at`
4. Store temporal edge với validity window

---

### Zep — Conversational Graph với Custom Ontology (F04)

Zep xây knowledge graph từ conversations với domain-specific entities:

```http
POST /v1/zep/graph/ontology
{
  "entity_types": ["Drug", "Symptom", "Diagnosis", "Patient"],
  "relation_types": ["treats", "causes", "indicates"]
}

POST /v1/zep/graph/facts
{
  "facts": [
    {"subject": "Aspirin", "predicate": "treats", "object": "Headache"}
  ]
}
```

**Graph Search — Temporal aware:**
```http
POST /v1/zep/graph/search
{
  "query": "What did user say about budget?",
  "scope": "nodes",
  "limit": 10
}
→ Returns nodes với timestamps, relationship paths
```

---

### Memory Lifecycle — isLatest & Version Chain (F09)

Tầng AgentMemory bổ sung **Jaccard-based versioning** để phát hiện duplicate/update:

```
Memory A: "User dùng React" (similarity threshold: 0.8)
Memory B: "User chuyển sang Vue"
        │
        ▼ Jaccard similarity check
        ├── B similar với A (same topic: frontend framework)
        ├── B mới hơn A
        └── Action: mark A.isLatest=false, B.isLatest=true
                    B.parent_id = A.id (version chain)
```

**Khi recall:** Chỉ return memories có `isLatest=true` → không bao giờ recall thông tin lỗi thời.

---

## Luồng xử lý temporal query

```
User hỏi: "Khi nào budget project Alpha thay đổi?"
        │
        ▼
POST /v1/memory/recall
{"query": "project Alpha budget history", "include_timeline": true}
        │
        ▼
vnp-search-hub → graphiti-search (temporal)
        │
        ▼
Graphiti trả về fact chain:
  - "$50K" [valid: Jan→Jun 2026]
  - "$80K" [valid: Jun 2026→now]
        │
        ▼
AI trả lời: "Budget thay đổi vào 20/06/2026, từ $50K lên $80K"  ✓
```

---

## Kết quả

| Scenario | Trước (Vector only) | Sau (Temporal Graph) |
|---|---|---|
| Recall fact mới nhất | Có thể recall fact cũ | Luôn trả về `isLatest=true` |
| Query tại thời điểm T | Không hỗ trợ | `reference_time` parameter |
| Contradiction resolution | Manual | Automatic (invalid_at) |
| "Khi nào X thay đổi?" | Không trả lời được | Timeline query |
| Domain-specific entities | Generic | Custom ontology |
