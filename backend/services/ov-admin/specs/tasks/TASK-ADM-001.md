# Task: Domain Layer & Core Models (TASK-ADM-001)

**Status:** DONE

## Description
Initialize the 4-layer Clean Architecture for the `ov-admin` service and implement the Domain layer.

## Requirements
- Scaffold the project structure (`cmd/`, `internal/domain/`, `internal/usecase/`, `internal/adapter/`, `internal/infra/`).
- Implement domain models in `internal/domain/model/`:
  - `account.go`: `Account`, `AccountConfig`, `NamespacePolicy`.
  - `user.go`: `User`, `Role` enum (`ROOT`, `ADMIN`, `USER`, `AGENT`).
  - `agent.go`: `Agent`, `AgentConfig`.
  - `api_key.go`: `APIKey`, `KeyStatus`, `ValidateResult`.
  - `namespace.go`: `NamespaceURI`, `NamespacePolicy`.
- Implement repository interfaces in `internal/domain/repository/`:
  - `AccountRepository`, `UserRepository`, `APIKeyRepository`.
- Implement custom errors in `internal/domain/errors.go` (`AccountNotFound`, `DuplicateUser`, `InvalidKey`, `PermissionDenied`).
- Implement domain events in `internal/domain/event.go` (`AccountCreated`, `UserDeleted`).
- Ensure no external dependencies are imported in the Domain layer.
