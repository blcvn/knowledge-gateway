---
id: DOC-S01
service: ov-resource
version: 1.1.0
status: Deprecated
created: 2026-05-09
updated: 2026-05-11
owner: VNP Memory — OpenViking Team
---

# ov-resource (DEPRECATED)

> **⚠️ DEPRECATION NOTICE**: This service has been merged into `ov-storage` per `ARCH-006`. The code and documentation here are kept for historical reference only. Please refer to `ov-storage` for the active implementation.

> **Group**: OpenViking | **gRPC Port**: 9054 | **Health Port**: 9107 | **Origin**: OpenViking

## Purpose

Resource ingestion pipeline: **parse files** (tree-sitter for code, markdown, PDF/DOCX), **generate embeddings**, **watch for changes**, and **refresh stale resources**. Replaces Python `openviking/resource/watch_manager.py` and `openviking/parse/`.

### Business Capability

- **Ingestion Pipeline**: Accept external files → parse → chunk → write to ov-fs → index via ov-search
- **Multi-Format Parsing**: Code (Go/Py/JS/TS via tree-sitter), Markdown (section-based), PDF/DOCX (page-based), Plain text (paragraph)
- **Watch Manager**: Monitor external directories/repositories for changes and auto-ingest
- **Watch Scheduler**: Configurable polling intervals per watch task
- **Refresh**: Re-parse and re-index stale resources when formats or parsers change

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Parsers**: tree-sitter Go bindings (`pkg/parse/`), custom Markdown, PDF/DOCX libraries
- **Database**: PostgreSQL (watch tasks, ingestion metadata)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-ov-resource
make run-ov-resource
docker compose up ov-resource postgresql nats
```

## API Surface

### gRPC Service

```protobuf
service OvResourceService {
  rpc Ingest(IngestRequest) returns (IngestResponse);
  rpc Parse(ParseRequest) returns (ParseResponse);
  rpc Watch(WatchRequest) returns (stream WatchEvent);
  rpc Refresh(RefreshRequest) returns (RefreshResponse);
}
```

### Parse Engine

| Format | Parser | Chunking Strategy |
|--------|--------|-------------------|
| Code (Go/Py/JS/TS) | tree-sitter | AST-aware (function/class/method) |
| Markdown | Custom | Section-based (by heading level) |
| PDF/DOCX | Go libraries | Page-based with overlap |
| Plain text | — | Paragraph (double newline) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| ov-fs | gRPC | Write parsed file content |
| ov-search | NATS | Notify `ov.resource.ingested` for embedding + indexing |
| Bifrost (LLM) | gRPC | Embedding generation for parsed chunks |
| PostgreSQL | SQL | Watch task metadata, ingestion history |

## NATS Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `ov.resource.ingested` | Publish | Resource parsed and written to ov-fs |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — OpenViking
