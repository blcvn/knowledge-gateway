---
id: TDD-cognee-ingestion
title: Technical Design — cognee-ingestion
service: cognee-ingestion
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: Cognee
---

# Technical Design — cognee-ingestion

> **Group**: Cognee (Semantic KG) | **gRPC Port**: 9011 | **Origin**: Cognee L2-L4

## 1. Service Overview

Multi-modal data ingestion pipeline: file upload, text extraction, URL scraping, dataset management. Entry point for all data flowing into the Cognee semantic knowledge graph pipeline. Organizes data by dataset/namespace per tenant.

**Core Use Cases**:
- `IngestFile` — Upload + extract text from PDF/DOCX/PPTX/CSV via streaming gRPC
- `IngestText` — Direct text input with auto-classification
- `IngestUrl` — Web scraping + content extraction
- `ManageDataset` — CRUD dataset lifecycle per tenant

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── entity.go           # Dataset, DataItem, DataSource
├── value_object.go     # DatasetStatus (PENDING/READY/COGNIFYING/ERROR), MimeType
├── event.go            # DataIngested domain event
└── errors.go           # DatasetNotFoundError, DuplicateDatasetError
```

**Key Entities**:
```go
type Dataset struct {
    ID             uuid.UUID
    TenantID       string
    Name           string
    Status         DatasetStatus  // PENDING, READY, COGNIFYING, ERROR
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
    Source      DataSource  // FILE, TEXT, URL
    Filename    string
    MimeType    string
    RawText     string
    StoragePath string
    SizeBytes   int64
    Metadata    map[string]string
    CreatedAt   time.Time
}
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── ingest_file.go       # File upload + text extraction pipeline
├── ingest_text.go       # Direct text ingestion
├── ingest_url.go        # URL scraping + extraction
├── manage_dataset.go    # Dataset CRUD operations
├── port/
│   ├── input.go         # FileIngester, TextIngester, UrlIngester, DatasetManager
│   └── output.go        # DatasetRepo, DataItemRepo, FileStorage, EventPublisher
└── dto/
    ├── request.go
    └── response.go
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/                # gRPC server handlers (inbound)
│   ├── handler.go       # CogneeIngestionServiceServer implementation
│   └── mapper.go        # Proto ↔ Domain mapping
├── event/               # NATS event publisher (outbound)
│   └── publisher.go     # cognee.data.ingested event publisher
└── client/              # External service gRPC clients (outbound)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/
│   ├── postgres/        # PostgreSQL: Dataset + DataItem repositories
│   └── minio/           # MinIO/S3: Raw file storage adapter
├── scraper/             # URL scraping implementation (colly/rod)
├── extractor/           # Text extraction: PDF, DOCX, PPTX, CSV
├── config/config.go     # Viper configuration loader
├── server/grpc.go       # gRPC server bootstrap
├── telemetry/           # OTel trace/metrics setup
└── wire/wire.go         # Google Wire DI providers + injector
```

## 3. gRPC API

```protobuf
service CogneeIngestionService {
  rpc CreateDataset(CreateDatasetRequest) returns (Dataset);
  rpc DeleteDataset(DeleteDatasetRequest) returns (google.protobuf.Empty);
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc GetDatasetStatus(GetDatasetStatusRequest) returns (DatasetStatus);
  rpc AddData(stream AddDataRequest) returns (AddDataResponse);
  rpc AddText(AddTextRequest) returns (AddTextResponse);
  rpc AddUrl(AddUrlRequest) returns (AddUrlResponse);
}
```

## 4. NATS Events

| Direction | Subject | Payload | Peer |
|-----------|---------|---------|------|
| **Publish** | `cognee.data.ingested` | `{dataset_id, tenant_id, item_ids[]}` | cognee-cognify |
| **Subscribe** | `admin.tenant.deleted` | `{tenant_id}` | vnp-admin |

## 5. Data Model

- **PostgreSQL**: `datasets` (metadata), `data_items` (item records)
- **MinIO/S3**: Raw file binaries at `{tenant_id}/{dataset_id}/{item_id}`
- **Redis**: Upload progress cache with TTL

## 6. Cross-Service Dependencies

| Target Service | Protocol | Purpose |
|---------------|----------|---------|
| cognee-cognify | NATS (async) | Trigger KG pipeline after ingestion |
| vnp-admin | NATS (subscribe) | Handle tenant deletion cascade |

## 7. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs + upload metrics
- **Traces**: OTel spans for every usecase method
- **Logs**: Structured JSON via slog with request_id, tenant_id, dataset_id
- **Health**: gRPC health check + HTTP /healthz on port 9091

## 8. Multi-Tenancy

Tenant isolation via gRPC metadata `x-tenant-id` → propagated to all DB queries via PostgreSQL RLS + MinIO namespace.

---

> **Next Steps**: Decompose this TDD into individual FEAT/ARCH specs in `specs/features/` and `specs/architecture/` before implementation.
