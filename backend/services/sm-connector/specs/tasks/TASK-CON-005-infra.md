---
id: TASK-CON-005
title: Infrastructure & External Clients
service: sm-connector
status: Done
priority: P0
created: 2026-05-11
---

# Infrastructure & External Clients

## Objective
Setup Wire DI, config, Telemetry, and external HTTP clients for Google Drive, Notion, OneDrive.

## Requirements
- Strictly follow the Clean Architecture definitions from `specs/tdd.md` and `docs/architecture.md`.
- No new features or architectures are to be created; only execute the documented design.
- Token-efficient execution: keep implementations focused entirely on the `infra` layer/component.

## Acceptance Criteria
- [x] Code compiles without errors.
- [x] Unit tests written and passing (if applicable).
- [x] 100% alignment with the `specs/tdd.md` document for `sm-connector`.
