---
id: TASK-FS-001
title: Implement ov-fs Domain Layer
service: ov-fs
status: Done
---

# TASK-FS-001: Implement ov-fs Domain Layer

## Objective
Implement the Domain layer (Layer 1) for the `ov-fs` microservice in accordance with the 4-layer Clean Architecture, adhering strictly to the zero-dependency rule.

## Requirements

1. **Domain Models**:
   - Implement `File`, `FileMetadata`, `DirEntry`, and `FileContent` in `internal/domain/model/file.go`.
   - Implement `TreeNode` and `TreeOptions` in `internal/domain/model/tree.go`.
   - Implement `FileRelation` and `RelationType` in `internal/domain/model/relation.go`.
   - Implement `ContextLevel` enum (L0, L1, L2) in `internal/domain/model/context_level.go`.
   - Implement `LockType` (Point, Subtree, Move) and `LockRequest` in `internal/domain/model/lock.go`.

2. **Repository Interfaces**:
   - Define `FileRepository` interface with CRUD, tree, grep, and glob operations in `internal/domain/repository/file_repo.go`.
   - Define `RelationRepository` interface in `internal/domain/repository/relation_repo.go`.
   - Define `AbstractRepository` interface for tiered context abstraction in `internal/domain/repository/abstract_repo.go`.

3. **Domain Events & Errors**:
   - Define `ContentWritten` and `ContentDeleted` events in `internal/domain/event.go`.
   - Define domain-specific errors (`PathNotFound`, `PathExists`, `LockContention`) in `internal/domain/errors.go`.

## Acceptance Criteria
- All domain code resides in `internal/domain/`.
- The Domain layer contains zero external dependencies (pure Go structs and interfaces only).
- All models, events, and interfaces are defined as specified in `docs/architecture.md` and `specs/tdd.md`.
