---
id: TASK-COG-003
title: Implement Pipeline Stages 1 to 4
feature: FEAT-COG-001
status: Done
---
# Task: Implement Pipeline Stages 1 to 4

## Objective
Implement the first half of the cognitive pipeline stages (Classify, Chunk, Extract Entities, Extract Relationships).

## Files to Create/Modify
- `internal/usecase/classify.go`
- `internal/usecase/chunk.go`
- `internal/usecase/extract_entities.go`
- `internal/usecase/extract_rels.go`

## Requirements
- Define a common `Stage` interface with a `Name() string` and `Execute(ctx, job)` method if not already defined.
- **Stage 1 (Classify)**: Use `LLMClient` to sample content and determine the best `ChunkingStrategy` (JSON structured output).
- **Stage 2 (Chunk)**: Segment the input text into chunks based on the resolved strategy.
- **Stage 3 (Extract Entities)**: Iterate over chunks and use `LLMClient` to perform Named Entity Recognition (NER) to extract `[]Entity`.
- **Stage 4 (Extract Relationships)**: Use `LLMClient` to discover relationships between extracted entities across chunks, yielding `[]Relationship`.
- Include unit tests with mock LLM responses using deterministic JSON fixtures.
