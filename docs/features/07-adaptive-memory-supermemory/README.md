# Feature 07 — Adaptive Memory (Supermemory)

> **Loại:** Memory Engine | **Priority:** P0 | **Status:** Implemented

## Mô tả

Supermemory quản lý **Adaptive Memory** — bộ nhớ thích nghi với living knowledge graph. Đặc điểm nổi bật:

1. **Auto-forgetting**: Memory có vòng đời (`forgetAfter`), tự động expired khi không còn relevant.
2. **Contradiction Resolution**: Khi memory mới mâu thuẫn với memory cũ, hệ thống tự động đánh dấu memory cũ là outdated.
3. **Memory Versioning**: Mỗi memory có chain `parent → root`, cho phép trace history đầy đủ.
4. **External Connectors**: Tự động sync dữ liệu từ Google Drive, Notion, GitHub, Gmail.
5. **RAG**: Tích hợp Retrieval-Augmented Generation trực tiếp.

---

## Business Logic

### Memory Versioning & Auto-forgetting

Khi AI Agent lưu một memory mới:

1. Supermemory kiểm tra xem có memory nào hiện có mâu thuẫn hoặc liên quan không (via similarity check).
2. Nếu tìm thấy:
   - Tạo memory mới với `parentID` = ID của memory cũ.
   - Đánh dấu memory cũ `isLatest = false`.
   - Relation type: `updates` / `extends` / `derives`.
3. Memory mới trở thành `isLatest = true`.
4. Toàn bộ chain `memory → parent → root` được preserve → full audit trail.

**forgetAfter**: Developer có thể set duration (e.g., "30 days"). Sau thời gian này, memory bị đánh dấu là expired và không xuất hiện trong search results.

### Static vs Dynamic Memory

- **Static**: Fact ổn định theo thời gian (e.g., "Tên công ty là Acme Corp").
- **Dynamic**: Fact thay đổi thường xuyên (e.g., "Project status: In Progress → Done").

Dynamic memories được kiểm tra conflict thường xuyên hơn.

### External Connectors

Supermemory có thể sync từ:
- **Google Drive**: Documents, spreadsheets
- **Gmail**: Email threads
- **Notion**: Pages, databases
- **OneDrive**: Microsoft documents
- **GitHub**: Issues, PRs, code

Sau khi connect (`POST /v1/sm/connections`), user có thể trigger sync (`POST /v1/sm/connections/{id}/sync`). Content được processed và added vào adaptive KG.

### User Profile (Supermemory)

Supermemory builds user profile từ interaction patterns — khác với Memobase (conversation-focused), SM profile dựa trên content preferences và behavior patterns.

### RAG Integration

`POST /v1/sm/rag` nhận query → tìm kiếm relevant memories → format thành context → return augmented response. Cho phép AI trả lời câu hỏi based on stored knowledge.

---

## Dataflow

### Memory Store Flow

```
POST /v1/sm/memories
        │
        ├── Input: {content: "...", type: "static|dynamic", forgetAfter: "30d"}
        │
        ▼
sm-memory service
        │
        ├── 1. Embed content → vector
        │
        ├── 2. Similarity check against existing memories
        │         └── "Do any existing memories conflict or relate?"
        │
        ├── 3. Contradiction/Extension check
        │         ├── CONFLICT found →
        │         │         ├── Set old memory: isLatest = false
        │         │         ├── Create new memory: parentID = old.ID
        │         │         └── relation_type = "updates"
        │         ├── EXTENSION found →
        │         │         └── Create new: parentID = related.ID, type = "extends"
        │         └── NEW →
        │                   └── Create fresh memory (no parent)
        │
        ├── 4. Set forgetAfter schedule (if provided)
        │         └── Schedule background job: expire at (now + duration)
        │
        └── 5. Store to PostgreSQL (metadata) + pgvector (embedding)
```

### Memory Version Chain

```
Memory Chain Example:

Root Memory (Jan 1)
    └── isLatest: false
    └── "Project status: Planning"

    Child Memory (Feb 1) — extends root
        └── isLatest: false
        └── "Project status: In Progress"

        Grandchild Memory (Mar 1) — updates parent
            └── isLatest: true  ← current truth
            └── "Project status: Complete"

Query: GET /v1/console/adaptive/memories/{id}/versions
→ Returns full chain: Root → Child → Grandchild
```

### External Connector Sync Flow

```
POST /v1/sm/connections
        │
        ├── Input: {type: "google_drive", credentials: {...}}
        │
        ▼
sm-connector service
        │
        ├── Validate & store credentials (encrypted)
        └── Return connection ID


POST /v1/sm/connections/{id}/sync
        │
        ▼
sm-connector service
        │
        ├── Fetch new/updated items from source
        ├── Parse content (PDF, DOCX, HTML...)
        └── For each item:
                  └── POST /v1/sm/documents → Document ingestion pipeline
                            → Embed + store in adaptive KG
```

### RAG Flow

```
POST /v1/sm/rag
        │
        ├── Input: {query: "...", max_memories: N, format: "..."}
        │
        ▼
sm-search service
        │
        ├── Semantic search: find relevant memories
        ├── Filter: isLatest=true, not expired
        └── Format memories as context string
                │
                ▼
        Return: {answer_context: "...", sources: [...], memories_used: N}
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/sm/documents` | Ingest document |
| `GET` | `/v1/sm/documents/{id}` | Lấy document |
| `POST` | `/v1/sm/memories` | Lưu memory (adaptive KG) |
| `POST` | `/v1/sm/search` | Hybrid search |
| `POST` | `/v1/sm/rag` | RAG query |
| `GET` | `/v1/sm/profiles/{uid}` | Lấy user profile |
| `POST` | `/v1/sm/connections` | Tạo external connector |
| `POST` | `/v1/sm/connections/{id}/sync` | Trigger sync |
| `POST` | `/v1/sm/projects/spaces` | Tạo project space |

---

## Services

| Service | Vai trò |
|---------|---------|
| `sm-document` | Document ingestion, chunking |
| `sm-memory` | Memory CRUD, versioning, conflict resolution |
| `sm-search` | Hybrid search, RAG |
| `sm-profile` | User profile management |
| `sm-connector` | External data connectors |
| `sm-mcp` | MCP tools for SM |
| `sm-auth` | Auth & RBAC for SM |
| `sm-analytics` | Usage analytics, cost tracking |
| `sm-project` | Project space management |
| `sm-engine` | Core adaptive KG engine |
