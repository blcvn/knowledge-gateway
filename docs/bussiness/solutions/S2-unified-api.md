# S2 — Unified Memory API

> **Giải quyết Pain Points:** PP-P1-02, PP-P6-01, PP-P6-02
> **Actor chính:** P1 (AI Agent Developer), P6 (Framework Integrator)
> **Features:** F01, F10

---

## Vấn đề cần giải quyết

Developer phải tích hợp 6 storage backends riêng lẻ, mỗi cái có API khác nhau, schema khác nhau, error format khác nhau. Không có unified view về "AI đang biết gì". Code phức tạp, dễ bug, khó maintain.

---

## Giải pháp: Single API Gateway + Auto-routing

### 1 endpoint duy nhất — Gateway tự route

```
POST /v1/memory/store   ← Mọi loại memory đều qua đây
POST /v1/memory/recall  ← Tìm kiếm cross-engine
POST /v1/memory/forget  ← Xóa cascading
GET  /v1/memory/timeline ← Timeline tất cả events
```

**Auto-routing logic:**
```
POST /v1/memory/store
{
  "content": "...",
  "type": "profile"      ← Gateway đọc field này
}
        │
        ▼
Router:
  "semantic"       → cognee-ingestion
  "episodic"       → graphiti-ingestion
  "conversational" → zep-memory
  "profile"        → memobase-ingestion  ←── đây
  "procedural"     → ov-fs
  "adaptive"       → sm-memory
  "auto"           → LLM classify content → route
```

Developer **không cần biết** engine nào xử lý. Không cần biết gRPC endpoint, schema, auth của từng engine.

---

### Cross-engine Search — 1 query, nhiều engines

**Trước VNP Memory:**
```python
# Developer phải viết mỗi project
results = []
results += vector_db.search(query, limit=10)
results += neo4j.run(cypher_query)
results += redis.get(f"session:{user_id}")
results += pg.execute("SELECT * FROM profiles WHERE ...")
merged = my_custom_merge(results)  # tự viết merge logic
reranked = my_rerank(merged)       # tự viết rerank
```

**Với VNP Memory:**
```http
POST /v1/memory/recall
{
  "query": "user coding preferences",
  "user_id": "user-123",
  "engines": ["all"],         // hoặc chỉ định: ["zep", "memobase"]
  "limit": 10,
  "token_budget": 2000
}
```

**Cơ chế Hybrid Search Engine (F10):**
```
vnp-search-hub nhận query
        │
        ▼ Parallel fan-out (gRPC)
        ├── cognee-search    → BM25 + semantic results
        ├── graphiti-search  → temporal graph results
        ├── zep-search       → session context results
        ├── memobase-context → profile context
        ├── ov-search        → filesystem search results
        └── sm-search        → adaptive KG results
        │
        ▼
RRF Fusion (Reciprocal Rank Fusion)
        │
        ▼
Merged + reranked results → Response
```

**3 chiến lược retrieval kết hợp:**
| Strategy | Implementation | Tốt cho |
|---|---|---|
| **BM25** | In-memory inverted index, TF-IDF | Exact keyword match |
| **Vector** | pgvector cosine similarity | Semantic similarity |
| **RRF** | Reciprocal Rank Fusion | Combine cả hai |

---

### Standard Protocol — MCP cho AI Frameworks

Thay vì mỗi framework implement memory khác nhau, VNP Memory expose **Model Context Protocol** (standard do Anthropic define):

```
AI Framework (LangChain, AutoGen, CrewAI...)
        │
        │ MCP JSON-RPC 2.0
        ▼
VNP Memory Gateway :8082
        │
        │ Internal gRPC
        ▼
Memory Engines (Zep, Memobase, Supermemory...)
```

**37+ MCP Tools** — framework gọi tool, Memory tự xử lý:
```json
{
  "method": "tools/call",
  "params": {
    "name": "memory_store",
    "arguments": {
      "content": "User prefers dark mode",
      "type": "profile",
      "user_id": "user-123"
    }
  }
}
```

---

## Kết quả

| Metric | Trước | Sau |
|---|---|---|
| Số API cần biết | 6 (mỗi engine) | 1 (Unified API) |
| Lines of code để integrate memory | ~500 LOC | ~20 LOC |
| Search logic tự viết | Có | Không (built-in RRF) |
| Framework integration | Custom per framework | MCP (universal) |
| Time to first memory operation | 1-2 ngày | < 5 phút |
