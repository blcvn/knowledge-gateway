---
id: TASK-CRY-004
title: gRPC Handler Implementation for ov-crypto
status: Done
created: 2026-05-11
---

# Task: gRPC Handler Implementation for ov-crypto

## Objective
Implement the gRPC adapter to expose the crypto functionalities over the network, mapping protobuf contracts to usecase inputs/outputs.

## Scope
- `api/proto/openviking/crypto/v1/service.proto` (if not already present or needs generation)
- `internal/adapter/grpc/handler.go`

## Requirements

### 1. gRPC Service Definition
Ensure the `OvCryptoService` is fully defined with the following RPCs:
- `Encrypt(EncryptRequest) returns (EncryptResponse)`
- `Decrypt(DecryptRequest) returns (DecryptResponse)`
- `RotateKey(RotateKeyRequest) returns (RotateKeyResponse)`
- `GetKeyStatus(KeyStatusRequest) returns (KeyStatus)`

Generate the Go stubs from the protobuf file.

### 2. Handler Implementation (`handler.go`)
Implement the gRPC server handler for `OvCryptoService`:
- Inject the `CryptoUseCase` interface.
- Implement the `Encrypt` RPC: map request to DTO, call usecase, map response.
- Implement the `Decrypt` RPC: map request to DTO, call usecase, map response.
- Implement the `RotateKey` RPC: handle account_id, reason, call usecase, map response.
- Implement the `GetKeyStatus` RPC: check account_id, call usecase, map response.

### 3. Error Mapping
Map domain errors to specific gRPC error codes:
- `Invalid account_id / empty plaintext` -> `INVALID_ARGUMENT` (400)
- `Account key not found` -> `NOT_FOUND` (404)
- `KMS provider error` -> `INTERNAL` (500)
- `AuthenticationFailedError / KeyMismatch` -> `UNAUTHENTICATED` (401)
- `InvalidMagicError / CorruptedCiphertextError` -> `DATA_LOSS` (500)

## Dependencies
- TASK-CRY-002 (Usecase Layer)

## Acceptance Criteria
- gRPC handler correctly translates requests and responses.
- Accurate mapping of domain-specific error types to standard gRPC status codes.
- No direct implementation of business logic within the handler.
