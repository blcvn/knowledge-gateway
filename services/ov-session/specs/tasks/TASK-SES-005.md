---
id: TASK-SES-005
title: Implement Two-Phase Commit - Phase 1 (Archive)
status: Done
---

# Task: Implement Two-Phase Commit - Phase 1 (Archive)

## Objective
Implement the first phase of the Two-Phase Commit algorithm: session compression and archiving to `ov-fs`.

## Requirements
1. **Compressor Usecase** (`internal/usecase/compressor.go`):
   - Implement `SessionCompressor` supporting both v1 (legacy) and v2 (template system) versions.
   - Integrate with the Bifrost LLM client (`llm_client.go`) to generate the summarization based on the session's uncommitted messages.
2. **Commit Usecase - Phase 1 Logic** (`internal/usecase/commit.go`):
   - Load uncommitted messages for the session.
   - Execute the compression process.
   - Use the `fs_client.go` to write the compressed archive to `ov-fs` at `viking://{account}/{user}/sessions/archives/`.
   - Update `ov_sessions` status to `committed` and set `archive_path` and `committed_at`.
3. **NATS Event Publishing** (`internal/adapter/event/publisher.go`):
   - Publish the `ov.session.committed` event containing `{session_id, account_id, archive_path}` via NATS to trigger `ov-search` hotness boosting.

## Acceptance Criteria
- [x] The compressor correctly utilizes the LLM client to generate summaries.
- [x] The archive file is successfully written to `ov-fs` with the correct path namespace.
- [x] The session status is reliably updated in the database.
- [x] The `ov.session.committed` NATS event is successfully dispatched.
