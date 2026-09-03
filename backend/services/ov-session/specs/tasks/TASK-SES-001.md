---
id: TASK-SES-001
title: Define Domain Models and Clean Architecture Skeleton
status: Done
---

# Task: Define Domain Models and Clean Architecture Skeleton

## Objective
Implement the Domain Layer (Layer 1) for the `ov-session` service based on the established 4-layer Clean Architecture and Data Model.

## Requirements
1. **Initialize Project Structure**:
   - Scaffold directories: `cmd/server`, `internal/domain/model`, `internal/domain/repository`, `internal/usecase/port`, `internal/usecase/dto`, `internal/adapter`, and `internal/infra`.
2. **Implement Domain Models** (`internal/domain/model/`):
   - `session.go`: Define `Session`, `SessionMeta`, `SessionStatus`.
   - `message.go`: Define `Message`, `MessageRole`, `ToolCall`.
   - `working_memory.go`: Define `WorkingMemory` v2 (title, state, goals, facts, errors, context).
   - `memory.go`: Define `CandidateMemory`, `MemoryCategory` (fact, preference, skill, procedure, tool_skill).
   - `compression.go`: Define `SessionCompression`, `ExtractionStats`, `CompressionVersion` (v1/v2).
3. **Define Interfaces & Constants** (`internal/domain/`):
   - `repository/session_repo.go`: Define `SessionRepository` interface.
   - `repository/message_repo.go`: Define `MessageRepository` interface.
   - `event.go`: Define structs for `SessionCommitted` and `MemoryExtracted` NATS events.
   - `errors.go`: Define domain-specific errors (e.g., `SessionNotFound`, `AlreadyCommitted`).

## Acceptance Criteria
- [x] Directory structure perfectly matches `architecture.md`.
- [x] All domain models strictly align with the `data-model.md` definitions.
- [x] No external infrastructure dependencies leak into the Domain Layer.
