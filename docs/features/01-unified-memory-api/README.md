# Feature 01 — Unified Memory API

> **Loại:** Core Platform | **Priority:** P0 | **Status:** Implemented

## Mô tả

Unified Memory API là lớp abstraction trung tâm của VNP Memory. Thay vì buộc developer phải biết engine nào xử lý loại memory nào, API này cung cấp 4 endpoints thống nhất — `store`, `recall`, `forget`, `timeline` — và tự động route request tới đúng engine phía sau.

Đây là điểm vào duy nhất cho AI Agent muốn tích hợp bộ nhớ mà không cần hiểu kiến trúc nội bộ.

---

## Business Logic

### Store — Lưu Memory

Khi AI Agent gọi `POST /v1/memory/store`:

1. **Type Detection**: Nếu `type=auto`, gateway gọi LLM để phân loại nội dung, xác định loại memory phù hợp nhất.
2. **Auto-routing**: Dựa vào `type` trong request, gateway route tới engine tương ứng:
   - `semantic` → Cognee (knowledge extraction pipeline)
   - `episodic` → Graphiti (temporal fact ingestion)
   - `conversational` → Zep (session memory)
   - `profile` → Memobase (blob buffer, auto-flush tại 20 blobs)
   - `procedural` → OpenViking (VikingFS)
   - `adaptive` → Supermemory (living knowledge graph)
3. **Non-blocking**: Xử lý background, trả về ngay `202 Accepted`.
4. **Event**: NATS publish `memory.blob.inserted` sau khi lưu thành công.

### Recall — Truy xuất Cross-Engine

Khi AI Agent cần context liên quan:

1. Gateway nhận query text + optional filter (type, time range, engine).
2. Delegate tới `vnp-search-hub`.
3. Search Hub fan-out song song tới tất cả 6 engine search services.
4. Mỗi engine trả về kết quả dạng scored list.
5. Search Hub merge + rerank theo relevance score.
6. Trả về top-K kết quả tổng hợp trong `< 500ms` (p95).

### Forget — Xóa Memory

Xóa cascading trên tất cả engines:

1. Nhận `user_id` hoặc `memory_id` cần xóa.
2. Gửi delete request song song tới tất cả 6 engines.
3. Mỗi engine xóa dữ liệu liên quan trong storage của mình.
4. Ghi audit log cho deletion event.
5. Đây là foundation cho GDPR forget flow.

### Timeline — Temporal Events

Truy vấn lịch sử sự kiện:

1. Gọi `vnp-event` service với filter (user, tenant, engine, event type).
2. Trả về danh sách `UserEvent` sorted by `CreatedAt`.
3. Mỗi event có `GistText` — AI-generated summary mô tả event đó.

---

## Dataflow

```
AI Agent
    │
    ▼
POST /v1/memory/store  ──►  Gateway (type detection)
                                │
                    ┌───────────┼───────────┬───────────┬───────────┬──────────────┐
                    ▼           ▼           ▼           ▼           ▼              ▼
               cognee-      graphiti-    zep-        memobase-   ov-fs         sm-memory
               ingestion    ingestion    memory      ingestion   (VikingFS)    (adaptive KG)
                    │           │           │           │           │              │
                    └───────────┴───────────┴───────────┴───────────┴──────────────┘
                                                    │
                                                    ▼
                                            NATS JetStream
                                        (memory.blob.inserted)


POST /v1/memory/recall  ──►  Gateway
                                │
                                ▼
                          vnp-search-hub
                (parallel gRPC fan-out, timeout 500ms)
                                │
            ┌───────────────────┼───────────────────┐
            ▼                   ▼                   ▼
      cognee-search       graphiti-search       sm-search
      memobase-context    ov-search             zep-search
            │                   │                   │
            └───────────────────┴───────────────────┘
                                │
                            Merge + Rerank
                                │
                            Top-K Results
                                │
                          AI Agent Response


GET /v1/memory/timeline  ──►  Gateway
                                │
                                ▼
                          vnp-event service
                     (query UserEvent by user/tenant/engine)
                                │
                    Sorted Timeline (CreatedAt DESC)
                                │
                          AI Agent Response
```

---

## Routing Rules

| Memory Type | Engine | Endpoint phía sau |
|-------------|--------|-------------------|
| `semantic` | Cognee | `cognee-ingestion` gRPC |
| `episodic` | Graphiti | `graphiti-ingestion` gRPC |
| `conversational` | Zep | `zep-memory` gRPC |
| `profile` | Memobase | `memobase-ingestion` gRPC |
| `procedural` | OpenViking | `ov-fs` gRPC |
| `adaptive` | Supermemory | `sm-memory` gRPC |
| `auto` | LLM classify → route | Determined at runtime |

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/memory/store` | Lưu memory (auto-route by type) |
| `POST` | `/v1/memory/recall` | Cross-engine recall |
| `POST` | `/v1/memory/forget` | Cascading delete |
| `GET` | `/v1/memory/timeline` | Temporal event timeline |

---

## Performance Targets

| Metric | Target |
|--------|--------|
| Store response | `< 50ms` (non-blocking, returns immediately) |
| Cross-engine recall (p95) | `< 500ms` |
| Timeline query (p95) | `< 200ms` |

---

## Business Value

### Pain Points được giải quyết

- **PP-P1-02 (Memory fragmented)**
- **PP-P6-01 (No standard API)**

### Actors hưởng lợi

P1 AI Agent Developer, P6 Framework Integrator, P2 Platform Engineer

### Giải pháp tham chiếu

- [S2 — Unified Memory API](../../bussiness/solutions/S2-unified-api.md)
- [S10 — Zero-config Infrastructure](../../bussiness/solutions/S10-infrastructure-simplicity.md)

### ROI / Kết quả đo được

> 6 APIs → 1 API | 500 LOC → 20 LOC | 3 tháng → 5 phút setup

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
