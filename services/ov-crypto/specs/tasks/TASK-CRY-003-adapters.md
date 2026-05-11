---
id: TASK-CRY-003
title: KMS Providers and Event Adapters for ov-crypto
status: Done
created: 2026-05-11
---

# Task: KMS Providers and Event Adapters for ov-crypto

## Objective
Implement the Adapter layer (Layer 3) components responsible for interfacing with Key Management Systems (KMS) and NATS event publishing.

## Scope
- `internal/adapter/kms/`
- `internal/adapter/event/`

## Requirements

### 1. Local KMS Provider (`local_provider.go`)
- Implement a local file-based KMS provider that satisfies the `KMSProviderPort` / `KMSProvider` interface.
- Intended for dev/test environments.
- Support `EncryptFileKey`, `DecryptFileKey`, and `RotateRootKey`.

### 2. Vault KMS Provider (`vault_provider.go`)
- Implement a KMS provider integrating with HashiCorp Vault Transit engine.
- Support `EncryptFileKey`, `DecryptFileKey`, and `RotateRootKey` using Vault APIs.

### 3. Cloud KMS Provider (`cloud_provider.go`)
- Implement a KMS provider structure capable of integrating with AWS KMS / GCP KMS.
- Support standard encryption/decryption of the File Key and rotation management using standard SDKs.

### 4. NATS Event Publisher (`publisher.go`)
- Implement an event publisher that satisfies the `EventPublisherPort`.
- Publish `ov.crypto.key.rotated` messages to NATS when key rotation completes successfully.
- Ensure correct serialization of the `KeyRotated` domain event.

## Dependencies
- TASK-CRY-001 (Domain Layer)
- TASK-CRY-002 (Usecase Layer)

## Acceptance Criteria
- All KMS providers successfully abstract the external encryption APIs.
- NATS publisher correctly handles message dispatch.
- Adapter components strictly adhere to the defined port interfaces.
