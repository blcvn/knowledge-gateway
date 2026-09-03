# Feature 03 — Semantic Memory (Cognee)

> **Loại:** Memory Engine | **Priority:** P0 | **Status:** Implemented

## Mô tả

Cognee là engine quản lý **Semantic Memory** — bộ nhớ tri thức có cấu trúc. Cognee nhận đầu vào đa dạng (PDF, text, audio, image, URL, CSV...) và chạy qua một pipeline 7 bước để trích xuất entities, relationships, và xây dựng knowledge graph. Search hỗ trợ 15+ strategies từ vector similarity đến graph completion.

---

## Business Logic

### Dataset Management

Trước khi ingest data, user tạo một **Dataset** — namespace logic để nhóm dữ liệu liên quan. Một tenant có thể có nhiều datasets (e.g., "product-docs", "support-tickets", "codebase").

### Cognify Pipeline (7 bước)

Khi user trigger `POST /v1/cognee/datasets/{id}/cognify`:

1. **Chunking**: Chia document thành các chunks phù hợp với context window của LLM.
2. **Embedding**: Tạo vector embeddings cho mỗi chunk (lưu vào pgvector/Qdrant).
3. **Entity Extraction**: LLM trích xuất entities (người, tổ chức, concept, sự kiện).
4. **Relationship Extraction**: LLM xác định relationships giữa entities.
5. **Deduplication**: Merge entities trùng lặp với entities đã tồn tại trong graph.
6. **Graph Construction**: Write nodes và edges vào Neo4j knowledge graph.
7. **Indexing**: Build search indexes cho BM25 và semantic search.

### Memify (Non-destructive Enrichment) — CR-COGNEE-001

Khác với `cognify` (xây dựng lại graph), `memify` là **enrichment non-destructive**: chỉ thêm thông tin mới, không xóa existing graph. Hữu ích khi muốn bổ sung context mà không rebuild toàn bộ dataset.

### DataPoint Schema Ingestion — CR-COGNEE-003

Cho phép ingest data theo structured schema được define sẵn — **không cần LLM calls** để extract. Phù hợp cho structured data (CSV, API responses) nơi schema đã biết trước.

### Multi-Strategy Search

Cognee search hỗ trợ 15+ strategies:
- `CHUNKS`: Raw chunk retrieval
- `SUMMARIES`: Document summaries
- `INSIGHTS`: LLM-extracted insights
- `GRAPH_COMPLETION`: Complete query thành graph path
- `CODE`: Code-specific retrieval
- `HYBRID`: Combine multiple strategies

---

## Dataflow

### Data Upload & Cognify Flow

```
POST /v1/cognee/datasets          → Create dataset (namespace)
        │
        ▼
POST /v1/cognee/datasets/{id}/data → Upload file(s)
        │
        ├── Supported formats: PDF, DOCX, TXT, CSV, URL, audio, image
        └── Stored in: MinIO (object storage) or local filesystem
        │
        ▼
POST /v1/cognee/datasets/{id}/cognify → Trigger pipeline
        │
        ▼
cognee-cognify service
        │
        ├── Step 1: Chunking
        │         └── Chunk by token count (overlap configurable)
        │
        ├── Step 2: Embedding
        │         └── Batch embed chunks → pgvector / Qdrant
        │
        ├── Step 3: Entity Extraction (LLM)
        │         └── Extract: Person, Org, Concept, Event, Location
        │
        ├── Step 4: Relationship Extraction (LLM)
        │         └── Extract: IS_A, RELATES_TO, PART_OF, CAUSED_BY...
        │
        ├── Step 5: Deduplication
        │         └── Fuzzy match existing nodes → MERGE or CREATE
        │
        ├── Step 6: Graph Construction
        │         └── Write to Neo4j (nodes + edges with properties)
        │
        └── Step 7: Index Build
                  └── BM25 index + vector index update
                                │
                                ▼
                    cognee-pipeline service
                    (manages job status, retry, queue)
```

### Search Flow

```
POST /v1/cognee/search
        │
        ├── strategy = "CHUNKS"         → Vector similarity search (pgvector)
        ├── strategy = "GRAPH_COMPLETION" → Graph traversal + LLM completion
        ├── strategy = "SUMMARIES"      → Pre-computed summary retrieval
        ├── strategy = "INSIGHTS"       → LLM-extracted insight retrieval
        └── strategy = "HYBRID"         → Combine multiple strategies
                │
                ▼
        cognee-search service
                │
        Merge + Score → Top-K results
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/cognee/datasets` | Tạo dataset mới |
| `POST` | `/v1/cognee/datasets/{id}/data` | Upload data (multi-modal) |
| `POST` | `/v1/cognee/datasets/{id}/cognify` | Trigger 7-step pipeline |
| `POST` | `/v1/cognee/datasets/{id}/memify` | Non-destructive enrichment |
| `GET` | `/v1/cognee/datasets/{id}/memify/status` | Trạng thái memify job |
| `POST` | `/v1/cognee/datasets/{id}/datapoints` | Schema-based ingestion (zero LLM) |
| `POST` | `/v1/cognee/search` | Multi-strategy search (15+ strategies) |

---

## Services

| Service | Vai trò |
|---------|---------|
| `cognee-ingestion` | Nhận data upload, chunking, routing |
| `cognee-cognify` | 7-step pipeline execution |
| `cognee-search` | Multi-strategy search |
| `cognee-pipeline` | Job management, queue, retry |

---

## Business Value

### Pain Points được giải quyết

- **PP-P1-02 (Memory fragmented)**
- **PP-P3-01 (Generic ontology)**

### Actors hưởng lợi

P1 AI Agent Developer, P3 ML/AI Engineer

### Giải pháp tham chiếu

- [S2 — Unified Memory API](../../bussiness/solutions/S2-unified-api.md)

### ROI / Kết quả đo được

> 15+ search strategies | Multi-modal ingestion (PDF, audio, image, CSV, URL)

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
