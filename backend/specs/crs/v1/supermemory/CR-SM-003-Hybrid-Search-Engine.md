# Change Request: CR-SM-003 — Hybrid Search Engine (RAG + Memory)

**CR ID:** CR-SM-003  
**Component:** `services/search-service` [NEW/UPGRADE SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** Supermemory PRD §3.3, SRS §2.4, specs/services/04-search-service.md

---

## 1. Mô tả

Nâng cấp Search Engine của VNP Memory thành **Hybrid Search** kết hợp RAG (document chunks) + Memory (extracted facts) trong một truy vấn duy nhất:

1. **Dual Search Mode**: `hybrid` (mặc định), `memories-only` (v4 API), `documents-only`.
2. **3-Goroutine Parallel Search**: Tìm kiếm đồng thời trên chunks, memories, và document summaries.
3. **pgvector Integration**: Sử dụng PostgreSQL pgvector extension (HNSW index) cho vector similarity.
4. **Query Rewriting**: AI tối ưu query trước khi search (+~400ms latency trade-off).
5. **Result Reranking**: AI rerank kết quả sau khi merge.
6. **Metadata Filtering**: AND/OR filter động trên metadata JSONB.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện tại có Vector Search nhưng chưa hợp nhất với Memory facts (chỉ search documents).
- Thiếu query rewriting để cải thiện recall.
- Chưa hỗ trợ metadata filtering phức tạp (AND/OR operators).
- Chưa có context assembly (adjacent chunks, related memories chain).

---

## 3. Thay đổi đề xuất

### 3.1. [UPGRADE] `services/search-service/` (Port gRPC: 9003)

### 3.2. Search Pipeline (3 Phase)

```
Phase 1 — PREPARE:
  - Optional: AI Query Rewrite (tăng recall, +400ms)
  - Generate query embedding (dense vector)
  - Build metadata filter SQL

Phase 2 — PARALLEL SEARCH (3 goroutines):
  G1: pgvector cosine search trên chunks (WHERE similarity > chunkThreshold)
  G2: pgvector cosine search trên memory_entries (WHERE is_latest=true AND is_forgotten=false)
  G3: pgvector cosine search trên document summary_embedding

Phase 3 — MERGE + ENRICH:
  - Deduplicate by result ID
  - Apply metadata filters
  - Optional: AI Rerank
  - Context assembly:
      * Adjacent chunks (prev/next position)
      * Related memories (parents/children chain)
      * Document summaries (if includeSummary=true)
      * Full documents (if includeFullDocs=true)
  - Apply limit + threshold
```

### 3.3. Metadata Filter Engine

```go
type FilterGroup struct {
    Operator   LogicalOp     // AND | OR
    Conditions []FilterCondition
}

type FilterCondition struct {
    Field    string       // JSONB metadata field
    Operator CompareOp   // eq, ne, gt, gte, lt, lte, contains, in
    Value    any
}
// Generates SQL: WHERE metadata->>'category' = 'tech' AND (metadata->>'priority')::int >= 3
```

### 3.4. V4 Memory Search API

Endpoint riêng `/api/v4/search` (tương đương `/v4/search` trong Supermemory) chỉ search trên MemoryEntries với similarity score rõ ràng hơn.

### 3.5. pgvector Schema

```sql
-- Chunks với HNSW index
CREATE INDEX ON chunks USING hnsw (embedding vector_cosine_ops);
-- Memories với HNSW index
CREATE INDEX ON memory_entries USING hnsw (memory_embedding vector_cosine_ops);
-- Matryoshka support: thêm cột matryoshka_embedding
ALTER TABLE chunks ADD COLUMN matryoshka_embedding vector(256);
```

### 3.6. API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/api/v1/search` | Hybrid search (RAG + Memory) |
| `POST` | `/api/v4/search` | Memory-only search với scores |

**Request body example:**
```json
{
  "q": "Python performance optimization",
  "containerTags": ["sm_project_backend"],
  "limit": 10,
  "chunkThreshold": 0.7,
  "documentThreshold": 0.6,
  "rerank": true,
  "rewriteQuery": false,
  "includeFullDocs": false,
  "filters": {
    "operator": "AND",
    "conditions": [{ "field": "language", "operator": "eq", "value": "python" }]
  }
}
```

---

## 4. Acceptance Criteria

- [ ] Tìm kiếm "Python" trả về cả document chunks VÀ memory entries trong cùng một response.
- [ ] Khi `rewriteQuery=true`, query được AI rewrite trước → kết quả relevant hơn (latency tăng ~400ms).
- [ ] Filter `{"operator": "AND", "conditions": [{"field": "lang", "operator": "eq", "value": "vi"}]}` chỉ trả về kết quả có `metadata.lang = "vi"`.
- [ ] Kết quả search không bao gồm memories có `isForgotten=true`.
- [ ] Memory search V4 trả về `similarity_score` cho mỗi kết quả.
- [ ] Response thời gian p95 < 500ms (không rewrite query).
