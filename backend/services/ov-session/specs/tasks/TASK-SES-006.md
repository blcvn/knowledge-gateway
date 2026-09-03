---
id: TASK-SES-006
title: Implement Two-Phase Commit - Phase 2 (Extract & Deduplicate)
status: Done
---

# Task: Implement Two-Phase Commit - Phase 2 (Extract & Deduplicate)

## Objective
Implement the final phase of the session commit lifecycle, extracting specific memory categories and deduplicating them before final storage.

## Requirements
1. **Memory Extractor Usecase** (`internal/usecase/memory_extractor.go`):
   - Implement the extraction process using the LLM client to scan the session archive and classify candidate memories into 5 domains: Persona, Project Context, Decisions, Action Items, and Errors/Blockers.
2. **Memory Deduplicator Usecase** (`internal/usecase/memory_deduplicator.go`):
   - Implement the Semantic Deduplication Algorithm comparing new candidate memories with existing ones.
   - Action mappings:
     - `sim == 1.0` -> `SKIP`
     - `sim > 0.85` -> `MERGE` (requires LLM fusion)
     - `sim > 0.60` -> `CREATE`
     - `sim < 0.60` -> `CREATE`
     - Invalidated -> `ARCHIVE`
3. **Commit Usecase - Phase 2 Logic** (`internal/usecase/commit.go`):
   - Orchestrate extraction and deduplication.
   - Persist final memories to `ov_extracted_memories`.
   - Write physical memory files to `ov-fs` via `fs_client.go`.
4. **NATS Event Publishing** (`internal/adapter/event/publisher.go`):
   - Publish the `ov.session.memory.extracted` event containing `{session_id, memories[], fs_paths[]}`.

## Acceptance Criteria
- [x] Memory extractor accurately utilizes the Bifrost LLM client to extract the 5 defined categories.
- [x] Deduplication algorithm correctly categorizes and assigns actions based on similarity thresholds.
- [x] Unique/Merged memories are persisted in the database and `ov-fs`.
- [x] The `ov.session.memory.extracted` NATS event is properly broadcasted to trigger asynchronous memory ingestion.
