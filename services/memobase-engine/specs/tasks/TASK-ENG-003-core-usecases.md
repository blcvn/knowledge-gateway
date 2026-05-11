---
id: TASK-ENG-003
title: Implement Core Use Cases (Extraction & YOLO Merge)
service: memobase-engine
layer: Use Case (Layer 2)
status: Done
---

# Task: Implement Core Use Cases (Extraction & YOLO Merge)

## Objective
Implement the isolated, pure-business-logic use cases for profile extraction, YOLO merging, and event processing.

## Requirements
1. **Extract Profile Use Case (`internal/usecase/extract_profile.go`)**:
   - Implement LLM Call #2: Take `user_memo_str` and project profile schema, and extract structured profile facts.
   
2. **Merge Profile Use Case (`internal/usecase/merge_profile.go`)**:
   - Implement FEAT-002 (YOLO Merge Algorithm).
   - Implement LLM Call #3: Take extracted facts and existing profiles (indexed), and compute `MergeDecision` (add, update, delete).

3. **Process Event Use Case (`internal/usecase/process_event.go`)**:
   - Implement FEAT-003 (Event Processing).
   - Generate tags (conditional LLM if `event_tags` configured).
   - Generate embeddings and structure the `UserEvent` for persistence.

## Constraints
- These use cases must not contain orchestration logic; they are specific steps in the pipeline.
- Implement strictly against the `port` interfaces.
