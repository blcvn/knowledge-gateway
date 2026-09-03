# Feature 02 — Episodic Memory (Graphiti)

> **Loại:** Memory Engine | **Priority:** P0 | **Status:** Implemented

## Mô tả

Graphiti là engine quản lý **Episodic Memory** — bộ nhớ sự kiện theo thời gian. Không giống vector DB thông thường chỉ lưu embedding, Graphiti xây dựng một **temporal knowledge graph** trong đó mỗi fact có cửa sổ hiệu lực (`valid_at` / `invalid_at`). Điều này cho phép AI thực hiện **temporal reasoning**: biết điều gì đúng vào thời điểm nào, fact nào đã hết hạn, fact nào mâu thuẫn với fact khác.

---

## Business Logic

### Episode Ingestion

Một "episode" là một đơn vị thông tin có ngữ cảnh thời gian (conversation turn, event, observation). Khi ingest:

1. **Receive Episode**: Text hoặc JSON hoặc fact_triple được nhận vào.
2. **LLM Extraction**: LLM trích xuất entities và relationships từ raw content.
3. **Deduplication**: So sánh entities/edges với graph hiện có. Nếu entity đã tồn tại, cập nhật thay vì tạo mới.
4. **Temporal Validity**: Mỗi edge được gán `valid_at` (thời điểm bắt đầu hiệu lực). Khi một fact bị thay thế, edge cũ được đánh dấu `invalid_at`.
5. **Graph Write**: Nodes và edges được persist vào Neo4j.
6. **Event Log**: NATS publish event để các service khác biết graph đã thay đổi.

### Temporal Fact Management

Mỗi fact trong Graphiti có vòng đời:
- **Active**: `valid_at <= now` và `invalid_at IS NULL` → fact hiện tại đúng
- **Expired**: `invalid_at <= now` → fact đã hết hiệu lực
- **Future**: `valid_at > now` → fact chưa có hiệu lực

Khi có thông tin mới mâu thuẫn, Graphiti:
1. Set `invalid_at` cho fact cũ = thời điểm nhận fact mới.
2. Insert fact mới với `valid_at` mới.
3. Maintain provenance (ai tạo, từ episode nào).

### Hybrid Search

Graphiti search kết hợp 3 strategies:
1. **Semantic Search**: Vector similarity trên node/edge embeddings.
2. **BM25**: Keyword matching cho entity names và fact text.
3. **Graph Traversal**: Traverse từ matched nodes ra neighbors, theo temporal constraints.
4. **Reranking**: Kết hợp scores theo weighted formula.

---

## Dataflow

### Episode Ingestion Flow

```
POST /v1/graphiti/episodes
        │
        ▼
graphiti-ingestion service
        │
        ├── 1. Parse input (text | JSON | fact_triple)
        │
        ├── 2. LLM Entity Extraction
        │         └── Extract: entities, relationships, temporal markers
        │
        ├── 3. Deduplication Check
        │         └── Query Neo4j: "Does entity X already exist?"
        │                  ├── YES → merge/update existing node
        │                  └── NO  → create new node
        │
        ├── 4. Temporal Edge Creation
        │         ├── Set valid_at = current timestamp (or extracted time)
        │         └── Invalidate conflicting facts (set invalid_at)
        │
        ├── 5. Neo4j Write (ACID transaction)
        │         ├── MERGE nodes
        │         └── CREATE edges with temporal properties
        │
        └── 6. NATS publish: graphiti.episode.ingested
```

### Search Flow

```
POST /v1/graphiti/search
        │
        ▼
graphiti-search service
        │
        ├── Semantic Search (pgvector/Qdrant)
        │         └── Find nodes/edges by embedding similarity
        │
        ├── BM25 Search
        │         └── Keyword match on node names + fact text
        │
        ├── Graph Traversal
        │         └── Expand from matched nodes → neighbors (depth 1-3)
        │                  └── Filter by temporal constraints (valid_at/invalid_at)
        │
        └── Rerank & Merge
                  └── Weighted score combination → Top-K results
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/graphiti/episodes` | Ingest episode (text/JSON/fact_triple) |
| `POST` | `/v1/graphiti/search` | Hybrid search trên temporal graph |
| `GET` | `/v1/graphiti/nodes/{id}` | Lấy node chi tiết |
| `GET` | `/v1/graphiti/edges/{id}` | Lấy edge và temporal validity |

---

## Services

| Service | Vai trò |
|---------|---------|
| `graphiti-ingestion` | Nhận episodes, LLM extraction, write graph |
| `graphiti-search` | Hybrid search (semantic + BM25 + traversal) |
| `graphiti-store` | Neo4j read/write operations |
| `graphiti-knowledge` | Knowledge graph management APIs |
| `graphiti-pipeline` | Background pipeline jobs |
| `graphiti-admin` | Admin APIs, observability |

---

## Storage

- **Neo4j 5+**: Primary graph storage (nodes, edges, temporal properties)
- **pgvector / Qdrant**: Vector embeddings for semantic search
- **PostgreSQL**: Metadata, provenance, audit records
