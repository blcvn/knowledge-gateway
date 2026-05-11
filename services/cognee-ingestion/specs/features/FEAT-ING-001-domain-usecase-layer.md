---
id: FEAT-ING-001
title: Ingestion Service — Domain + Usecase Layer
service: cognee-ingestion
version: 1.0.0
status: InProgress
priority: P0
created: 2026-05-10
updated: 2026-05-10
implementation_started: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement Layer 1 (Domain) và Layer 2 (Usecase) cho cognee-ingestion service — multi-modal data ingestion pipeline entry point.

## Bối Cảnh Nghiệp Vụ

cognee-ingestion là điểm nhận data cho toàn bộ Cognee semantic KG engine. Hỗ trợ 3 loại input: File upload (PDF/DOCX/PPTX/CSV), direct text, và URL scraping. Data được tổ chức theo Dataset per tenant.

## Scope

### In Scope
- Domain entities: `Dataset`, `DataItem`, `DataSource`, `DatasetStatus`, `MimeType`
- Domain events: `DataIngestedEvent`
- Domain errors: `DatasetNotFoundError`, `DuplicateDatasetError`, `ExtractionFailedError`
- Usecase: `IngestFile`, `IngestText`, `IngestUrl`, `ManageDataset`
- Port interfaces (output): `DatasetRepository`, `DataItemRepository`, `FileStorage`, `EventPublisher`, `TextExtractor`
- Port interfaces (input): `FileIngester`, `TextIngester`, `UrlIngester`, `DatasetManager`
- DTOs: Request/Response structs

### Out of Scope
- gRPC handlers (FEAT-ING-002)
- Database implementations (FEAT-ING-002)
- Config, server bootstrap, Wire (FEAT-ING-003)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
internal/
├── domain/
│   ├── entity.go           # Dataset, DataItem
│   ├── value_object.go     # DatasetStatus, MimeType, DataSource
│   ├── event.go            # DataIngestedEvent
│   └── errors.go           # Domain error types
├── usecase/
│   ├── ingest_file.go      # FileIngester implementation
│   ├── ingest_text.go      # TextIngester implementation
│   ├── ingest_url.go       # UrlIngester implementation
│   ├── manage_dataset.go   # DatasetManager implementation
│   ├── port/
│   │   ├── input.go        # Input port interfaces
│   │   └── output.go       # Output port interfaces (repos, storage, events)
│   └── dto/
│       ├── request.go      # IngestFileReq, IngestTextReq, IngestUrlReq
│       └── response.go     # IngestResult, DatasetInfo
```

### Domain Entities

```go
// entity.go
type Dataset struct {
    ID             uuid.UUID
    TenantID       string
    Name           string
    Description    string
    Status         DatasetStatus
    FileCount      int
    TotalSizeBytes int64
    Metadata       map[string]string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type DataItem struct {
    ID          uuid.UUID
    DatasetID   uuid.UUID
    TenantID    string
    Source      DataSource
    Filename    string
    MimeType    MimeType
    RawText     string
    StoragePath string
    SizeBytes   int64
    FileHash    string        // SHA256 for dedup
    Metadata    map[string]string
    CreatedAt   time.Time
}
```

### Output Port Interfaces

```go
// port/output.go
type DatasetRepository interface {
    Create(ctx context.Context, ds *domain.Dataset) error
    GetByID(ctx context.Context, tenantID string, id uuid.UUID) (*domain.Dataset, error)
    List(ctx context.Context, tenantID string, cursor string, limit int) ([]*domain.Dataset, string, error)
    Delete(ctx context.Context, tenantID string, id uuid.UUID) error
    UpdateStatus(ctx context.Context, tenantID string, id uuid.UUID, status domain.DatasetStatus) error
}

type DataItemRepository interface {
    Create(ctx context.Context, item *domain.DataItem) error
    ListByDataset(ctx context.Context, datasetID uuid.UUID) ([]*domain.DataItem, error)
    DeleteByDataset(ctx context.Context, datasetID uuid.UUID) error
}

type FileStorage interface {
    Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64) (string, error)
    Delete(ctx context.Context, bucket, key string) error
}

type TextExtractor interface {
    Extract(ctx context.Context, reader io.Reader, mimeType domain.MimeType) (string, error)
    Supported() []domain.MimeType
}

type EventPublisher interface {
    PublishDataIngested(ctx context.Context, event domain.DataIngestedEvent) error
}
```

### Business Logic: IngestFile

```go
func (uc *IngestFileUseCase) Execute(ctx context.Context, req dto.IngestFileReq) (*dto.IngestResult, error) {
    // 1. Validate: dataset exists, mime type supported
    // 2. Compute file hash (SHA256) for dedup check
    // 3. Upload raw file to S3/MinIO
    // 4. Extract text content using TextExtractor
    // 5. Create DataItem record in DB
    // 6. Update dataset file count + size
    // 7. Publish DataIngestedEvent via NATS
    // 8. Return result with item ID + extracted preview
}
```

## Acceptance Criteria

- [ ] AC-1: Given valid domain entities, When instantiated, Then all fields are correctly typed and validated
- [ ] AC-2: Given a valid file input, When IngestFile usecase is called with mock ports, Then the flow executes: validate → hash → upload → extract → persist → publish
- [ ] AC-3: Given a duplicate file (same hash + dataset), When IngestFile is called, Then return existing item without re-processing
- [ ] AC-4: Given an unsupported mime type, When IngestFile is called, Then return `ErrUnsupportedFormat` domain error
- [ ] AC-5: Given valid text input, When IngestText is called, Then text is stored as DataItem and DataIngestedEvent published
- [ ] AC-6: Given a valid URL, When IngestUrl is called, Then URL is scraped, text extracted, and stored as DataItem
- [ ] AC-7: Port interfaces define clean contracts with no dependency on infrastructure types

## Test Requirements

- **Unit tests**: All usecase methods with mock ports (mockgen)
- **Domain tests**: Entity validation, value object behavior
- **Minimum coverage**: 80%

## Definition of Done

- [ ] Code implement đủ Acceptance Criteria
- [ ] Unit tests pass, coverage ≥ 80%
- [ ] Linter (golangci-lint) pass
- [ ] Domain layer has ZERO external imports (no gRPC, no DB, no framework)
- [ ] Usecase layer imports only domain — no adapter/infra imports
