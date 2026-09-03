# TASK-SM-004 — services/document-service: Domain Model & Extractor Pipeline

**Task ID:** TASK-SM-004  
**Wave:** 2 (Core Memory)  
**Solution:** [SOL-SM-001](../solutions/SOL-SM-001-Document-Ingestion-Pipeline.md)  
**Depends on:** TASK-SM-001 (OrgID from Auth context)  
**Ước tính:** 4h  
**Priority:** Critical

**Trạng thái:** 🔄 Partial  
**Ghi chú:** sm-document: 3 .go - domain scaffold only, extractors missing  
---

## Mục tiêu

Tạo `services/document-service/` — phần 1: Domain Model, Extractors, và PostgreSQL Schema:
1. `Document` entity với 11 DocumentTypes + 7-stage pipeline status
2. `Chunk` entity với Matryoshka embeddings (1536 + 256 dims)
3. `ExtractorRegistry` pattern
4. Text + PDF + Webpage + Image/Video extractors
5. PostgreSQL schema (documents + chunks với HNSW indexes)

---

## Công việc cụ thể

### 1. Tạo Domain Model

**`services/document-service/internal/domain/document.go`**

```go
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
    StatusQueued     DocumentStatus = "queued"
    StatusExtracting DocumentStatus = "extracting"
    StatusChunking   DocumentStatus = "chunking"
    StatusEmbedding  DocumentStatus = "embedding"
    StatusIndexing   DocumentStatus = "indexing"
    StatusDone       DocumentStatus = "done"
    StatusFailed     DocumentStatus = "failed"
)

type Document struct {
    ID             string
    CustomID       *string        // User-defined dedup key
    ContentHash    *string        // SHA-256 hex (unique per OrgID)
    OrgID          string
    UserID         string
    ConnectionID   *string
    Title          *string
    Content        *string
    Summary        *string
    URL            *string
    Type           DocumentType
    Status         DocumentStatus
    Metadata       map[string]any
    ProcessingMeta *ProcessingMetadata
    TokenCount     *int
    ChunkCount     *int
    SummaryEmbedding []float32    // 1536-dim for doc-level search
    ContainerTags  []string       // Space/project tags
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type ProcessingMetadata struct {
    StartedAt     *time.Time
    CompletedAt   *time.Time
    ErrorMsg      *string
    RetryCount    int
    ExtractorUsed string
    ChunkerUsed   string
}

type Chunk struct {
    ID               string
    DocumentID       string
    OrgID            string
    Content          string
    EmbeddedContent  *string   // Content used for embedding (may differ from raw)
    Type             ChunkType // "text" | "image"
    Position         int
    Embedding        []float32 // 1536-dim dense vector
    MatryoshkaEmbed  []float32 // 256-dim truncatable
    CreatedAt        time.Time
}
```

### 2. Tạo Extractor Interface & Registry

**`services/document-service/internal/domain/extractor.go`**

```go
type Extractor interface {
    Extract(ctx context.Context, doc *Document) (content string, err error)
    Supports(docType DocumentType) bool
}

type ExtractorRegistry struct{ extractors []Extractor }
func (r *ExtractorRegistry) Get(docType DocumentType) (Extractor, error)
func (r *ExtractorRegistry) Register(e Extractor)
```

### 3. Implement Extractors

**`services/document-service/internal/infra/extractor/text.go`**:
```go
// TextExtractor: passthrough — content là raw text
// Supports: DocTypeText, DocTypeTweet
```

**`services/document-service/internal/infra/extractor/pdf.go`**:
```go
// PDFExtractor: dùng github.com/pdfcpu/pdfcpu
// Extract → text from all pages
// Fallback: nếu pdfcpu fail (scanned PDF) → chuyển sang image extractor
// Dep: go get github.com/pdfcpu/pdfcpu
```

**`services/document-service/internal/infra/extractor/webpage.go`**:
```go
// WebpageExtractor: HTTP GET → goquery HTML clean
// Remove: script, style, nav, footer, ads
// Extract: main content via article/main selector, fallback to body
// Dep: go get github.com/PuerkitoBio/goquery
```

**`services/document-service/internal/infra/extractor/image.go`**:
```go
// ImageExtractor: Vision LLM qua Bifrost
// Prompt: "Extract all text and describe visual content from this image"
// Supports: DocTypeImage
// Timeout: 30s (OCR is slow)
```

**`services/document-service/internal/infra/extractor/video.go`**:
```go
// VideoExtractor: Transcription via Bifrost
// Supports: DocTypeVideo
```

### 4. Tạo PostgreSQL Schema

**`services/document-service/migrations/001_create_documents.sql`**:

```sql
CREATE TABLE documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custom_id       TEXT,
    content_hash    TEXT,
    org_id          UUID NOT NULL,
    user_id         UUID NOT NULL,
    connection_id   UUID,
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

-- Indexes
CREATE INDEX idx_documents_org_status ON documents(org_id, status);
CREATE INDEX idx_documents_container_tags ON documents USING GIN(container_tags);
CREATE INDEX ON documents USING hnsw (summary_embedding vector_cosine_ops) WITH (m=16, ef_construction=128);
CREATE INDEX ON chunks USING hnsw (embedding vector_cosine_ops) WITH (m=16, ef_construction=128);
CREATE INDEX ON chunks USING hnsw (matryoshka_embedding vector_cosine_ops) WITH (m=8, ef_construction=64);
CREATE INDEX idx_chunks_doc ON chunks(document_id);
```

### 5. Tạo DocumentRepository + ChunkRepository

**`services/document-service/internal/infra/postgres/`**:
- `FindByContentHash(ctx, orgID, hash)` → dedup check
- `FindByCustomID(ctx, orgID, customID)` → dedup check
- `UpdateStatus(ctx, docID, status, processingMeta)` → pipeline progress
- `ChunkRepo.BulkUpsert(ctx, chunks)` → batch 100 using `unnest()`

### 6. Tests

- `TestExtractorRegistry_GetPDF`: registry.Get("pdf") → PDFExtractor
- `TestExtractorRegistry_GetUnknown`: unknown type → ErrExtractorNotFound
- `TestTextExtractor_Passthrough`: content = raw input unchanged
- `TestWebpageExtractor_RemovesNavFooter`: HTML with nav/footer → clean content
- `TestContentHash_Dedup`: same content, same orgID → same hash

---

## Acceptance Criteria

- [ ] `go build ./services/document-service/...` không lỗi
- [ ] Migration SQL chạy, HNSW indexes tạo thành công
- [ ] ExtractorRegistry.Get("pdf") → non-nil PDFExtractor
- [ ] ExtractorRegistry.Get("unknown") → ErrExtractorNotFound
- [ ] TextExtractor → content unchanged
- [ ] WebpageExtractor → no `<script>`, `<style>` tags in output
- [ ] UNIQUE (org_id, content_hash) constraint enforced
- [ ] `go test ./services/document-service/...` pass

---

## Files tạo ra

```
services/document-service/
├── internal/
│   ├── domain/
│   │   ├── document.go
│   │   └── extractor.go
│   └── infra/
│       ├── postgres/
│       │   ├── document_repo.go
│       │   └── chunk_repo.go
│       └── extractor/
│           ├── text.go
│           ├── pdf.go
│           ├── webpage.go
│           ├── image.go
│           ├── video.go
│           └── extractor_test.go
└── migrations/
    └── 001_create_documents.sql
```

## Sau khi hoàn thành

Chạy: `go build ./services/document-service/... && go test ./services/document-service/...`
