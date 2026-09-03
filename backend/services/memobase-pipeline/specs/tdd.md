---
id: TDD-memobase-pipeline
title: Technical Design Document — memobase-pipeline
service: memobase-pipeline
version: 1.1.0
status: Ready
created: 2026-05-10
updated: 2026-05-11
linked_sol: SOL-001
linked_adr: ADR-0001
---

# memobase-pipeline — Technical Design Document

## 1. Overview

The `memobase-pipeline` service is the result of consolidating `memobase-ingestion` and `memobase-engine` into a single, unified pipeline. It handles the ingestion of user memory blobs (conversations, documents, summaries), manages the Buffer Zone FSM to batch data efficiently, and executes a fixed 3-LLM-call extraction and merge pipeline (YOLO merge) to produce structured user profiles and temporal event gists.

## 2. Core Algorithms

Based on the Memobase reference implementation, the pipeline follows these key algorithms:

### 2.1. Buffer Zone FSM (Token-Aware Batching)

The ingestion phase prevents hot-path LLM calls by accumulating raw blobs until an optimal token threshold is reached.

- **State Machine**: `IDLE → PROCESSING → DONE / FAILED`
  - `IDLE`: Blobs are accumulating. If `token_sum >= 1024` or idle duration `> 1h`, state transitions to `PROCESSING`.
  - `PROCESSING`: The YOLO merge pipeline is executing locally.
  - `DONE`: Pipeline succeeded, profiles updated, blobs can be marked done/deleted.
  - `FAILED`: Pipeline failed, blobs remain in buffer for retry.

### 2.2. Memory Processing Pipeline (Chat Modal)

When the buffer flushes, the accumulated blobs are processed sequentially. The pipeline enforces exactly **3 fixed LLM calls** per flush for predictable cost and performance.

```text
Step 1: Entry Summary
  -> LLM Call #1: `entry_chat_summary`
  -> Input: Raw buffered blobs (conversations, etc.)
  -> Output: `user_memo_str` (A concise summary string of the entire buffer)

Step 2a: Profile Processing
  -> LLM Call #2: `extract_topics`
  -> Input: `user_memo_str`
  -> Output: Extracted structured facts (Topic, Subtopic, Content, Attributes)
  
Step 2b: YOLO Merge
  -> LLM Call #3: `merge_yolo`
  -> Input: Extracted facts + Existing user profiles
  -> Output: Add/Update/Delete decisions for each profile slot

Step 2c: Organize Profiles (No LLM)
  -> `organize_profiles`: Reorganize subtopics if the number of topics exceeds limits.

Step 2d: Re-summary (Conditional)
  -> `re_summary`: Re-summarize if a profile slot's content exceeds the configured token limit.

Step 3: Event Tagging (Conditional)
  -> `tag_event`: Tag the event with configurable categories.

Step 4: Event Persistence & Embedding
  -> `append_user_event`: Store the resulting temporal event and generate vector embeddings (via embedding adapter, e.g., OpenAI, Jina) for semantic retrieval.
```

## 3. Architecture Layer Mapping

- **Domain Layer**: Defines `BufferZone`, `BufferState` FSM, `Profile`, `EventGist`, and the core `MergeResult` structures.
- **Usecase Layer**: 
  - `ingest`: Handles `InsertBlob` and triggers `FlushBuffer` locally.
  - `engine`: Implements the 3-step LLM extraction (`ExtractTopics`, `MergeYOLO`) and event summarization.
- **Adapter Layer**: 
  - **gRPC**: Exposes `MemobaseIngestionService` for external insertion.
  - **PostgreSQL**: Stores Blobs, BufferState, Profiles, and EventGists (with `pgvector`).
  - **Redis**: Fast lookup for Buffer state FSM and token counting cache.
  - **Event/NATS**: Publishes `memobase.pipeline.completed` and `memobase.profile.changed` out to the ecosystem.
- **Infrastructure Layer**: Bifrost integration for LLM calls.

## 4. Acceptance Criteria

- [ ] AC-1: Buffer Zone FSM functions correctly (`IDLE → PROCESSING → DONE`).
- [ ] AC-2: YOLO merge executes exactly 3 LLM calls per flush.
- [ ] AC-3: `memobase.pipeline.completed` and `memobase.profile.changed` events are emitted correctly via NATS.
- [ ] AC-4: Token counting uses the equivalent tiktoken algorithm and respects the `1024` threshold.
- [ ] AC-5: Blobs are properly transitioned or cleaned up upon successful completion.
