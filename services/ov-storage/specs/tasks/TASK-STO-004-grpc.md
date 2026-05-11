---
id: TASK-STO-004
title: Implement gRPC Handlers for ov-storage
service: ov-storage
status: Done
---

# TASK-STO-004: Implement gRPC Handlers

## Objective
Implement the gRPC transport layer (Adapters) to expose the 3 consolidated services (`OvFsService`, `OvCryptoService`, `OvResourceService`).

## Requirements
1. **OvFsService Handler**:
   - Implement `ReadFile`, `WriteFile`, `DeleteFile`, `MkDir`, `ListDir`, `Tree`, `Grep`, `Glob`, `Move`, `GetRelations`.
2. **OvCryptoService Handler**:
   - Implement `Encrypt`, `Decrypt`, `RotateKey`, `GetKeyStatus`.
3. **OvResourceService Handler**:
   - Implement `Ingest`, `Parse`, `Watch`, `Refresh`.
4. **Integration**:
   - Map gRPC requests to corresponding Usecase methods.
   - Extract and validate `x-tenant-id` from gRPC metadata for tenant isolation.

## Acceptance Criteria
- [x] All 18 RPC endpoints are correctly mapped to usecases.
- [x] Tenant context is accurately parsed and propagated.
