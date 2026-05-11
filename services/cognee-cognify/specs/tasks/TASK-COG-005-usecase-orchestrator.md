---
id: TASK-COG-005
title: Implement Usecase Pipeline Orchestrator
feature: FEAT-COG-001
status: Done
---
# Task: Implement Usecase Pipeline Orchestrator

## Objective
Implement the central pipeline orchestrator that coordinates the 8 execution stages of the knowledge graph pipeline.

## Files to Create/Modify
- `internal/usecase/cognify.go`

## Requirements
- Implement the `CognifyOrchestrator` struct fulfilling the `CognifyUseCase` interface.
- Initialize the `CognifyJob` via `JobRepository`.
- Orchestrate the 8 stages sequentially: Classify -> Chunk -> ExtractEntities -> ExtractRels -> Deduplicate -> BuildGraph -> Embed -> Summarize.
- Maintain job state: Persist state changes via `JobRepository` whenever calling `job.AdvanceStage`, `job.Complete`, or `job.Fail`.
- Calculate and record comprehensive `PipelineMetrics` (e.g., tokens used, duration, total entities).
- Trigger `PipelineCompletedEvent` via `EventPublisher` upon successful completion.
- Write a full orchestrator integration test using mock stages and mock ports.
