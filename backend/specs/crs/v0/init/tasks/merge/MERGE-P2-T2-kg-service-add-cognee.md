---
id: MERGE-P2-T2
title: "kg-service: Extend — Thêm Cognee adapter (4 services)"
phase: P2
service: kg-service (extend)
priority: P1
status: Done
estimated: 8h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P2-T1]
---

## Mục Tiêu

Mở rộng `kg-service` để xử lý thêm tất cả Cognee routes. **Quan trọng:** Cognee là Python service — `kg-service` không embed Python runtime mà implement **HTTP client adapter** gọi vào Cognee Python instance.

## Services Bị Absorb

| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `cognee-ingestion` | 4,588 | Dataset + data management API |
| `cognee-cognify` | 2,488 | Cognification pipeline trigger |
| `cognee-pipeline` | 655 | Batch processing |
| `cognee-search` | 1,192 | Cognee semantic search |

**Tổng: 8,923 lines** → added to kg-service as HTTP adapter layer

## Architecture

```
kg-service mở rộng:
├── internal/
│   ├── domain/
│   │   └── cognee/
│   │       ├── entity.go        # Dataset, DataUpload, CognifyJob, CogneeResult
│   │       └── errors.go
│   ├── usecase/
│   │   └── cognee/
│   │       ├── dataset.go       # CreateDataset, UploadData
│   │       ├── cognify.go       # Cognify (trigger pipeline)
│   │       └── search.go        # Search
│   ├── adapter/
│   │   ├── cognee/              # HTTP client adapter → Cognee Python
│   │   │   └── client.go        # HTTP calls to COGNEE_URL
│   │   └── grpc/
│   │       └── router.go        # Add cognee routes to existing router
│   └── infra/
│       └── cognee/              # Cognee HTTP client impl
```

## Domain Entities

```go
// domain/cognee/entity.go

type Dataset struct {
    ID          string
    Name        string
    TenantID    string
    Status      string    // "empty" | "uploading" | "ready" | "cognifying" | "indexed"
    DataCount   int
    CreatedAt   time.Time
}

type DataItem struct {
    DatasetID  string
    ContentType string  // "text" | "pdf" | "url" | "json"
    Content    []byte
    URI        string   // optional: URL or file path
    Metadata   map[string]any
}

type CognifyJob struct {
    JobID     string
    DatasetID string
    Status    string    // "pending" | "running" | "completed" | "failed"
    Progress  float64
    StartedAt time.Time
    DoneAt    *time.Time
}

type CogneeSearchResult struct {
    Content  string
    Score    float64
    Source   string
    Metadata map[string]any
}
```

## Usecase Implementation

```go
// usecase/cognee/dataset.go
type DatasetUseCase struct {
    client port.CogneeClient  // Interface to Python Cognee API
    store  port.DatasetRepository
}

func (uc *DatasetUseCase) CreateDataset(ctx context.Context, tenantID, name string) (*Dataset, error) {
    // Call Cognee Python API: POST /datasets
    resp, err := uc.client.CreateDataset(ctx, name)
    if err != nil {
        return nil, fmt.Errorf("cognee: create dataset: %w", err)
    }
    // Persist metadata locally for tenant tracking
    ds := &Dataset{ID: resp.ID, Name: name, TenantID: tenantID}
    return ds, uc.store.Save(ctx, ds)
}

func (uc *DatasetUseCase) UploadData(ctx context.Context, datasetID string, item *DataItem) error {
    // Call Cognee Python API: POST /datasets/{id}/data
    return uc.client.UploadData(ctx, datasetID, item)
}

// usecase/cognee/cognify.go
func (uc *CognifyUseCase) Cognify(ctx context.Context, datasetID string) (*CognifyJob, error) {
    // Trigger cognification: POST /datasets/{id}/cognify
    job, err := uc.client.Cognify(ctx, datasetID)
    // Poll status or return job ID for async tracking
    return job, err
}

// usecase/cognee/search.go
func (uc *SearchUseCase) Search(ctx context.Context, query string, tenantID string) ([]*CogneeSearchResult, error) {
    // Call Cognee Python API: POST /search
    return uc.client.Search(ctx, query)
}
```

## HTTP Client Adapter (Cognee Python)

```go
// adapter/cognee/client.go
type CogneeHTTPClient struct {
    baseURL    string
    httpClient *http.Client
    apiKey     string
}

func NewCogneeHTTPClient(baseURL, apiKey string) *CogneeHTTPClient {
    return &CogneeHTTPClient{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 60 * time.Second},
        apiKey:     apiKey,
    }
}

func (c *CogneeHTTPClient) CreateDataset(ctx context.Context, name string) (*DatasetResponse, error) {
    body, _ := json.Marshal(map[string]string{"name": name})
    req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/datasets", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("cognee http: %w", err)
    }
    defer resp.Body.Close()
    var result DatasetResponse
    return &result, json.NewDecoder(resp.Body).Decode(&result)
}

func (c *CogneeHTTPClient) UploadData(ctx context.Context, datasetID string, item *DataItem) error { ... }
func (c *CogneeHTTPClient) Cognify(ctx context.Context, datasetID string) (*CognifyJob, error) { ... }
func (c *CogneeHTTPClient) Search(ctx context.Context, query string) ([]*CogneeSearchResult, error) { ... }
```

## ForwardService Routes (Added to existing router)

```go
// Thêm vào adapter/grpc/router.go
func RegisterCogneeRoutes(router *forward.Router, ds DatasetHandler, cog CognifyHandler, search CogneeSearchHandler) {
    router.Handle("POST", "/v1/cognee/datasets",            ds.CreateDataset)
    router.Handle("POST", "/v1/cognee/datasets/*/data",     ds.UploadData)
    router.Handle("POST", "/v1/cognee/datasets/*/cognify",  cog.Cognify)
    router.Handle("POST", "/v1/cognee/search",              search.Search)
}
```

## Database Migration (Cognee metadata tracking)

```sql
-- migrations/002_cognee_datasets.sql
CREATE TABLE IF NOT EXISTS cognee_datasets (
    id         TEXT PRIMARY KEY,           -- Cognee's dataset ID
    tenant_id  TEXT NOT NULL,
    name       TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'empty',
    data_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cognee_datasets_tenant ON cognee_datasets(tenant_id);

CREATE TABLE IF NOT EXISTS cognee_jobs (
    job_id     TEXT PRIMARY KEY,
    dataset_id TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending',
    progress   FLOAT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    done_at    TIMESTAMPTZ
);
```

## Config Environment Variables (thêm vào kg-service)

```bash
# Cognee Python service connection
COGNEE_URL=http://cognee:8000         # Required for Cognee features
COGNEE_API_KEY=...                     # Optional: Cognee API key
COGNEE_TIMEOUT_SECONDS=120            # Long timeout for cognification
COGNEE_ENABLED=true                   # Feature flag
```

## docker-compose.yml Update (Cognee dependency)

```yaml
# Cần thêm cognee container vào docker-compose.consolidated.yml
cognee:
  image: cogneeai/cognee:latest      # Python Cognee service
  ports:
    - "18000:8000"                   # Expose for debugging only
  environment:
    DATABASE_URL: "postgresql+asyncpg://vnp:vnppassword@postgres:5432/cognee"
    VECTOR_DB_URL: "postgresql+asyncpg://vnp:vnppassword@postgres:5432/cognee"
  depends_on:
    postgres:
      condition: service_healthy

kg-service:
  ...
  environment:
    COGNEE_URL: "http://cognee:8000"
  depends_on:
    - cognee
    - neo4j
```

## Acceptance Criteria

- [ ] `POST /v1/cognee/datasets` → calls Cognee Python API, returns dataset ID
- [ ] `POST /v1/cognee/datasets/{id}/data` → uploads data to Cognee
- [ ] `POST /v1/cognee/datasets/{id}/cognify` → triggers cognification, returns job ID
- [ ] `POST /v1/cognee/search` → returns semantic search results from Cognee
- [ ] When `COGNEE_URL` không accessible → return meaningful error (not panic)
- [ ] Feature flag `COGNEE_ENABLED=false` → routes return 503 with message
- [ ] Dataset metadata tracked locally in PostgreSQL
- [ ] `go build ./services/kg-service/...` passes
- [ ] Unit tests với mock Cognee client pass

## Ghi Chú

- **Cognee Python service phải chạy riêng** — kg-service là thin adapter layer
- **Circuit breaker** cho Cognee HTTP calls — nếu Cognee down, fail gracefully
- **Timeout dài** cho cognify operation — có thể mất vài phút cho large datasets
- 4 cognee services gốc giữ nguyên cho đến P4 cleanup
