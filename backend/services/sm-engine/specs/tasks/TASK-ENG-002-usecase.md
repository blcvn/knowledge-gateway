---
id: TASK-ENG-002
title: Usecases & Orchestration
service: sm-engine
status: Done
priority: P0
created: 2026-05-11
---

# Usecases & Orchestration

## Objective
Implement DocumentUseCase, MemoryUseCase, ProfileUseCase. Handle the local orchestration of document -> memory -> profile.

## Requirements
- Strictly follow the Clean Architecture definitions from `specs/tdd.md` and `docs/architecture.md`.
- No new features or architectures are to be created; only execute the documented design.
- Token-efficient execution: keep implementations focused entirely on the `usecase` layer/component.

## Acceptance Criteria
- [x] Code compiles without errors.
- [x] Unit tests written and passing (if applicable).
- [x] 100% alignment with the `specs/tdd.md` document for `sm-engine`.
