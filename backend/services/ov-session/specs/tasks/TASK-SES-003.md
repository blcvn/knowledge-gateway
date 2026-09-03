---
id: TASK-SES-003
title: Implement Session Lifecycle Usecase and gRPC API
status: Done
---

# Task: Implement Session Lifecycle Usecase and gRPC API

## Objective
Develop the Usecase Layer for core session management and expose it via the gRPC Adapter, enabling session creation and message appending.

## Requirements
1. **gRPC Protocol Definition**:
   - Ensure `service.proto` includes `OvSessionService` definition with `CreateSession`, `AddMessage`, and `GetMessages` rpc methods.
2. **Session Lifecycle Usecase** (`internal/usecase/session_lifecycle.go`):
   - Implement logic to handle session creation (`CreateSession`), mapping `account_id`, `user_id`, `agent_id` (default 'default').
   - Implement logic to append a new message (`AddMessage`), maintaining sequence order and updating `token_count`.
   - Implement logic to retrieve session history (`GetMessages`).
3. **gRPC Handler Implementation** (`internal/adapter/grpc/handler.go`):
   - Implement the `OvSessionService` gRPC server.
   - Map gRPC requests to Usecase DTOs and handle domain errors, translating them to appropriate gRPC error codes (e.g., `NOT_FOUND`, `INVALID_ARGUMENT`).

## Acceptance Criteria
- [x] `session_lifecycle.go` properly delegates to the repository layer.
- [x] gRPC handlers correctly expose the usecases and map input/output boundaries.
- [x] Multi-tenancy is enforced by requiring `account_id` validation.
