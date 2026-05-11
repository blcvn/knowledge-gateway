---
id: TASK-ENG-004
title: Implement Pipeline Orchestration Use Case
service: memobase-engine
layer: Use Case (Layer 2)
status: Done
---

# Task: Implement Pipeline Orchestration Use Case

## Objective
Implement the main `ProcessBufferUseCase` (FEAT-001) that orchestrates the entire 3-LLM-call pipeline and the subsequent post-processing steps.

## Requirements
1. **Process Buffer Use Case (`internal/usecase/process_buffer.go`)**:
   - Fetch blobs from storage.
   - Execute LLM #1: `entry_chat_summary`.
   - Execute Parallel processing: LLM #2 (`extract_topics`) and Event Processing (`tag_event`, `append_user_event`).
   - Execute LLM #3: `merge_yolo` based on extracted facts.
   
2. **Post-Processing (No LLM path)**:
   - Implement `organize_profiles`: Merge similar subtopics if they exceed `MAX_PROFILE_SUBTOPICS`.
   - Implement `re_summary`: Summarize if profile content exceeds `MAX_PRE_PROFILE_TOKEN_SIZE` (conditional LLM).
   - Persist data (upsert profiles, store events) via output ports.
   - Dispatch notifications via publisher port.

## Constraints
- The pipeline orchestration must be strictly bound to the sequence detailed in `tdd.md` and `api.md`.
- All cross-component communication goes through injected ports.
