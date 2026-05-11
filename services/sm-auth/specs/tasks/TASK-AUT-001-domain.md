---
id: TASK-AUT-001
title: Domain Models & Crypto Algorithms
service: sm-auth
status: Done
priority: P0
created: 2026-05-11
---

# Domain Models & Crypto Algorithms

## Objective
Implement AuthContext, APIKey. Implement Base58, SHA-256/Argon2id hashing, and RS256 JWT validation algorithms.

## Requirements
- Strictly follow the Clean Architecture definitions from `specs/tdd.md` and `docs/architecture.md`.
- No new features or architectures are to be created; only execute the documented design.
- Token-efficient execution: keep implementations focused entirely on the `domain` layer/component.

## Acceptance Criteria
- [x] Code compiles without errors.
- [x] Unit tests written and passing (if applicable).
- [x] 100% alignment with the `specs/tdd.md` document for `sm-auth`.
