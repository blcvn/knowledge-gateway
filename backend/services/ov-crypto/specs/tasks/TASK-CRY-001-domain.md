---
id: TASK-CRY-001
title: Domain Layer Implementation for ov-crypto
status: Done
created: 2026-05-11
---

# Task: Domain Layer Implementation for ov-crypto

## Objective
Implement the Domain layer (Layer 1) for the `ov-crypto` microservice, strictly defining models, repositories interfaces, events, and domain errors without any external dependencies.

## Scope
- `internal/domain/model/`
- `internal/domain/repository/`
- `internal/domain/event.go`
- `internal/domain/errors.go`

## Requirements

### 1. Domain Models
Create the following structs and enums in `internal/domain/model/`:
- **`envelope.go`**:
  - `Envelope`: Struct representing the complete OVE1 envelope.
  - `EnvelopeHeader`: Struct for envelope header attributes.
  - `ProviderType`: Enum for KMS providers (local, vault, aws_kms, gcp_kms).
  - Define `OVE1` magic constant.
- **`key.go`**:
  - `KeyMaterial`: Struct for raw key data.
  - `KeyVersion`: Type/struct for version management.
  - `KeyStatus`: Enum (active, expired, revoked).
- **`kms.go`**:
  - `KMSProvider` interface (domain-level abstraction).

### 2. Repository Interfaces
Create `internal/domain/repository/key_repo.go`:
- Define `KeyRepository` interface for persisting key metadata (`ov_account_keys`) and rotation audit logs (`ov_key_rotation_log`).

### 3. Domain Events
Create `internal/domain/event.go`:
- Define `KeyRotated` struct (`account_id`, `old_version`, `new_version`) for the `ov.crypto.key.rotated` event payload.

### 4. Domain Errors
Create `internal/domain/errors.go`:
- Define core domain errors:
  - `AuthenticationFailedError` (AES-GCM auth tag mismatch)
  - `CorruptedCiphertextError` (Envelope parse failure)
  - `InvalidMagicError` (Not an OVE1 envelope)
  - `KeyMismatchError` (File Key decryption failed)

## Dependencies
- None.

## Acceptance Criteria
- Code compiles.
- Zero dependencies on `internal/usecase`, `internal/adapter`, or `internal/infra`.
- Domain interfaces and models accurately reflect the `data-model.md` and `architecture.md` documents.
