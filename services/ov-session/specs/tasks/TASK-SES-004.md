---
id: TASK-SES-004
title: Implement Working Memory v2 Usecase
status: Done
---

# Task: Implement Working Memory v2 Usecase

## Objective
Implement the Working Memory (WM) v2 lifecycle handling, managing the evolving JSONB state machine during a session.

## Requirements
1. **gRPC Protocol Integration**:
   - Ensure `GetWorkingMemory` and `UpdateWorkingMemory` rpc methods are fully mapped in the protobuf.
2. **Working Memory Usecase** (`internal/usecase/working_memory.go`):
   - Implement `GetWorkingMemory`: Retrieve the current JSONB state from `ov_working_memory` for a given `session_id`.
   - Implement `UpdateWorkingMemory`: Handle updates to the state fields (goals, facts, errors, context, title, state).
   - *Note*: As per architecture, WM updates are evaluated and patched iteratively. Ensure the persistence layer effectively updates the JSONB fields.
3. **gRPC Handler Integration** (`internal/adapter/grpc/handler.go`):
   - Bind the Working Memory usecases to their respective gRPC endpoints.

## Acceptance Criteria
- [x] WM state correctly parses and persists JSONB payloads containing structured facts, goals, and errors.
- [x] gRPC endpoints accurately read and write WM state.
- [x] Updates properly adjust the `updated_at` timestamp.
