---
id: TASK-FS-003
title: Implement ov-fs Adapter Layer
service: ov-fs
status: Done
---

# TASK-FS-003: Implement ov-fs Adapter Layer

## Objective
Implement the Adapter layer (Layer 3) to handle gRPC communication, event messaging, and external service clients for `ov-fs`.

## Requirements

1. **gRPC Interface**:
   - Implement `OvFsService` gRPC handlers in `internal/adapter/grpc/handler.go` corresponding to the 10 RPCs specified in `docs/api.md`.
   - Implement Proto-to-Domain and Domain-to-Proto mappers in `internal/adapter/grpc/mapper.go`.

2. **Event Pub/Sub (NATS)**:
   - Implement NATS publisher for `ov.content.written` and `ov.content.deleted` events in `internal/adapter/event/publisher.go`.
   - Implement NATS subscriber for `ov.crypto.key.rotated` and `ov.session.memory.extracted` events in `internal/adapter/event/subscriber.go`.

3. **External Clients**:
   - Implement `ov-crypto` gRPC client to fulfill the `EncryptionPort` interface for envelope encryption/decryption in `internal/adapter/client/crypto_client.go`.

## Acceptance Criteria
- Full coverage of the `OvFsService` gRPC definition from `docs/api.md`.
- NATS events publish specific payloads (`path, account_id, size, checksum`) correctly.
- Mappers correctly map multi-tenant information (like `x-tenant-id` metadata) down to usecase DTOs.
