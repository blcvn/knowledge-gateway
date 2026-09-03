# Feature 06 — Procedural Memory (OpenViking)

> **Loại:** Memory Engine | **Priority:** P0 | **Status:** Implemented

## Mô tả

OpenViking quản lý **Procedural Memory** — bộ nhớ cấu trúc dạng filesystem và context phân tầng. Đặc trưng là **VikingFS** (Go-native virtual filesystem) và hệ thống **Tiered Context L0/L1/L2** cho phép load context theo độ sâu cần thiết — từ summary ngắn gọn đến full detail — tối ưu token budget.

OpenViking đặc biệt hữu ích cho AI coding assistants cần đọc/ghi files, index codebase, và duy trì session working memory.

---

## Business Logic

### VikingFS — Virtual Filesystem

VikingFS là một virtual filesystem được implement hoàn toàn bằng Go, lưu trữ files trong PostgreSQL (metadata) và MinIO (content). Developer tương tác như một filesystem thực sự:

- Read/Write/Delete files theo path
- List directory tree
- Grep content trong files

Mọi file operation đều được namespaced theo `TenantID` + `UserID`, đảm bảo isolation.

### Tiered Context (L0/L1/L2)

Thay vì load toàn bộ file content (tốn tokens), OpenViking cung cấp 3 tầng context:

| Tier | Size | Nội dung | Use case |
|------|------|---------|----------|
| **L0** | ~100 tokens | One-sentence summary | Quick overview, navigation |
| **L1** | ~2K tokens | Core info + usage scenarios | Normal context assembly |
| **L2** | Full content | Deep detail | When AI needs complete info |

AI agent bắt đầu với L0 để "nhìn" tổng quan, sau đó load L1/L2 khi cần sâu hơn. Approach này giúp giảm token consumption đáng kể.

### Session Management (2-Phase Commit)

OpenViking có session model riêng để track working context trong một phiên làm việc:

1. **Create Session**: Tạo session với structured document (title, state, goals, facts, errors).
2. **Add Messages**: Thêm messages/observations vào session.
3. **Commit**: 2-phase commit — archive session và extract learnings về long-term memory.

Working Memory trong session giúp AI agent biết: đang làm gì, đã làm gì, gặp lỗi gì.

### Resource Ingestion

OpenViking có thể ingest resources từ external sources:
- **Git repos**: Clone và index codebase
- **HTTP URLs**: Fetch và store web content
- **Local files**: Direct file upload

Sau khi ingest, resources được index cho cả semantic search và grep.

---

## Dataflow

### File Operations Flow

```
GET /v1/ov/files/{path}
        │
        ▼
ov-fs service
        │
        ├── Lookup metadata in PostgreSQL (by TenantID + path)
        ├── Load content from MinIO (by StorageKey)
        └── Return: file content + metadata (size, created_at, tier)


PUT /v1/ov/files/{path}
        │
        ▼
ov-fs service
        │
        ├── 1. Determine tier based on content size
        ├── 2. Generate L0 summary (if new file)
        │         └── LLM: one-sentence summary
        ├── 3. Generate L1 overview (if new file)
        │         └── LLM: core info + usage scenarios
        ├── 4. Store full content → MinIO
        ├── 5. Store metadata → PostgreSQL
        └── 6. Index for search (embedding + BM25)
```

### Tiered Search Flow

```
POST /v1/ov/search
        │
        ├── Input: {query: "...", tier: "L0|L1|L2", path_prefix: "..."}
        │
        ▼
ov-search service
        │
        ├── Semantic search on embeddings (tier-filtered)
        ├── Keyword search (BM25) on stored content
        └── Return results at requested tier
                │
                ├── tier=L0 → Return only summaries (~100 tok each)
                ├── tier=L1 → Return overviews (~2K tok each)
                └── tier=L2 → Return full content (load from MinIO)
```

### Session Flow (2-Phase Commit)

```
POST /v1/ov/sessions                → Create session
        │
        ▼
ov-session service
        │
        ├── Init WorkingMemory document:
        │         {title, state: "active", goals: [], facts: [], errors: []}
        │
        ▼
POST /v1/ov/sessions/{id}/messages  → Add messages during work
        │
        └── Append to session message log + update working memory state
        │
        ▼
POST /v1/ov/sessions/{id}/commit    → 2-phase commit
        │
        ├── Phase 1: Archive session
        │         └── Compress + store full session to MinIO
        │
        └── Phase 2: Extract learnings
                  └── Write important learnings to long-term VikingFS
                      (L0/L1/L2 summaries generated)
```

### Resource Ingestion Flow

```
POST /v1/ov/resources/ingest
        │
        ├── source_type: "git" | "http" | "local"
        │
        ├── git  → Clone repo → Walk files → Ingest each file
        ├── http → Fetch URL → Parse content → Ingest as document
        └── local → Direct upload
                │
                ▼
        ov-resource service
                │
                ├── Store files in VikingFS
                └── Build search indexes (semantic + keyword)
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/ov/files/{path}` | Read file |
| `PUT` | `/v1/ov/files/{path}` | Write file |
| `DELETE` | `/v1/ov/files/{path}` | Delete file |
| `GET` | `/v1/ov/tree/{path}` | List directory tree |
| `POST` | `/v1/ov/grep` | Grep content |
| `POST` | `/v1/ov/search` | Semantic + keyword search |
| `POST` | `/v1/ov/sessions` | Create session |
| `POST` | `/v1/ov/sessions/{id}/messages` | Add messages |
| `POST` | `/v1/ov/sessions/{id}/commit` | 2-phase commit |
| `POST` | `/v1/ov/resources/ingest` | Ingest Git/HTTP/local resources |

---

## Services

| Service | Vai trò |
|---------|---------|
| `ov-fs` | VikingFS CRUD operations |
| `ov-search` | Tiered search (L0/L1/L2), semantic + keyword |
| `ov-session` | Session management, 2-phase commit |
| `ov-resource` | External resource ingestion |
| `ov-crypto` | File encryption at rest |
| `ov-admin` | Admin APIs, storage management |
| `ov-storage` | Low-level MinIO integration |

---

## Business Value

### Pain Points được giải quyết

- **PP-P5-01 (AI quên project context)**
- **PP-P5-02 (Không tìm code cũ)**
- **PP-P5-04 (Re-read files)**

### Actors hưởng lợi

P1 AI Agent Developer, P5 IDE Plugin User

### Giải pháp tham chiếu

- [S1 — Persistent Memory Layer](../../bussiness/solutions/S1-persistent-memory.md)
- [S6 — Smart Context Assembly](../../bussiness/solutions/S6-context-efficiency.md)

### ROI / Kết quả đo được

> L0/L1/L2 tiered loading giảm 60-80% token | Semantic grep không chỉ string match

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
