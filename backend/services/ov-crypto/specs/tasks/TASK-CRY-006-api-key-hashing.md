---
id: TASK-CRY-006
title: API Key Hashing for ov-crypto
status: Done
created: 2026-05-11
---

# Task: API Key Hashing for ov-crypto

## Objective
Implement API key hashing and validation logic using the Argon2id algorithm.

## Scope
- `internal/domain/model/api_key.go`
- `internal/usecase/api_key.go`
- `internal/usecase/port/`

## Requirements

### 1. Domain Models
Create `internal/domain/model/api_key.go`:
- Define structures for API key hashing based on `ov_api_key_hashes` schema.
- Map fields: `id`, `account_id`, `user_id`, `key_hash`, `key_prefix`, `role`, `status`.

### 2. Repository Interface
Update `internal/domain/repository/key_repo.go` (or add a new repo interface) to include:
- Methods to save and retrieve API key hashes by `key_prefix` and `account_id`.

### 3. Usecase Logic (`api_key.go`)
Implement the API key usecase:
- **Hash Generation**: Use Argon2id to hash raw API keys. The hashing parameters (time, memory_kb, threads) should be configurable via env vars (default: time=3, memory=64MB, threads=4).
- **Validation**: Compare a provided raw API key against the stored Argon2id hash. Handle the expected CPU-intensive nature of this validation (~100ms).

## Dependencies
- TASK-CRY-001 (Domain Layer)
- TASK-CRY-005 (Infrastructure Layer)

## Acceptance Criteria
- API keys are never stored in plaintext.
- Argon2id is correctly implemented with the required parameters.
- Validation correctly authenticates the keys.
