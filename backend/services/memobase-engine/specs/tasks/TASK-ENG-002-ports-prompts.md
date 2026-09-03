---
id: TASK-ENG-002
title: Implement Use Case Ports and Prompt Templates
service: memobase-engine
layer: Use Case (Layer 2) & Infrastructure (Layer 4)
status: Done
---

# Task: Implement Use Case Ports and Prompt Templates

## Objective
Define the Input/Output ports in the Use Case layer to support the processing pipeline and implement the multi-language Prompt Template system in the Infrastructure layer (ARCH-001).

## Requirements
1. **Use Case Ports (`internal/usecase/port/`)**:
   - `input.go`: Define primary driving interfaces like `BufferProcessor` and `ProfileExtractor`.
   - `output.go`: Define secondary driven interfaces like `LLMClient`, `EmbedderClient`, and `ProfileStore`.
   - `dto/`: Define DTOs (`request.go`, `response.go`).

2. **Prompt Templates (`internal/infra/prompt/`)**:
   - Implement the prompt loader system supporting EN/ZH languages based on project config.
   - `summary_entry_chats.go`: Prompt for LLM #1 (Entry Chat Summary).
   - `extract_profile.go`: Prompt for LLM #2 (Extract Topics).
   - `merge_profile_yolo.go`: Prompt for LLM #3 (Merge YOLO).

## Constraints
- Usecase ports must depend ONLY on the Domain layer.
- Prompt system must strictly implement the 3 Fixed LLM Calls as described in `api.md` and `architecture.md`.
