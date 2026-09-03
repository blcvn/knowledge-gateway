# 02 — Document Service

> **gRPC**: 9001 | **Health**: 9081

---

## 1. Purpose

Quản lý vòng đời document: CRUD, content extraction, chunking, embedding generation. Xử lý 11 content types (text, PDF, image, video, webpage, Google Docs, Notion, OneDrive, etc.) thông qua async ingestion pipeline.

---

## 2. Clean Architecture

```
services/document-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Document, Chunk, ProcessingJob
│   │   ├── value_object.go     # DocumentType, DocumentStatus, ChunkType, ContentHash
│   │   ├── event.go            # DocumentCreated, DocumentProcessed, DocumentDeleted
│   │   └── errors.go           # ErrDuplicateContent, ErrUnsupportedType
│   ├── usecase/
│   │   ├── create_document.go  # Validate → dedup check → persist → enqueue
│   │   ├── get_document.go     # Fetch with chunks + status
│   │   ├── list_documents.go   # Paginated listing with filters
│   │   ├── delete_document.go  # Soft delete → publish event
│   │   ├── bulk_delete.go      # Batch delete by IDs or container tags
│   │   ├── process_document.go # Orchestrate: extract → chunk → embed → index
│   │   ├── port/
│   │   │   ├── input.go        # CreateDocumentUC, ProcessDocumentUC, etc.
│   │   │   └── output.go       # DocumentRepo, ChunkRepo, ContentExtractor,
│   │   │                       # EmbeddingGenerator, EventPublisher, ObjectStorage
│   │   └── dto/
│   │       └── document.go     # CreateDocumentInput, DocumentOutput
│   ├── adapter/
│   │   ├── grpc/handler.go     # DocumentServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── document.go # SQL queries (pgx)
│   │   │       └── chunk.go    # Chunk insert batch
│   │   ├── extractor/          # Content extraction adapters
│   │   │   ├── text.go         # Plaintext passthrough
│   │   │   ├── pdf.go          # PDF text extraction (pdfcpu/unidoc)
│   │   │   ├── html.go         # HTML → clean text (goquery)
│   │   │   ├── image.go        # OCR via AI provider
│   │   │   ├── video.go        # Transcription via AI provider
│   │   │   └── registry.go     # DocumentType → Extractor mapping
│   │   ├── chunker/
│   │   │   ├── semantic.go     # Semantic boundary chunking
│   │   │   ├── ast.go          # Code AST-aware chunking
│   │   │   ├── fixed.go        # Fixed-size fallback
│   │   │   └── registry.go     # ContentType → Chunker mapping
│   │   ├── embedding/
│   │   │   └── provider.go     # Uses pkg/adapters/embedding
│   │   ├── storage/
│   │   │   └── s3.go           # File upload/download (S3/Minio)
│   │   └── event/
│   │       └── publisher.go    # NATS: document.created, document.processed
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   ├── 001_create_documents.up.sql
│   └── 002_create_chunks.up.sql
└── Dockerfile
```

---

## 3. Domain Model

```go
type Document struct {
    ID              string
    CustomID        *string          // User-provided dedup ID
    ContentHash     *string          // SHA-256 for dedup
    OrgID           string           // Tenant scope
    UserID          string           // Creator
    ConnectionID    *string          // Source connection FK
    Title           *string
    Content         *string
    Summary         *string
    URL             *string
    Source          *string
    Type            DocumentType     // text|pdf|image|video|webpage|...
    Status          DocumentStatus   // queued→extracting→chunking→embedding→indexing→done|failed
    Metadata        map[string]any
    ProcessingMeta  *ProcessingMetadata
    TokenCount      *int
    WordCount       *int
    ChunkCount      *int
    SummaryEmbedding    []float32
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type Chunk struct {
    ID              string
    DocumentID      string
    Content         string
    EmbeddedContent *string          // Optimized embedding text
    Type            ChunkType        // text | image
    Position        int              // Order within document
    Embedding       []float32        // Primary vector
    MatryoshkaEmbed []float32        // Truncatable vector
    CreatedAt       time.Time
}

type DocumentStatus string
const (
    StatusQueued      DocumentStatus = "queued"
    StatusExtracting  DocumentStatus = "extracting"
    StatusChunking    DocumentStatus = "chunking"
    StatusEmbedding   DocumentStatus = "embedding"
    StatusIndexing    DocumentStatus = "indexing"
    StatusDone        DocumentStatus = "done"
    StatusFailed      DocumentStatus = "failed"
)
```

---

## 4. Processing Pipeline (Async Worker)

```
NATS: document.ingest.requested
          │
          ▼
┌── Worker Pool (N goroutines) ─────────────────────────────┐
│                                                            │
│  1. EXTRACTING                                             │
│     ├── Detect DocumentType                                │
│     ├── Fetch content (URL/S3/inline)                      │
│     ├── Extract text via type-specific extractor            │
│     └── Generate summary (LLM)                             │
│                                                            │
│  2. CHUNKING                                               │
│     ├── Select chunking strategy (semantic/AST/fixed)      │
│     ├── Split into chunks with position tracking           │
│     ├── Compute contentHash (SHA-256) for dedup            │
│     └── Optimize embeddedContent per chunk                 │
│                                                            │
│  3. EMBEDDING                                              │
│     ├── Batch embedding generation (max 100 per batch)     │
│     ├── Primary embeddings (all chunks)                    │
│     ├── Matryoshka embeddings (configurable)               │
│     └── Summary embedding (document-level)                 │
│                                                            │
│  4. INDEXING                                               │
│     ├── Batch insert chunks with embeddings                │
│     ├── Publish document.processed event                   │
│     │   → Memory Service listens to extract facts          │
│     └── Update document status → DONE                      │
│                                                            │
│  Error: Update status → FAILED, store error in metadata    │
│  Retry: 3 attempts with exponential backoff                │
└────────────────────────────────────────────────────────────┘
```

---

## 5. gRPC Interface

```protobuf
service DocumentService {
  rpc CreateDocument(CreateDocumentRequest) returns (CreateDocumentResponse);
  rpc GetDocument(GetDocumentRequest) returns (DocumentResponse);
  rpc ListDocuments(ListDocumentsRequest) returns (ListDocumentsResponse);
  rpc ListDocumentsWithMemories(ListDocumentsWithMemoriesRequest) returns (ListDocumentsWithMemoriesResponse);
  rpc DeleteDocument(DeleteDocumentRequest) returns (google.protobuf.Empty);
  rpc BulkDeleteDocuments(BulkDeleteRequest) returns (BulkDeleteResponse);
  rpc GetProcessingStatus(GetProcessingStatusRequest) returns (ProcessingStatusResponse);
}
```
