---
id: TASK-SRCH-001
title: "Implement Domain Layer for ov-search"
service: ov-search
status: Done
priority: High
created_at: 2026-05-11
---

# TASK-SRCH-001: Implement Domain Layer for ov-search

## Objective
Establish the foundational Domain Layer (Layer 1) for the `ov-search` service, including models, repository interfaces, events, and custom errors.

## Requirements

1. **Domain Models (`internal/domain/model/`)**:
   - Create `search_result.go` defining `SearchResult`, `Score`, and `MatchedContext`.
   - Create `hotness.go` defining `HotnessScore` and `DecayConfig`.
   - Create `embedding.go` defining `EmbeddingVector` (1536-dim) and `UpsertPayload`.
   - Create `context_type.go` defining `ContextType`, `QueryPlan`, and `TypedQuery`.

2. **Repository Interfaces (`internal/domain/repository/`)**:
   - Define `VectorRepository` interface in `vector_repo.go` with methods for upserting, searching, and deleting embeddings.
   - Define `HotnessRepository` interface in `hotness_repo.go`.

3. **Events and Errors**:
   - Define domain events in `event.go` (e.g., `SearchCompleted`, `HotnessUpdated`).
   - Define custom domain errors in `errors.go` (e.g., `IndexNotFound`, `EmbeddingFailed`).

## Constraints
- Follow the 4-layer Clean Architecture.
- The Domain layer must have zero external dependencies.
- Stick strictly to the structure provided in the architecture document.
