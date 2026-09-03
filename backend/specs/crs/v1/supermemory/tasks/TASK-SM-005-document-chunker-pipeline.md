# TASK-SM-005 — services/document-service: Chunkers, Pipeline & NATS Worker

**Task ID:** TASK-SM-005  
**Wave:** 2 (Core Memory)  
**Solution:** [SOL-SM-001](../solutions/SOL-SM-001-Document-Ingestion-Pipeline.md)  
**Depends on:** TASK-SM-004 (domain + extractors)  
**Ước tính:** 4h  
**Priority:** Critical

**Trạng thái:** ⏳ Pending  
**Ghi chú:** Document chunker pipeline not implemented (sm-document minimal)  
---

## Mục tiêu

Hoàn thiện `services/document-service/` — phần 2: Chunkers, Ingestion Pipeline, và NATS Async Worker:
1. AST chunker (`go-tree-sitter`) + Semantic chunker + Fixed fallback
2. `ChunkerRegistry` pattern
3. `IngestionPipeline` (6 steps: extract→chunk→embed→index→done)
4. NATS JetStream consumer (worker pool, 5 workers default)
5. Use Cases: CreateDocument (dedup + publish) + CRUD

---

## Công việc cụ thể

### 1. Implement Chunkers

**`services/document-service/internal/infra/chunker/fixed.go`**

```go
// FixedChunker: fallback — split by token count
// MaxTokens: 512 per chunk, overlap: 50 tokens
// Supports: all types
func (c *FixedChunker) Chunk(ctx context.Context, doc *Document) ([]Chunk, error)
func (c *FixedChunker) Supports(docType DocumentType) bool { return true } // catch-all
```

**`services/document-service/internal/infra/chunker/semantic.go`**

```go
// SemanticChunker: split by sentence boundaries + semantic similarity
// Merge sentences until cosine similarity drops below threshold (0.8)
// Requires: embedder calls per sentence (expensive → cache)
// Supports: text, pdf, webpage, tweet, notion_doc, onedrive
func (c *SemanticChunker) Chunk(ctx context.Context, doc *Document) ([]Chunk, error)
```

**`services/document-service/internal/infra/chunker/ast.go`**

```go
// ASTChunker: go-tree-sitter based code chunking
// Split at function, struct, interface, const block boundaries
// Dep: go get github.com/smacker/go-tree-sitter
// Languages: Go (initial), Python, TypeScript (via tree-sitter grammars)
// Fallback: if AST parse fails → FixedChunker
// Supports: docType with code content (detected via metadata["language"])
func (c *ASTChunker) Chunk(ctx context.Context, doc *Document) ([]Chunk, error)
func (c *ASTChunker) Supports(docType DocumentType) bool // check metadata
```

**`services/document-service/internal/domain/chunker.go`**

```go
type Chunker interface {
    Chunk(ctx context.Context, doc *Document) ([]Chunk, error)
    Supports(docType DocumentType) bool
}

// ChunkerRegistry: try in order AST → Semantic → Fixed
type ChunkerRegistry struct{ chunkers []Chunker }
func (r *ChunkerRegistry) Get(doc *Document) Chunker // returns first Supports=true
```

### 2. Implement Ingestion Pipeline

**`services/document-service/internal/adapter/worker/pipeline.go`**

```go
type IngestionPipeline struct {
    extractors  *ExtractorRegistry
    chunkers    *ChunkerRegistry
    embedder    EmbedderPort         // Bifrost embedder
    docRepo     DocumentRepository
    chunkRepo   ChunkRepository
    publisher   EventPublisher
}

// Process: 6-step pipeline với retry 3x (1s, 2s, 4s backoff)
func (p *IngestionPipeline) Process(ctx context.Context, docID string) error {
    // Step 1: StatusExtracting → extractor.Extract
    // Step 2: StatusChunking  → chunker.Chunk
    // Step 3: StatusEmbedding → embedder.BatchEmbed (100 chunks/call)
    //         → also embed summary for doc-level search
    // Step 4: StatusIndexing  → chunkRepo.BulkUpsert
    // Step 5: StatusDone      → update doc.ChunkCount, doc.TokenCount
    // Step 6: Publish NATS "document.processed"
}

// Retry: exponential backoff 1s → 2s → 4s, max 3 attempts
// On maxRetry exceeded: StatusFailed, publish "document.failed"
func (p *IngestionPipeline) handleRetry(ctx, doc, err) error
```

### 3. Implement NATS Worker Pool

**`services/document-service/internal/adapter/worker/pool.go`**

```go
// WorkerPool: configurable N workers (default 5)
// Each worker: subscribe to "document.ingest.requested", process docID
// On error: NATS JetStream NAK → auto re-delivery (MaxDeliver=3)
type WorkerPool struct {
    pipeline *IngestionPipeline
    workers  int
    nats     NATSClient
}

func (p *WorkerPool) Start(ctx context.Context) error {
    for i := 0; i < p.workers; i++ {
        go p.runWorker(ctx)
    }
    return nil
}
```

### 4. Implement CreateDocument Use Case

**`services/document-service/internal/usecase/create_document.go`**

```go
// CreateDocument: dedup → create → publish NATS (NOT sync processing)
func (uc *CreateDocumentUseCase) Execute(ctx context.Context, req CreateDocumentRequest) (*Document, error) {
    // 1. Tính ContentHash nếu có content (SHA-256)
    // 2. FindByContentHash → ErrDuplicateContent (409 Conflict)
    // 3. FindByCustomID → ErrDuplicateCustomID (409 Conflict)
    // 4. Create doc với Status=queued
    // 5. Publish NATS "document.ingest.requested" {DocID}
    // 6. Return doc (processing is async)
}
```

### 5. REST API + Gateway Bootstrap

**`services/document-service/internal/adapter/grpc/document_server.go`**:
- CreateDocument, ListDocuments, GetDocument, DeleteDocument, BulkDeleteDocuments

**`gateway/adapter/handler/document_handler.go`**:
```
POST   /api/v1/documents           → CreateDocument (requires document:create)
POST   /api/v1/documents/list      → ListDocuments
GET    /api/v1/documents/{id}      → GetDocument (with status + chunk count)
DELETE /api/v1/documents/{id}      → DeleteDocument (requires document:delete)
DELETE /api/v1/documents/bulk      → BulkDeleteDocuments (batch 100)
```

**`apps/memory/internal/bootstrap/document.go`**: register service với InProcessRegistry

### 6. Tests

- `TestChunkerRegistry_ASTFirst`: Go code doc → ASTChunker selected
- `TestChunkerRegistry_FixedFallback`: unknown type → FixedChunker
- `TestFixedChunker_MaxTokens512`: 1024-token content → 2 chunks
- `TestCreateDocument_DuplicateHash`: same content + orgID → 409
- `TestCreateDocument_PublishesNATS`: after create → NATS event published
- `TestIngestionPipeline_Steps`: mock extractor + chunker → all 6 steps executed
- `TestIngestionPipeline_RetryOnExtractorError`: error → retry 3x → StatusFailed
- `TestBulkDelete_Batch100`: 1000 IDs → 10 batches of 100

---

## Acceptance Criteria

- [ ] `go build ./services/document-service/...` không lỗi
- [ ] Go code document → ASTChunker selected (not Fixed)
- [ ] PDF document → SemanticChunker (not AST)
- [ ] FixedChunker: 1024 tokens → ceil(1024/512) = 2 chunks
- [ ] Duplicate contentHash → 409 Conflict response
- [ ] After CreateDocument → NATS "document.ingest.requested" published
- [ ] Pipeline ProcessingMeta.RetryCount incremented on each retry
- [ ] StatusFailed after 3 failed retries
- [ ] BulkDelete 1000 IDs → processed in batches of 100
- [ ] `go test ./services/document-service/...` pass

---

## Files tạo ra

```
services/document-service/
├── internal/
│   ├── domain/
│   │   └── chunker.go
│   ├── usecase/
│   │   ├── create_document.go
│   │   ├── create_document_test.go
│   │   ├── list_documents.go
│   │   ├── get_document.go
│   │   ├── delete_document.go
│   │   └── bulk_delete.go
│   ├── adapter/
│   │   ├── worker/
│   │   │   ├── pipeline.go
│   │   │   ├── pipeline_test.go
│   │   │   └── pool.go
│   │   └── grpc/
│   │       └── document_server.go
│   └── infra/
│       └── chunker/
│           ├── fixed.go
│           ├── semantic.go
│           ├── ast.go
│           └── chunker_test.go
├── migrations/
│   └── (already done in TASK-004)

apps/memory/internal/bootstrap/
└── document.go   (NEW)

gateway/adapter/handler/
└── document_handler.go (NEW)
```

## Sau khi hoàn thành

Chạy: `go build ./... && go test ./services/document-service/...`
