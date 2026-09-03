---
id: TASK-SEA-002
title: Search Usecases
service: sm-search
status: Done
priority: P0
created: 2026-05-11
---

# Search Usecases

## Objective
Implement HybridSearchUseCase (parallel vector + fulltext), MemorySearchUseCase, RAGCompleteUseCase, and QueryRewriteUseCase.

## Requirements
- Strictly follow the Clean Architecture definitions from `specs/tdd.md` and `docs/architecture.md`.
- No new features or architectures are to be created; only execute the documented design.
- Token-efficient execution: keep implementations focused entirely on the `usecase` layer/component.

## Acceptance Criteria
- [x] Code compiles without errors.
- [x] Unit tests written and passing (if applicable).
- [x] 100% alignment with the `specs/tdd.md` document for `sm-search`.
