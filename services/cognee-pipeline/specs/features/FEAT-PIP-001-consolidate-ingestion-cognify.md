---
id: FEAT-PIP-001
title: Pipeline Service — Consolidate Ingestion + Cognify (Single Binary)
service: cognee-pipeline
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_feat: FEAT-ING-003, FEAT-COG-003
---

## Mục Tiêu

Consolidate cognee-ingestion + cognee-cognify into a single binary `cognee-pipeline`. Both gRPC services exposed from one process. NATS `cognee.data.ingested` event replaced by local function call within the same process.

## Bối Cảnh Nghiệp Vụ

For small/medium deployments, running separate ingestion and cognify services is overhead. `cognee-pipeline` provides the same functionality in a single binary — proto interfaces unchanged, deployment topology simplified.

## Scope

### In Scope
- Merge ingestion + cognify domain/usecase/adapter code via shared module imports
- Single `cmd/server/main.go` bootstrapping both gRPC services
- Local function call: ingestion → cognify (instead of NATS event)
- Single Dockerfile, single health endpoint
- Dual gRPC service registration (CogneeIngestionService + CogneeCognifyService on same port)
- Combined Wire injector
- Shared database connection pools

### Out of Scope
- Changing proto definitions (backward compatible)
- Re-implementing business logic (reuse from ingestion + cognify)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
services/cognee-pipeline/
├── cmd/server/main.go                    # Entry point — registers both services
├── internal/
│   ├── domain/
│   │   ├── ingestion/                    # → imports from cognee-ingestion/domain
│   │   │   └── entity.go                # Dataset, DataItem (or re-export)
│   │   └── cognify/                     # → imports from cognee-cognify/domain
│   │       └── entity.go                # CognifyJob, Entity, Relationship
│   ├── usecase/
│   │   ├── ingest/                      # Ingestion usecases
│   │   │   └── ingest.go               # IngestFile → triggers local cognify
│   │   └── cognify/                     # Cognify usecases
│   │       └── cognify.go              # 8-stage pipeline
│   │   └── port/
│   │       └── interfaces.go           # Combined port interfaces
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── ingestion_handler.go    # CogneeIngestionService handler
│   │   │   └── cognify_handler.go      # CogneeCognifyService handler
│   │   ├── repository/
│   │   │   ├── postgres/               # Shared PG pool: datasets + jobs
│   │   │   ├── neo4j/                  # Knowledge graph
│   │   │   └── pgvector/               # Vector embeddings (replaces Qdrant)
│   │   └── event/nats/
│   │       └── publisher.go            # cognee.pipeline.completed only
│   └── infra/
│       ├── config/config.go            # Combined config
│       ├── server/grpc.go              # Registers both services
│       ├── telemetry/
│       └── wire/wire.go
├── Dockerfile
├── go.mod
└── Makefile
```

### Key Difference from Standalone

```go
// In standalone: ingestion publishes NATS event → cognify subscribes
// In pipeline: ingestion triggers cognify directly

func (uc *IngestFileUseCase) Execute(ctx context.Context, req dto.IngestFileReq) (*dto.IngestResult, error) {
    // ... normal ingestion flow ...
    
    // Instead of: eventPublisher.PublishDataIngested(...)
    // Direct call: cognifyUseCase.Execute(...)
    if uc.autoTriggerCognify {
        go uc.cognifyUseCase.Execute(ctx, dto.TriggerCognifyReq{
            DatasetID: result.DatasetID,
            TenantID:  req.TenantID,
        })
    }
}
```

## Acceptance Criteria

- [ ] AC-1: Given single binary, When started, Then both CogneeIngestionService + CogneeCognifyService are registered on same gRPC port
- [ ] AC-2: Given file upload via ingestion API, When processed, Then cognify pipeline auto-triggers via local call (no NATS)
- [ ] AC-3: Given standalone cognify API call, When triggered, Then pipeline executes same as standalone service
- [ ] AC-4: Proto definitions unchanged — existing clients work without modification
- [ ] AC-5: Single health endpoint reports status for both ingestion + cognify subsystems
- [ ] AC-6: Docker image ≤50MB, single process

## Test Requirements

- **Integration test**: Upload file → verify graph created (end-to-end within single process)
- **Compatibility test**: Same gRPC requests work against both pipeline and standalone services
- **Coverage**: ≥ 80%
