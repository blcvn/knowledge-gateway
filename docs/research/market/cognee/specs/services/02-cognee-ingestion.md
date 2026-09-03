# 02 — Cognee Ingestion Service

> **gRPC**: 9011 | **Health**: 9091

---

## 1. Purpose

Xử lý data ingestion pipeline: nhận raw data (file/URL/text), extract content, chunking cơ bản, lưu storage, và emit event cho Cognify.

---

## 2. Clean Architecture

```
services/cognee-ingestion/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Dataset, DataEntry, FileMetadata, PipelineRun
│   │   ├── value_object.go     # ContentType, FileHash, DataSource
│   │   ├── event.go            # DataIngestedEvent
│   │   └── errors.go           # ErrUnsupportedFormat, ErrExtractionFailed
│   ├── usecase/
│   │   ├── add_data.go         # Orchestrate file → extract → store → emit
│   │   ├── list_datasets.go    # Paginated dataset listing
│   │   ├── delete_dataset.go   # Cascade delete data + notify downstream
│   │   ├── get_pipeline_status.go
│   │   ├── port/
│   │   │   ├── input.go        # AddDataUseCase, ListDatasetsUseCase interfaces
│   │   │   └── output.go       # DatasetRepository, FileExtractor, ObjectStorage, EventPublisher
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go      # cognee.ingestion.v1.IngestionService impl
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/       # Dataset, DataEntry tables
│   │   ├── extractor/          # Content extractors
│   │   │   ├── pdf.go          # PDF text extraction (pdfcpu)
│   │   │   ├── html.go         # HTML → text (colly/goquery)
│   │   │   ├── docx.go         # DOCX extraction
│   │   │   ├── csv.go          # CSV/TSV parsing
│   │   │   ├── text.go         # Plain text passthrough
│   │   │   ├── web.go          # URL fetcher + readability
│   │   │   └── registry.go     # ContentType → Extractor mapping
│   │   ├── event/
│   │   │   └── publisher.go    # NATS publisher for cognee.ingestion.*
│   │   └── storage/
│   │       └── s3_adapter.go   # MinIO/S3 file storage
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Entities

```go
type Dataset struct {
    ID          uuid.UUID
    TenantID    string
    Name        string
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type DataEntry struct {
    ID          uuid.UUID
    DatasetID   uuid.UUID
    TenantID    string
    Source      DataSource    // FILE, URL, TEXT, API
    ContentType ContentType   // PDF, HTML, TEXT, CSV, DOCX
    RawContent  string        // Original text or S3 key
    StoragePath string        // S3 path for files
    FileHash    string        // SHA256 deduplication
    SizeBytes   int64
    Status      EntryStatus   // PENDING, PROCESSING, COMPLETED, FAILED
    CreatedAt   time.Time
}

type PipelineRun struct {
    ID          uuid.UUID
    DatasetID   uuid.UUID
    TenantID    string
    Stage       string        // INGESTION, COGNIFY, COMPLETED
    Status      RunStatus     // RUNNING, COMPLETED, FAILED
    StartedAt   time.Time
    CompletedAt *time.Time
    Error       *string
}
```

---

## 4. Use Case Flow: AddData

```
Client → Gateway → gRPC AddData(files, urls, text)
                        │
                        ▼
              ┌──── AddDataUseCase ────┐
              │ 1. Validate input      │
              │ 2. Create/Get Dataset  │
              │ 3. For each item:      │
              │    a. Extract content  │
              │    b. Store raw file   │
              │    c. Create DataEntry │
              │ 4. Create PipelineRun  │
              │ 5. Emit DataIngested   │
              └────────────────────────┘
                        │
                        ▼
              NATS: cognee.ingestion.data.ingested
                        │
                        ▼
              Cognify Service (subscriber)
```

---

## 5. NATS Events

| Subject | Payload | Subscriber |
|---------|---------|------------|
| `cognee.ingestion.data.ingested` | `{dataset_id, entry_ids[], tenant_id}` | cognee-cognify |
| `cognee.ingestion.pipeline.failed` | `{pipeline_run_id, error}` | vnp-admin (alerting) |

---

## 6. Extractor Registry

```go
type FileExtractor interface {
    Extract(ctx context.Context, reader io.Reader) (string, error)
    SupportedTypes() []ContentType
}

// Registry maps ContentType → Extractor
var extractors = map[ContentType]FileExtractor{
    PDF:  &PDFExtractor{},
    HTML: &HTMLExtractor{},
    DOCX: &DocxExtractor{},
    CSV:  &CSVExtractor{},
    TEXT: &TextExtractor{},
}
```
