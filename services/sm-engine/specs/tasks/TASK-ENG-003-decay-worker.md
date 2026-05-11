---
id: TASK-ENG-003
title: Ebbinghaus Decay Worker
service: sm-engine
status: Done
priority: P0
created: 2026-05-11
---

# Ebbinghaus Decay Worker

## Objective
Implement the background worker to recalculate forgetting curves for memories and soft-delete forgotten ones.

## Requirements
- Strictly follow the Clean Architecture definitions from `specs/tdd.md` and `docs/architecture.md`.
- No new features or architectures are to be created; only execute the documented design.
- Token-efficient execution: keep implementations focused entirely on the `decay-worker` layer/component.

## Acceptance Criteria
- [x] Code compiles without errors.
- [x] Unit tests written and passing (if applicable).
- [x] 100% alignment with the `specs/tdd.md` document for `sm-engine`.
