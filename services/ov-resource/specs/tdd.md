---
id: TDD-ov-resource
title: Technical Design — ov-resource
service: ov-resource
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: OpenViking
---

# Technical Design — ov-resource

> **Group**: OpenViking | **gRPC Port**: 9054 | **Origin**: OpenViking (ResourceManager + WatchManager)

## 1. Service Overview

Resource ingestion pipeline: multi-format parsing (tree-sitter for code, section-based for Markdown, page-based for PDF/DOCX), automatic chunking, writing to ov-fs, and watch manager for external directory monitoring.

**Origin mapping**: `openviking/resource/resource_manager.py` + `openviking/resource/watch_manager.py` + `openviking/parse/` (parser registry).

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── resource.go              # Resource, ResourceType, IngestionResult
│   ├── chunk.go                 # Chunk, ChunkMetadata (position, tokens, AST info)
│   ├── watch.go                # WatchTask, WatchEvent, WatchStatus
│   └── parser.go               # ParserConfig, ParserType enum
├── repository/
│   ├── resource_repo.go         # ResourceRepository
│   └── watch_repo.go            # WatchRepository
├── event.go                     # ResourceIngested domain event
└── errors.go                    # UnsupportedFormat, ParseFailed, IngestFailed
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── ingest.go                    # Full pipeline: parse → chunk → write → notify
├── parse.go                     # Parse-only (returns chunks without writing)
├── watch.go                     # Watch manager lifecycle (create/pause/resume/delete)
├── refresh.go                   # Re-parse stale resources
├── port/
│   ├── input.go                # IngestUseCase, ParseUseCase, WatchUseCase
│   └── output.go              # FileWriterPort, ParserPort, EventPublisherPort
└── dto/
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/handler.go              # OvResourceService gRPC
├── event/publisher.go           # NATS: ov.resource.ingested
├── parser/
│   ├── registry.go             # Extension → parser routing (pkg/parse/)
│   ├── treesitter.go           # tree-sitter adapter for code files
│   ├── markdown.go             # Markdown section parser
│   └── document.go             # PDF/DOCX page parser
└── client/
    └── fs_client.go             # ov-fs gRPC (write parsed chunks)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/
│   ├── resource_repo.go         # PostgreSQL resource repository
│   └── watch_repo.go            # Watch task persistence
├── config/config.go
└── wire/wire.go
```

## 3. gRPC API

```protobuf
service OvResourceService {
  rpc Ingest(IngestRequest) returns (IngestResponse);
  rpc Parse(ParseRequest) returns (ParseResponse);
  rpc Watch(WatchRequest) returns (stream WatchEvent);
  rpc Refresh(RefreshRequest) returns (RefreshResponse);
}
```

## 4. NATS Events

### Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `ov.resource.ingested` | `{path, account_id, chunks, parser_type}` | After ingest completes |

## 5. Data Model

- **ov_resources**: Ingestion metadata (source/target paths, parser, chunks, hash, status)
- **ov_watch_tasks**: Watch task definitions (source, target, patterns, interval, status)

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Purpose |
|---------|-----------|----------|---------|
| ov-fs | Outbound | gRPC | Write parsed chunks + file content |
| ov-search | Outbound (NATS) | Async | `ov.resource.ingested` → embed + index |
| pkg/parse | Import | Go | Parser registry (tree-sitter, markdown, PDF) |

## 7. Observability

- **Metrics**: Ingest count/duration by parser, chunks produced, watch polls, parse errors
- **Traces**: OTel spans: `ov-resource.Ingest` (parse → write → notify sub-spans)
- **Health**: gRPC Health v1 + HTTP `/healthz` on port 9107

## 8. Multi-Tenancy

- `account_id` scopes all resources and watch tasks

---

> **Next Steps**: Decompose into FEAT specs: FEAT-001 (Ingestion Pipeline), FEAT-002 (Parser Registry), FEAT-003 (Watch Manager), FEAT-004 (Refresh Mechanism).
