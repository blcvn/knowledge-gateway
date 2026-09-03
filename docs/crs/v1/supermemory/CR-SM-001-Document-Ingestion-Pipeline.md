# Change Request: CR-SM-001 — Document Ingestion Pipeline

**CR ID:** CR-SM-001  
**Component:** `services/document-service` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** Supermemory PRD §3.4, SRS §2.1–2.2, specs/services/02-document-service.md  
**Maps to existing code:** VNP Memory hiện có ingest cơ bản, cần nâng cấp full pipeline với 11 content types và async worker

---

## 1. Mô tả

Xây dựng **Document Service** — đường ống xử lý tài liệu (Ingestion Pipeline) đa định dạng cho VNP Memory:

1. **CRUD đầy đủ**: Tạo, đọc, danh sách, xóa (đơn + hàng loạt) tài liệu.
2. **11 Content Types**: `text`, `pdf`, `tweet`, `google_doc`, `google_slide`, `google_sheet`, `image`, `video`, `notion_doc`, `webpage`, `onedrive`.
3. **Async Processing Pipeline**: `queued → extracting → chunking → embedding → indexing → done`.
4. **Content Deduplication**: SHA-256 content hash để tránh xử lý trùng lặp.
5. **AST-aware Chunking**: Chunking code theo cấu trúc AST (go-tree-sitter), semantic chunking cho văn bản.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện tại chưa có pipeline xử lý nội dung hoàn chỉnh với trạng thái.
- Thiếu hỗ trợ OCR (ảnh), transcription (video), parser PDF/DOCX đa dạng.
- Chưa có content deduplication (contentHash) để tránh ingest trùng.
- Thiếu Matryoshka embeddings (vector có thể cắt ngắn theo chiều dài để tiết kiệm chi phí embedding).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/document-service/` (Port gRPC: 9001)

Tạo service mới với Clean Architecture 4 lớp (domain → usecase → adapter → infra).

### 3.2. Domain Model

```go
type Document struct {
    ID             string
    CustomID       *string          // User-defined dedup ID
    ContentHash    *string          // SHA-256 của content
    OrgID          string
    UserID         string
    ConnectionID   *string          // Nếu được import từ Connector
    Title, Content, Summary *string
    URL            *string
    Type           DocumentType     // text|pdf|image|video|webpage|...
    Status         DocumentStatus   // queued→extracting→chunking→embedding→indexing→done|failed
    Metadata       map[string]any
    ProcessingMeta *ProcessingMetadata
    TokenCount, ChunkCount *int
    SummaryEmbedding []float32
}

type Chunk struct {
    ID              string
    DocumentID      string
    Content         string
    EmbeddedContent *string
    Type            ChunkType      // text | image
    Position        int
    Embedding       []float32
    MatryoshkaEmbed []float32      // Truncatable embedding
}
```

### 3.3. Extractor Registry

| Loại | Cách xử lý |
|------|-----------|
| `text` | Passthrough |
| `pdf` | Go PDF library (pdfcpu/unidoc) |
| `webpage` | HTML → clean text (goquery + trafilatura-style) |
| `image` | OCR qua AI provider (Vision LLM) |
| `video` | Transcription qua AI provider |
| `google_doc/slide/sheet` | Google API export |
| `notion_doc` | Notion API |

### 3.4. Chunker Registry

| Strategy | Dùng cho |
|----------|---------|
| **Semantic** | Văn bản thông thường (split tại ranh giới nghĩa) |
| **AST-aware** | Code files (go-tree-sitter bindings) |
| **Fixed-size** | Fallback cho các format không nhận dạng được |

### 3.5. Async Worker (NATS JetStream)

```
1. CreateDocument → Publish "document.ingest.requested"
2. Worker Pool nhận → chạy pipeline 4 bước (extract, chunk, embed, index)
3. Batch embedding (max 100 chunks/call)
4. Khi xong → Publish "document.processed" (Memory Service nhận để extract facts)
5. Lỗi → status=failed, metadata có thông tin lỗi, retry 3 lần exponential backoff
```

### 3.6. API Endpoints qua Gateway

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/api/v1/documents` | Tạo document (text/URL/file) |
| `POST` | `/api/v1/documents/list` | Liệt kê với pagination + filter |
| `GET` | `/api/v1/documents/:id` | Lấy chi tiết + status |
| `DELETE` | `/api/v1/documents/:id` | Xóa document |
| `DELETE` | `/api/v1/documents/bulk` | Xóa hàng loạt (by IDs hoặc containerTags) |

---

## 4. Acceptance Criteria

- [ ] Upload một file PDF 100 trang → được xử lý và chuyển sang status `done` trong vòng 2 phút.
- [ ] Gửi cùng một nội dung hai lần → lần 2 bị từ chối với lỗi "Duplicate content" (contentHash trùng).
- [ ] Gửi URL ảnh → hệ thống OCR và lưu text vào chunk.
- [ ] File code `.go` được chunked theo cấu trúc hàm/struct (AST-aware), không phải cắt giữa chừng.
- [ ] Status tracking hoạt động: gọi `GET /documents/:id` thấy trạng thái real-time.
- [ ] Bulk delete với 1000 IDs hoàn thành trong vòng 10 giây.
