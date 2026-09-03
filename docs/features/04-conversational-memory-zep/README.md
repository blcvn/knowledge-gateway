# Feature 04 — Conversational Memory (Zep)

> **Loại:** Memory Engine | **Priority:** P0 | **Status:** Implemented

## Mô tả

Zep quản lý **Conversational Memory** — bộ nhớ hội thoại theo phiên làm việc. Zep đặc biệt ở chỗ nó kết hợp session memory truyền thống với một **knowledge graph** được xây dựng tự động từ conversations. Điều này cho phép AI không chỉ nhớ nội dung cuộc trò chuyện mà còn hiểu được entities và relationships được đề cập.

Mục tiêu: Context assembly `< 200ms` (p95).

---

## Business Logic

### User & Session Management

- **User**: Thực thể đại diện cho người dùng, có thể có nhiều sessions.
- **Session**: Một cuộc hội thoại cụ thể. Session lưu toàn bộ message history.
- **Memory**: Compressed representation của session — thay vì giữ tất cả messages, Zep tóm tắt và extract key information.

### Memory Ingestion

Khi developer push messages vào session (`PUT /v1/zep/sessions/{id}/memory`):

1. Messages được append vào session message history.
2. Background process phân tích message mới:
   - **Entity Extraction**: Trích xuất entities được nhắc đến (tên người, địa điểm, sản phẩm...).
   - **Relationship Extraction**: Xác định relationships giữa entities.
   - **Graph Update**: Cập nhật Zep's internal knowledge graph.
   - **Summary Generation**: Tạo summary cho đoạn conversation mới.

### Graph-based Retrieval

Zep có knowledge graph riêng (dạng property graph), cho phép:
- **Graph Search**: Tìm entities và facts liên quan đến query.
- **Session Search**: Tìm trong message history của session cụ thể.
- **Facts Management**: Thêm explicit facts vào graph (`POST /v1/zep/graph/facts`).

### Custom Ontology

ML Engineer có thể định nghĩa custom ontology cho domain cụ thể:
- Các entity types domain-specific (e.g., "MedicalCondition", "DrugName" cho healthcare).
- Relationship types domain-specific.
- Constraints giữa entity types.

---

## Dataflow

### Session Memory Flow

```
PUT /v1/zep/sessions/{id}/memory
        │
        ├── Input: [{role: "user"|"assistant", content: "..."}]
        │
        ▼
zep-memory service
        │
        ├── 1. Append messages to session store (PostgreSQL)
        │
        ├── 2. Background: Graph Ingestion
        │         └── zep-graph service
        │                  ├── Entity extraction (LLM)
        │                  ├── Relationship extraction (LLM)
        │                  └── Update property graph (Neo4j)
        │
        ├── 3. Background: Summary Generation
        │         └── Compress old messages → summary text
        │                  └── Store summary in session metadata
        │
        └── 4. Context Assembly (for retrieval)
                  ├── Recent messages (raw)
                  ├── Older messages (summarized)
                  └── Relevant graph facts
```

### Graph Search Flow

```
POST /v1/zep/graph/search
        │
        ├── Input: {query: "...", scope: "edges|nodes", limit: N}
        │
        ▼
zep-graph service
        │
        ├── Embed query → vector similarity search on graph nodes/edges
        ├── BM25 keyword search on entity names and fact text
        └── Merge results → ranked list
                │
                ▼
        Return: [entities, relationships, facts]
```

### Context Assembly

```
GET /v1/zep/sessions/{id}/memory
        │
        ▼
zep-memory service
        │
        ├── Load recent messages (last N turns)
        ├── Load session summary (compressed history)
        ├── Load relevant graph facts (via zep-graph)
        └── Assemble context string (< 200ms)
                │
                ▼
        Return: {messages: [...], summary: "...", facts: [...]}
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/zep/users` | Tạo user mới |
| `GET` | `/v1/zep/users/{id}` | Lấy thông tin user |
| `PATCH` | `/v1/zep/users/{id}` | Cập nhật user metadata |
| `PUT` | `/v1/zep/sessions/{id}/memory` | Push messages vào session |
| `GET` | `/v1/zep/sessions/{id}/memory` | Lấy assembled context |
| `POST` | `/v1/zep/sessions/{id}/search` | Tìm kiếm trong session |
| `POST` | `/v1/zep/graph/search` | Search trên knowledge graph |
| `POST` | `/v1/zep/graph/facts` | Thêm explicit facts |
| `POST` | `/v1/zep/graph/ontology` | Set custom ontology |

---

## Services

| Service | Vai trò |
|---------|---------|
| `zep-user` | User management |
| `zep-thread` | Thread/session lifecycle |
| `zep-memory` | Session memory, context assembly |
| `zep-graph` | Knowledge graph operations |
| `zep-search` | Semantic + graph search |
| `zep-admin` | Admin APIs, tenant management |

---

## Business Value

### Pain Points được giải quyết

- **PP-P1-01 (Agent mất context)**
- **PP-P3-01 (Generic ontology)**
- **PP-P1-03 (No temporal)**

### Actors hưởng lợi

P1 AI Agent Developer, P3 ML/AI Engineer

### Giải pháp tham chiếu

- [S1 — Persistent Memory Layer](../../bussiness/solutions/S1-persistent-memory.md)
- [S3 — Temporal Reasoning](../../bussiness/solutions/S3-temporal-reasoning.md)

### ROI / Kết quả đo được

> Session context persist cross-session | Custom ontology cho domain-specific entities | Graph RAG < 200ms

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
