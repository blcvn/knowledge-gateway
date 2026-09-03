---
id: TASK-AUT-004
title: gRPC Handlers & NATS Publisher
service: sm-auth
status: Done
priority: P0
created: 2026-05-11
---

# gRPC Handlers & NATS Publisher

## Objective
Implement SmAuthService gRPC endpoints. Publish to `sm.auth.api_key.used`.

## Requirements
- Strictly follow the Clean Architecture definitions from `specs/tdd.md` and `docs/architecture.md`.
- No new features or architectures are to be created; only execute the documented design.
- Token-efficient execution: keep implementations focused entirely on the `grpc` layer/component.

## Acceptance Criteria
- [x] Code compiles without errors.
- [x] Unit tests written and passing (if applicable).
- [x] 100% alignment with the `specs/tdd.md` document for `sm-auth`.
