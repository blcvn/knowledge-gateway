---
id: TASK-RES-004
service: ov-resource
status: Done
---

# TASK-RES-004: Implement Usecase Orchestration

## Objective
Implement the core business workflows (`internal/usecase/`) exactly as defined in `tdd.md` and `api.md`.

## Requirements
1. **Ingest Pipeline (`ingest.go`)**:
   - Check file size against `MAX_INGESTION_SIZE_MB` (default 100MB).
   - Detect parser (extension or `force_parser` override).
   - Generate `[]Chunk` via ParserPort.
   - Write chunks to `ov-fs` target path (L0/L1 abstracts) via `FileWriterPort`.
   - Update `ov_resources` DB record (update state, chunks, parse_duration_ms, token count, content_hash).
   - Publish `ov.resource.ingested` via `EventPublisherPort`.
2. **Parse Workflow (`parse.go`)**:
   - Apply `chunk_size` and `chunk_overlap` configs dynamically if provided in request. Return structured chunks without persisting.
3. **Watch Manager (`watch.go`)**:
   - Background goroutine polling directories based on `poll_interval_ms` (fallback to `WATCH_DEFAULT_POLL_MS` = 30000).
   - Limit concurrent tasks using `WATCH_MAX_TASKS` configuration.
   - Monitor files matching `patterns` glob. Auto-trigger `IngestUseCase` when modified/created.
4. **Refresh Workflow (`refresh.go`)**:
   - Re-parse and re-index resources by paths. Override hash dedup if `force=true`.

## Dependencies
- All internal ports.
