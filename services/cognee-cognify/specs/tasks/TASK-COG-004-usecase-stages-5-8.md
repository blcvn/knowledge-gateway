---
id: TASK-COG-004
title: Implement Pipeline Stages 5 to 8
feature: FEAT-COG-001
status: Done
---
# Task: Implement Pipeline Stages 5 to 8

## Objective
Implement the second half of the cognitive pipeline stages (Deduplicate, Build Graph, Embed, Summarize).

## Files to Create/Modify
- `internal/usecase/deduplicate.go`
- `internal/usecase/build_graph.go`
- `internal/usecase/embed.go`
- `internal/usecase/summarize.go`

## Requirements
- **Stage 5 (Deduplicate)**: Resolve identical entity pairs using LLM checks or vector similarity. Must support skipping via `skip_dedup=true` config. Merge duplicates and maintain provenance.
- **Stage 6 (Build Graph)**: Persist extracted Entities and Relationships to Neo4j via `GraphRepository`.
- **Stage 7 (Embed)**: Generate vector embeddings for the text chunks and entity descriptions via `EmbedderClient` and save them via `VectorRepository`.
- **Stage 8 (Summarize)**: Group entities by community (provided by `GraphRepository` or analyzed locally) and use `LLMClient` to generate summarized insights for each community.
- Include unit tests utilizing mock interfaces.
