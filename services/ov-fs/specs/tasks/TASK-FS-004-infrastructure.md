---
id: TASK-FS-004
title: Implement ov-fs Infrastructure Layer
service: ov-fs
status: Done
---

# TASK-FS-004: Implement ov-fs Infrastructure Layer

## Objective
Implement the Infrastructure layer (Layer 4) for `ov-fs`, including database persistence, locking mechanisms, and dependency injection setup.

## Requirements

1. **Persistence (Database)**:
   - Implement `vikingfs_repo.go` to support PostgreSQL/SurrealDB storage in `internal/infra/persistence/`.
   - Implement `relation_repo.go` mapping to the `ov_file_relations` table.
   - Implement `abstract_repo.go` to store L0/L1 abstracted content.
   - Strictly follow the schema specifications defined in `docs/data-model.md`.

2. **Concurrency & Locking**:
   - Implement the `PathLock` (Point, Subtree, Move) mechanism in `internal/infra/lock/pathlock.go`.
   - Integrate in-memory locking with distributed PostgreSQL advisory locks for multi-replica concurrency support.

3. **Bootstrap and Configuration**:
   - Implement configuration parsing using Viper in `internal/infra/config/config.go`.
   - Setup the gRPC server listener in `internal/infra/server/grpc.go`.
   - Setup Google Wire dependency injection in `internal/infra/wire/wire.go`.
   - Create `cmd/server/main.go` entry point to bind and start the service.

## Acceptance Criteria
- Database queries correctly filter by `account_id` for tenant isolation.
- Schema aligns with `docs/data-model.md` including indexes.
- PathLock operates correctly preventing lock contention on concurrent writes/moves.
- Service is capable of starting via `main.go`.
