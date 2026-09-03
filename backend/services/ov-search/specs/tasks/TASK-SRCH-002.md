---
id: TASK-SRCH-002
title: "Implement Usecase Layer for ov-search"
service: ov-search
status: Done
priority: High
created_at: 2026-05-11
---

# TASK-SRCH-002: Implement Usecase Layer for ov-search

## Objective
Implement the Usecase Layer (Layer 2) containing the core search algorithms, tiered context loading, hotness scoring, and embedding operations.

## Requirements

1. **Ports (`internal/usecase/port/`)**:
   - Define input ports: `SearchUseCase`, `EmbeddingUseCase`.
   - Define output ports: `EmbedderPort`, `FileReaderPort`, `RerankPort`.

2. **Hierarchical Search Pipeline (`internal/usecase/hierarchical_search.go`)**:
   - Implement the 7-step pipeline: Query Intent Analysis, Dense/Sparse Vector Search, Hierarchical Score Propagation (Child -> Parent), Hotness Score Integration, Convergence Detection, Cross-Encoder Reranking, Tiered Context Loading.

3. **Context Retrieval (`internal/usecase/context_retrieval.go`)**:
   - Implement tiered loading depending on required depth (L0 - Abstract, L1 - Overview, L2 - Full Content).

4. **Hotness Scoring & Decay (`internal/usecase/hotness.go`)**:
   - Implement exponential decay logic: `H(t) = H_0 * exp(-λ * Δt)`.
   - Implement hotness boost for active session references.
   - Implement a background worker to periodically recompute hotness scores (driven by `HOTNESS_RECOMPUTE_INTERVAL_M`).

5. **Embedding Operations (`internal/usecase/embedding_ops.go`)**:
   - Implement `UpsertEmbedding` and `DeleteEmbedding` use cases.

## Constraints
- Ensure strict adherence to the algorithms described in the architectural documentation.
- Do not introduce external dependencies; use defined ports for external interactions.
