---
id: ARCH-004
title: Merge memobase-ingestion + memobase-engine → memobase-pipeline
service: memobase-pipeline
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

`memobase-ingestion` manages Buffer Zone FSM and emits `memobase.buffer.ready` to trigger `memobase-engine`. Engine performs 3 fixed LLM calls (YOLO merge) and emits completion events. Both services share PostgreSQL schema and LLM adapter. NATS hop between them is unnecessary for a synchronous buffer→flush workflow.

## Kiến Trúc Mới

```
services/memobase-pipeline/
├── internal/
│   ├── domain/
│   │   ├── ingestion/      # Blob, BufferZone, BufferState (FSM)
│   │   └── engine/         # Profile, EventGist, MergeResult
│   ├── usecase/
│   │   ├── ingest/         # InsertBlob, GetBufferStatus, FlushBuffer
│   │   └── engine/         # ExtractTopics, MergeYOLO (3 LLM calls)
│   ├── adapter/grpc/
│   │   ├── ingestion_handler.go    # MemobaseIngestionService
│   │   └── engine_handler.go       # (internal, not exposed as gRPC)
│   └── infra/
```

**Key**: Buffer flush triggers engine processing via local function call. External events: `memobase.pipeline.completed` (→ memobase-context), `memobase.profile.changed` (→ memobase-context cache invalidation), `memobase.event.created` (→ vnp-platform event timeline).

## Acceptance Criteria

- [ ] AC-1: Buffer Zone FSM functions correctly (IDLE → PROCESSING → DONE)
- [ ] AC-2: YOLO merge executes exactly 3 LLM calls per flush
- [ ] AC-3: `memobase.pipeline.completed` emitted to memobase-context
- [ ] AC-4: `memobase.profile.changed` triggers cache invalidation in memobase-context
- [ ] AC-5: Token counting threshold (1024) maintained
