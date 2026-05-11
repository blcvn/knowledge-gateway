---
id: TASK-PIPE-001
title: Implement Domain Layer
layer: domain
status: Done
---

## Objective
Implement the Domain Layer for memobase-pipeline.

## Requirements
1. **Ingestion Domain**: Implement `Blob`, `BufferZone`, `BufferState` FSM (IDLE → PROCESSING → DONE / FAILED), and `TokenCounter`.
2. **Engine Domain**: Implement `Profile`, `EventGist`, `MergeResult`, and `TopicCategory`.
3. **FSM Rules**: Define logic for `BufferState` transition (e.g., IDLE → PROCESSING when `token_sum >= 1024` or idle duration > 1h).

## Constraints
- Pure Go structs and interfaces. No external dependencies or DB logic.
- Follow Clean Architecture.
