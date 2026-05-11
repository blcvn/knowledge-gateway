---
id: TASK-PIPE-002
title: Implement Usecase Layer
layer: usecase
status: Done
---

## Objective
Implement the Usecase Layer for `ingest` and `engine` logic.

## Requirements
1. **Ingest Usecase**:
   - `InsertBlob`: Insert blob, update token counts.
   - `GetBufferStatus`: Retrieve current FSM state.
   - `FlushBuffer`: Trigger YOLO merge locally when token threshold is met.
2. **Engine Usecase (YOLO Pipeline)**:
   - `ExtractTopics`: Step 1 & 2a (Entry Summary and Topic Extraction).
   - `MergeYOLO`: Step 2b (Compare facts with profiles -> Add/Update/Delete).
   - `OrganizeProfiles` & `ReSummary`: Step 2c & 2d (Subtopic reorganization, conditional summary).
   - `GenerateGist`: Step 3 & 4 (Event tagging, persistence, and embedding trigger).

## Constraints
- `FlushBuffer` must trigger `MergeYOLO` locally (no NATS event).
- Strict invariant: The pipeline must execute exactly 3 LLM calls per flush.
