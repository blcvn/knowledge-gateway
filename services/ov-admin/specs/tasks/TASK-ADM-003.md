# Task: Usecase Layer - Identity, Auth Modes & API Keys (TASK-ADM-003)

**Status:** DONE

## Description
Implement the core authentication resolution, RBAC enforcement, and API Key lifecycle.

## Requirements
- Implement the Argon2id hashing adapter in `internal/adapter/hasher/argon2_hasher.go`.
- Implement `APIKeyUseCase` in `internal/usecase/api_key_ops.go`:
  - Create (generate 32-byte secret, Argon2id hash, store prefix and hash).
  - Validate (constant-time Argon2id hash comparison).
  - Revoke API key.
- Implement Auth Mode Resolution logic:
  - `dev` (skips validation, default ROOT).
  - `trusted` (trusts `X-OpenViking-*` headers, validates optional root key).
  - `api_key` (validates token against Argon2id hashes, extracts namespace and role).
- Implement hierarchical RBAC enforcement logic (ROOT > ADMIN > USER > AGENT).
