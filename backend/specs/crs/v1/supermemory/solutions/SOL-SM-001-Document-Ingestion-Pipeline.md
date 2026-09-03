# Solution: SOL-SM-001 — Document Ingestion Pipeline

**CR ID:** CR-SM-001  
**Solution ID:** SOL-SM-001  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/document-service/` theo Clean Architecture 4 lớp, tích hợp vào VNP Memory Monolith qua InProcessRegistry (bufconn). Service này thay thế và mở rộng logic ingest đơn giản hiện có trong `sm-document` thuộc `services/memory-service/`.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu trong codebase hiện tại

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `SMDocument` entity | `services/memory-service/internal/domain/sm/` | Tồn tại nhưng thiếu Status, ContentHash, ProcessingMeta |
| `sm-document` gRPC service | `apps/memory/internal/bootstrap/` | Có, nhưng chỉ hỗ trợ CRUD cơ bản |
| `/v1/sm/*` routes (9 routes) | `gateway/adapter/handler/` | Có, cần bổ sung document pipeline routes |
| NATS JetStream embedded | `apps/memory/internal/bus/` | Sẵn sàng, dùng để publish events |

### Gap phân tích

- Thiếu trạng thái pipeline (`queued → extracting → chunking → embedding → indexing → done`)
- Thiếu Extractor Registry (OCR, PDF, webpage)
- Thiếu AST-aware chunker (`go-tree-sitter`)
- Thiếu SHA-256 content deduplication
- Thiếu Matryoshka embeddings
- Thiếu async worker pool với NATS

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service Mới

```
services/document-service/
├── cmd/                          # Entry point nếu deploy standalone
├── internal/
│   ├── domain/
│   │   ├── document.go           # Document, Chunk, ProcessingMetadata entities
│   │   ├── extractor.go          # Extractor interface + Registry
│   │   ├── chunker.go            # Chunker interface + Registry
│   │   └── repository.go        # Repository ports
│   ├── usecase/
│   │   ├── create_document.go    # Validate + hash + publish NATS
│   │   ├── list_documents.go     # Pagination + filter
│   │   ├── get_document.go       # Lấy status + chunks
│   │   ├── delete_document.go    # Single delete
│   │   └── bulk_delete.go        # Bulk delete by IDs/tags (batch 100)
│   ├── adapter/
│   │   ├── grpc/                 # gRPC server (DocumentService proto)
│   │   └── worker/
│   │       ├── ingestion_worker.go  # NATS consumer: document.ingest.requested
│   │       └── pipeline.go          # Orchestrate: extract → chunk → embed → index
│   └── infra/
│       ├── postgres/             # DocumentRepo, ChunkRepo
│       ├── extractor/
│       │   ├── text.go           # Passthrough
│       │   ├── pdf.go            # pdfcpu integration
│       │   ├── webpage.go        # goquery + HTML cleaner
│       │   ├── image.go          # Vision LLM via Bifrost
│       │   ├── video.go          # Transcription via Bifrost
│       │   ├── google.go         # Google Drive/Docs/Sheets API
│       │   └── notion.go         # Notion API
│       └── chunker/
│           ├── semantic.go       # Semantic chunking (ranh giới nghĩa)
│           ├── ast.go            # go-tree-sitter bindings
│           └── fixed.go          # Fixed-size fallback
```

### 3.2. Domain Model Chi tiết

```go
// services/document-service/internal/domain/document.go

package domain

import "time"

type DocumentType string

const (
    DocTypeText        DocumentType = "text"
    DocTypePDF         DocumentType = "pdf"
    DocTypeTweet       DocumentType = "tweet"
    DocTypeGoogleDoc   DocumentType = "google_doc"
    DocTypeGoogleSlide DocumentType = "google_slide"
    DocTypeGoogleSheet DocumentType = "google_sheet"
    DocTypeImage       DocumentType = "image"
    DocTypeVideo       DocumentType = "video"
    DocTypeNotionDoc   DocumentType = "notion_doc"
    DocTypeWebpage     DocumentType = "webpage"
    DocTypeOneDrive    DocumentType = "onedrive"
)

type DocumentStatus string

const (
    StatusQueued    DocumentStatus = "queued"
    StatusExtracting DocumentStatus = "extracting"
    StatusChunking  DocumentStatus = "chunking"
    StatusEmbedding DocumentStatus = "embedding"
    StatusIndexing  DocumentStatus = "indexing"
    StatusDone      DocumentStatus = "done"
    StatusFailed    DocumentStatus = "failed"
)

type Document struct {
    ID           string
    CustomID     *string        // User-defined dedup ID
    ContentHash  *string        // SHA-256 của content, unique per OrgID
    OrgID        string
    UserID       string
    ConnectionID *string        // Từ Connector Service
    Title        *string
    Content      *string        // Raw content (sau extract)
    Summary      *string        // LLM-generated summary
    URL          *string
    Type         DocumentType
    Status       DocumentStatus
    Metadata     map[string]any
    ProcessingMeta *ProcessingMetadata
    TokenCount   *int
    ChunkCount   *int
    SummaryEmbedding []float32  // Để search trên document level
    ContainerTags []string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type ProcessingMetadata struct {
    StartedAt    *time.Time
    CompletedAt  *time.Time
    ErrorMsg     *string
    RetryCount   int
    ExtractorUsed string
    ChunkerUsed   string
}

type Chunk struct {
    ID              string
    DocumentID      string
    OrgID           string
    Content         string
    EmbeddedContent *string        // Content đưa vào embed (có thể khác raw)
    Type            ChunkType      // text | image
    Position        int            // Thứ tự trong document
    Embedding       []float32      // Dense vector (1536 dims cho text-embedding-3-small)
    MatryoshkaEmbed []float32      // Truncatable 256 dims
    CreatedAt       time.Time
}

type ChunkType string

const (
    ChunkTypeText  ChunkType = "text"
    ChunkTypeImage ChunkType = "image"
)
```

### 3.3. Extractor Interface

```go
// services/document-service/internal/domain/extractor.go

type Extractor interface {
    Extract(ctx context.Context, doc *Document) (content string, err error)
    Supports(docType DocumentType) bool
}

// ExtractorRegistry — chọn extractor phù hợp theo DocumentType
type ExtractorRegistry struct {
    extractors []Extractor
}

func (r *ExtractorRegistry) Get(docType DocumentType) (Extractor, error) {
    for _, e := range r.extractors {
        if e.Supports(docType) {
            return e, nil
        }
    }
    return nil, ErrExtractorNotFound
}
```

### 3.4. Chunker Interface

```go
// services/document-service/internal/domain/chunker.go

type Chunker interface {
    Chunk(ctx context.Context, doc *Document) ([]Chunk, error)
    Supports(docType DocumentType) bool
}

// ChunkerRegistry — ưu tiên AST cho code, Semantic cho text, Fixed fallback
type ChunkerRegistry struct {
    chunkers []Chunker  // Thứ tự: AST → Semantic → Fixed
}
```

### 3.5. Ingestion Worker Pipeline

```go
// services/document-service/internal/adapter/worker/pipeline.go

type IngestionPipeline struct {
    extractors *ExtractorRegistry
    chunkers   *ChunkerRegistry
    embedder   EmbedderPort        // Gọi Bifrost để embed
    docRepo    DocumentRepository
    chunkRepo  ChunkRepository
    publisher  EventPublisher      // NATS
}

func (p *IngestionPipeline) Process(ctx context.Context, docID string) error {
    doc := p.docRepo.Get(ctx, docID)

    // Step 1: Extract
    p.updateStatus(ctx, doc, StatusExtracting)
    extractor, _ := p.extractors.Get(doc.Type)
    content, err := extractor.Extract(ctx, doc)
    if err != nil { return p.handleRetry(ctx, doc, err) }
    doc.Content = &content

    // Step 2: Chunk
    p.updateStatus(ctx, doc, StatusChunking)
    chunker, _ := p.chunkers.Get(doc.Type)
    chunks, err := chunker.Chunk(ctx, doc)
    if err != nil { return p.handleRetry(ctx, doc, err) }

    // Step 3: Embed (batch 100 chunks/call)
    p.updateStatus(ctx, doc, StatusEmbedding)
    for batch := range batchChunks(chunks, 100) {
        p.embedder.BatchEmbed(ctx, batch)
    }
    // Embed summary
    doc.SummaryEmbedding, _ = p.embedder.Embed(ctx, *doc.Summary)

    // Step 4: Index (upsert to postgres)
    p.updateStatus(ctx, doc, StatusIndexing)
    p.chunkRepo.BulkUpsert(ctx, chunks)

    // Done
    p.updateStatus(ctx, doc, StatusDone)
    p.publisher.Publish(ctx, "document.processed", DocumentProcessedEvent{DocID: doc.ID, OrgID: doc.OrgID})
    return nil
}

// Retry 3 lần với exponential backoff (1s, 2s, 4s)
func (p *IngestionPipeline) handleRetry(ctx context.Context, doc *Document, err error) error {
    if doc.ProcessingMeta.RetryCount >= 3 {
        doc.Status = StatusFailed
        doc.ProcessingMeta.ErrorMsg = ptr(err.Error())
        p.docRepo.Update(ctx, doc)
        return err
    }
    delay := time.Duration(1<<doc.ProcessingMeta.RetryCount) * time.Second
    time.Sleep(delay)
    doc.ProcessingMeta.RetryCount++
    p.docRepo.Update(ctx, doc)
    return p.Process(ctx, doc.ID)
}
```

### 3.6. Content Deduplication

```go
// services/document-service/internal/usecase/create_document.go

func (uc *CreateDocumentUseCase) Execute(ctx context.Context, req CreateDocumentRequest) (*Document, error) {
    // 1. Tính ContentHash nếu có content
    if req.Content != "" {
        hash := sha256Hex(req.Content)
        existing, _ := uc.repo.FindByContentHash(ctx, req.OrgID, hash)
        if existing != nil {
            return nil, ErrDuplicateContent // "Duplicate content"
        }
        req.ContentHash = &hash
    }

    // 2. Check CustomID dedup
    if req.CustomID != "" {
        existing, _ := uc.repo.FindByCustomID(ctx, req.OrgID, req.CustomID)
        if existing != nil {
            return nil, ErrDuplicateCustomID
        }
    }

    // 3. Create document với status=queued
    doc := &Document{Status: StatusQueued, ...}
    uc.repo.Create(ctx, doc)

    // 4. Publish NATS event
    uc.publisher.Publish(ctx, "document.ingest.requested", IngestRequestedEvent{DocID: doc.ID})

    return doc, nil
}
```

---

## 4. Database Schema

```sql
-- documents table
CREATE TABLE documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custom_id       TEXT,
    content_hash    TEXT,
    org_id          UUID NOT NULL,
    user_id         UUID NOT NULL,
    connection_id   UUID REFERENCES connections(id) ON DELETE SET NULL,
    title           TEXT,
    content         TEXT,
    summary         TEXT,
    url             TEXT,
    type            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued',
    metadata        JSONB DEFAULT '{}',
    processing_meta JSONB DEFAULT '{}',
    token_count     INT,
    chunk_count     INT,
    summary_embedding vector(1536),
    container_tags  TEXT[] DEFAULT '{}',
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now(),
    UNIQUE (org_id, content_hash),
    UNIQUE (org_id, custom_id)
);

-- Indexes
CREATE INDEX idx_documents_org_status ON documents(org_id, status);
CREATE INDEX idx_documents_container_tags ON documents USING GIN(container_tags);
CREATE INDEX ON documents USING hnsw (summary_embedding vector_cosine_ops);

-- chunks table
CREATE TABLE chunks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id         UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    org_id              UUID NOT NULL,
    content             TEXT NOT NULL,
    embedded_content    TEXT,
    type                TEXT NOT NULL DEFAULT 'text',
    position            INT NOT NULL,
    embedding           vector(1536),
    matryoshka_embedding vector(256),
    created_at          TIMESTAMPTZ DEFAULT now()
);

-- HNSW indexes cho vector search
CREATE INDEX ON chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX ON chunks USING hnsw (matryoshka_embedding vector_cosine_ops);
CREATE INDEX idx_chunks_doc ON chunks(document_id);
```

---

## 5. NATS Events

| Event Subject | Publisher | Subscribers |
|---------------|-----------|-------------|
| `document.ingest.requested` | CreateDocument UC | IngestionWorker |
| `document.processed` | IngestionPipeline | Memory Service (fact extraction) |
| `document.failed` | IngestionPipeline | Analytics Service |
| `document.deleted` | Delete UC | Connector Service, Analytics |

---

## 6. Gateway Integration

### Đăng ký service với InProcessRegistry (Monolith)

```go
// apps/memory/internal/bootstrap/document.go

func initDocumentService(reg *bus.InProcessRegistry, cfg *config.Config) {
    repo := postgres.NewDocumentRepo(cfg.Postgres)
    chunkRepo := postgres.NewChunkRepo(cfg.Postgres)
    extractors := buildExtractorRegistry(cfg)
    chunkers := buildChunkerRegistry()
    embedder := bifrost.NewEmbedder(cfg.Bifrost)
    publisher := nats.NewPublisher(cfg.NATS)

    pipeline := worker.NewIngestionPipeline(extractors, chunkers, embedder, repo, chunkRepo, publisher)
    workerPool := worker.NewPool(pipeline, cfg.DocumentService.Workers) // default 5 workers
    workerPool.Start()

    grpcSrv := grpc.NewDocumentServiceServer(repo, publisher)
    reg.Register("document-service", grpcSrv)
}
```

### REST Handler (thêm vào `/v1/sm/*`)

```go
// gateway/adapter/handler/document_handler.go

func (h *DocumentHandler) Register(mux *http.ServeMux) {
    mux.HandleFunc("POST /api/v1/documents", h.Create)
    mux.HandleFunc("POST /api/v1/documents/list", h.List)
    mux.HandleFunc("GET /api/v1/documents/{id}", h.Get)
    mux.HandleFunc("DELETE /api/v1/documents/{id}", h.Delete)
    mux.HandleFunc("DELETE /api/v1/documents/bulk", h.BulkDelete)
}
```

---

## 7. Dependencies Mới

| Dependency | Package | Mục đích |
|------------|---------|----------|
| PDF extraction | `github.com/pdfcpu/pdfcpu` | Parse PDF → text |
| HTML cleaning | `github.com/PuerkitoBio/goquery` | Webpage → clean text |
| AST chunking | `github.com/smacker/go-tree-sitter` | Code structure chunking |
| Content hash | stdlib `crypto/sha256` | SHA-256 dedup |

---

## 8. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + Repository (postgres) | 2 ngày |
| **P2** | Text + PDF + Webpage extractors | 2 ngày |
| **P3** | Semantic + Fixed chunker | 1 ngày |
| **P4** | AST chunker (go-tree-sitter) | 1 ngày |
| **P5** | Embedding integration (Bifrost) | 1 ngày |
| **P6** | NATS async worker pool | 2 ngày |
| **P7** | Image/Video extractors (Vision LLM) | 2 ngày |
| **P8** | Google + Notion extractors | 2 ngày |
| **P9** | Gateway integration + REST handlers | 1 ngày |
| **P10** | Tests + Acceptance Criteria validation | 2 ngày |

**Tổng:** ~16 ngày (Wave 2)

---

## 9. Rủi ro và Mitigation

| Rủi ro | Mức độ | Mitigation |
|--------|--------|-----------|
| go-tree-sitter có ít maintained Go bindings | Medium | Fallback về Fixed chunker nếu AST thất bại |
| Vision LLM latency cho OCR (~5-10s/ảnh) | High | Xử lý async hoàn toàn, không ảnh hưởng API response |
| PDF phức tạp (form-based, scanned) | Medium | Fallback về Vision LLM nếu pdfcpu thất bại |
| Bulk delete 1000 docs trong 10s | Low | Batch delete theo 100, parallel goroutines |

---

## 10. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| PDF 100 trang → done trong 2 phút | Async worker + batch embed 100 chunks/call |
| Duplicate content → "Duplicate content" error | SHA-256 contentHash, UNIQUE index |
| URL ảnh → OCR + text chunk | Image extractor qua Vision LLM (Bifrost) |
| Code `.go` → AST-aware chunking | go-tree-sitter + function/struct boundaries |
| Status real-time qua GET | `status` field được cập nhật mỗi bước pipeline |
| Bulk delete 1000 IDs trong 10s | Batch DELETE với `WHERE id = ANY($1)`, parallel |
