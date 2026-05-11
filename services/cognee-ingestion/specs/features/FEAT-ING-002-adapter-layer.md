---
id: FEAT-ING-002
title: Ingestion Service — Adapter Layer (gRPC + NATS + Repositories)
service: cognee-ingestion
version: 1.0.0
status: InProgress
priority: P0
created: 2026-05-10
updated: 2026-05-10
implementation_started: 2026-05-10
linked_sol: SOL-001
linked_feat: FEAT-ING-001
---

## Mục Tiêu

Implement Layer 3 (Adapter) cho cognee-ingestion — gRPC handlers, repository implementations (PostgreSQL, MinIO), NATS event publisher, text extractors.

## Scope

### In Scope
- gRPC handler: `CogneeIngestionServiceServer` implementation
- Proto ↔ Domain mapper
- PostgreSQL repositories: `DatasetRepository`, `DataItemRepository`
- MinIO/S3 file storage adapter
- NATS event publisher: `cognee.data.ingested`
- Text extractors: PDF, DOCX, PPTX, CSV, HTML, plain text
- URL scraper adapter

### Out of Scope
- Domain + Usecase (FEAT-ING-001 — already done)
- Config, Wire, server bootstrap (FEAT-ING-003)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
internal/adapter/
├── grpc/
│   ├── handler.go           # CogneeIngestionServiceServer impl
│   └── mapper.go            # Proto ↔ Domain mapping
├── repository/
│   └── postgres/
│       ├── dataset_repo.go  # DatasetRepository impl
│       └── dataitem_repo.go # DataItemRepository impl
├── storage/
│   └── minio_adapter.go     # FileStorage impl (S3/MinIO)
├── extractor/
│   ├── registry.go          # MimeType → Extractor routing
│   ├── pdf.go               # PDF text extraction
│   ├── docx.go              # DOCX extraction
│   ├── pptx.go              # PPTX extraction
│   ├── csv.go               # CSV/TSV parsing
│   ├── html.go              # HTML → text
│   └── text.go              # Plain text passthrough
├── scraper/
│   └── url_scraper.go       # URL scraping (colly/rod)
├── event/
│   └── nats_publisher.go    # NATS JetStream publisher
└── client/                  # (empty for now — no outbound gRPC)
```

### gRPC Service Definition

```protobuf
service CogneeIngestionService {
  rpc CreateDataset(CreateDatasetRequest) returns (Dataset);
  rpc DeleteDataset(DeleteDatasetRequest) returns (google.protobuf.Empty);
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
  rpc GetDatasetStatus(GetDatasetStatusRequest) returns (DatasetStatusResponse);
  rpc AddData(stream AddDataRequest) returns (AddDataResponse);  // streaming upload
  rpc AddText(AddTextRequest) returns (AddTextResponse);
  rpc AddUrl(AddUrlRequest) returns (AddUrlResponse);
}
```

### PostgreSQL Schema

```sql
-- datasets table
CREATE TABLE datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'PENDING',
    file_count INT DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);
ALTER TABLE datasets ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_rls ON datasets USING (tenant_id = current_setting('app.tenant_id'));

-- data_items table
CREATE TABLE data_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    tenant_id VARCHAR(255) NOT NULL,
    source VARCHAR(20) NOT NULL,     -- FILE, TEXT, URL
    filename VARCHAR(500),
    mime_type VARCHAR(100),
    raw_text TEXT,
    storage_path TEXT,
    size_bytes BIGINT DEFAULT 0,
    file_hash VARCHAR(64),           -- SHA256
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_data_items_dataset ON data_items(dataset_id);
CREATE INDEX idx_data_items_hash ON data_items(file_hash, dataset_id);
```

## Acceptance Criteria

- [ ] AC-1: Given a gRPC CreateDataset request, When handler processes it, Then dataset is persisted in PostgreSQL and returned
- [ ] AC-2: Given a streaming AddData request with PDF file, When handler processes it, Then file is uploaded to MinIO, text extracted, DataItem created, and DataIngestedEvent published to NATS
- [ ] AC-3: Given Proto messages, When mapper converts to domain and back, Then no data is lost
- [ ] AC-4: Given tenant_id in gRPC metadata, When repository queries, Then only that tenant's data is returned (RLS)
- [ ] AC-5: Given supported file types (PDF/DOCX/CSV/HTML/TXT), When extractor processes, Then text is correctly extracted
- [ ] AC-6: Given a valid URL, When scraper processes, Then page content is extracted as clean text

## Test Requirements

- **Unit tests**: Handler with mock usecase, Repo with test DB (testcontainers), Extractors with fixture files
- **Integration tests**: gRPC → handler → usecase → mock repos
- **Minimum coverage**: 80%
