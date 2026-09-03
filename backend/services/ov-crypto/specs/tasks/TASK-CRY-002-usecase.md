---
id: TASK-CRY-002
title: Usecase Layer Implementation for ov-crypto
status: Done
created: 2026-05-11
---

# Task: Usecase Layer Implementation for ov-crypto

## Objective
Implement the Usecase layer (Layer 2) for the `ov-crypto` microservice to orchestrate envelope encryption, decryption, key rotation, and status querying.

## Scope
- `internal/usecase/port/`
- `internal/usecase/dto/`
- `internal/usecase/encrypt.go`
- `internal/usecase/decrypt.go`
- `internal/usecase/key_rotation.go`
- `internal/usecase/key_status.go`

## Requirements

### 1. Ports Definition
Define ports in `internal/usecase/port/`:
- `input.go`: Define `CryptoUseCase` interface with methods for Encrypt, Decrypt, RotateKey, GetKeyStatus.
- `output.go`: Define `KMSProviderPort`, `EventPublisherPort`, `KeyRepoPort` (mapping to domain interfaces).

### 2. DTOs
Define necessary Data Transfer Objects in `internal/usecase/dto/` for usecase input/output.

### 3. Encrypt Pipeline (`encrypt.go`)
Implement `Encrypt` logic:
1. Generate random 32-byte File Key.
2. Generate random 12-byte Data IV.
3. AES-256-GCM encrypt plaintext with (File Key, Data IV).
4. Encrypt File Key with Account Key (via KMS provider).
5. Build OVE1 envelope: Magic + Version + Header + EFK + KIV + DIV + Ciphertext.

### 4. Decrypt Pipeline (`decrypt.go`)
Implement `Decrypt` logic:
1. Check OVE1 magic (if missing, return as-is for backward-compatibility).
2. Parse envelope header.
3. Decrypt File Key via KMS provider.
4. AES-256-GCM decrypt content using the unwrapped File Key.

### 5. Key Rotation (`key_rotation.go`)
Implement `RotateKey` logic:
1. Orchestrate rotation of root or account key via KMSProvider.
2. Update key version in the repository.
3. Append entry to the rotation audit log.
4. Publish `KeyRotated` event (`ov.crypto.key.rotated`) via `EventPublisherPort`.

### 6. Key Status (`key_status.go`)
Implement `GetKeyStatus` logic to retrieve key version and status from the repository.

## Dependencies
- TASK-CRY-001 (Domain Layer)

## Acceptance Criteria
- All usecases implemented according to the cryptographic pipelines defined in `architecture.md`.
- No direct dependencies on infrastructure or external libraries for business logic (except standard crypto libraries).
