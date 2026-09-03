---
id: TASK-FS-002
title: Implement ov-fs Usecase Layer
service: ov-fs
status: Done
---

# TASK-FS-002: Implement ov-fs Usecase Layer

## Objective
Implement the Usecase layer (Layer 2) for the `ov-fs` microservice to handle the core file system business logic and orchestrate domain entities.

## Requirements

1. **Usecase Operations**:
   - Implement `ReadFile`, `WriteFile`, and `DeleteFile` in `internal/usecase/file_ops.go`.
   - Implement `MkDir`, `ListDir`, and `Tree` in `internal/usecase/dir_ops.go`.
   - Implement `Grep` and `Glob` in `internal/usecase/search_ops.go`.
   - Implement `Move` (with mv lock logic) in `internal/usecase/move_ops.go`.
   - Implement `GetRelations` and `AddRelation` in `internal/usecase/relation_ops.go`.

2. **Ports and DTOs**:
   - Define input interfaces (`FileUseCase`, `DirUseCase`, `SearchUseCase`) in `internal/usecase/port/input.go`.
   - Define output interfaces (`EncryptionPort`, `EventPublisherPort`, `AbstractGeneratorPort`) in `internal/usecase/port/output.go`.
   - Implement Request and Response DTOs in `internal/usecase/dto/file_dto.go`.

3. **Integration Logic**:
   - Ensure `WriteFile` securely encrypts content through the `EncryptionPort` and emits `ContentWritten` via `EventPublisherPort`.
   - Ensure `DeleteFile` emits `ContentDeleted`.
   - Enforce tiered context abstraction levels (L0, L1, L2).

## Acceptance Criteria
- The Usecase layer only imports the Domain layer.
- Core logic fully satisfies the specification defined in `docs/api.md` and `specs/tdd.md`.
- File content encryption logic uses the defined `EncryptionPort` correctly.
