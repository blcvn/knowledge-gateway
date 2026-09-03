# memobase-pipeline — Architecture

> **Pattern**: Pipeline Merge (Buffer Zone FSM + YOLO Engine → Single Binary)

---

## Internal Layer Structure

```
services/memobase-pipeline/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── ingestion/      # Blob, BufferZone, BufferState (FSM), TokenCounter
│   │   └── engine/         # Profile, EventGist, MergeResult, TopicCategory
│   ├── usecase/
│   │   ├── ingest/         # InsertBlob, GetBufferStatus, FlushBuffer
│   │   │                   # FlushBuffer triggers engine.MergeYOLO LOCALLY
│   │   └── engine/         # ExtractTopics, MergeYOLO (3 LLM calls), GenerateGist
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── ingestion_handler.go   # MemobaseIngestionService (proto unchanged)
│   │   ├── repository/
│   │   │   └── postgres/   # Blobs, BufferState, Profiles, EventGists
│   │   └── event/nats/
│   │       └── publisher.go   # memobase.pipeline.completed, memobase.profile.changed
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       ├── llm/bifrost.go     # YOLO merge: exactly 3 LLM calls
│       └── wire/wire.go
```

## Key Design Decisions

1. **Buffer → Engine is local**: Buffer flush triggers YOLO merge via local function call instead of NATS event. Eliminates `memobase.buffer.ready` subject.
2. **3 fixed LLM calls**: YOLO merge pattern MUST use exactly 3 LLM calls per flush — this is a cost-control invariant. Local call does not change this constraint.
3. **Token threshold**: Buffer flushes when accumulated tokens ≥ 1024 (configurable). FSM state transitions tracked in Redis for fast lookups.
4. **External events preserved**: `memobase.pipeline.completed` → memobase-context (for context assembly), `memobase.profile.changed` → memobase-context (cache invalidation), `memobase.event.created` → vnp-platform (event timeline).

## Memory Processing Pipeline Algorithm

The pipeline executes a deterministic, fixed-cost sequence for extracting profiles and events:

1. **Entry Summary (LLM #1)**: Summarizes the entire token-buffered conversational input into a single `user_memo_str`.
2. **Profile Processing**:
   - **Topic Extraction (LLM #2)**: Parses `user_memo_str` into structured facts (Topic, Subtopic, Content).
   - **YOLO Merge (LLM #3)**: Compares extracted facts with existing profiles to make one-shot Add/Update/Delete decisions.
   - **Organize Profiles**: Subtopic reorganization to ensure topic budgets are maintained (no LLM).
   - **Re-summary**: Triggered only if a merged profile slot exceeds the hard token limit (conditional LLM).
3. **Temporal Processing**:
   - **Event Tagging**: Optional semantic classification of the event.
   - **Persistence & Embedding**: Event and gist data saved and embedded via vector adapters (pgvector).

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| PostgreSQL + pgvector | Blob storage, profiles, event gists, embeddings |
| Redis | Buffer state FSM, token count cache |
| NATS | Emit pipeline/profile events → memobase-context, vnp-platform |
| Bifrost (LLM) | YOLO merge (3 calls: extract, merge, summarize) |

## Known Limitations

- YOLO merge is synchronous and LLM-bound — need bulkhead to protect fast-path blob insertion
- Token counting depends on LLM tokenizer — must match model being used for extraction
