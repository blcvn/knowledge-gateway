---
id: TASK-CRY-005
title: Infrastructure Layer and Bootstrap for ov-crypto
status: Done
created: 2026-05-11
---

# Task: Infrastructure Layer and Bootstrap for ov-crypto

## Objective
Implement the Infrastructure layer (Layer 4) for data persistence, dependency injection, and overall service bootstrap.

## Scope
- `internal/infra/persistence/`
- `internal/infra/config/`
- `internal/infra/wire/`
- `cmd/server/main.go`

## Requirements

### 1. PostgreSQL Persistence (`key_repo.go`)
Implement the `KeyRepository` interface to interact with PostgreSQL.
- Handle interactions with `ov_account_keys`, `ov_key_rotation_log`, and `ov_api_key_hashes` tables.
- Implement necessary SQL migrations matching the schema defined in `data-model.md`.
- Support transactional inserts/updates where necessary (e.g., during rotation).

### 2. Configuration (`config.go`)
- Implement configuration loading mechanism (e.g., using Viper).
- Load environmental variables: `GRPC_PORT` (9055), `HEALTH_PORT` (9108), `LOG_LEVEL`, `OTEL_ENDPOINT`, `NATS_URL`, `DB_DSN`, `DB_MAX_CONNECTIONS`.
- Load KMS specific variables: `KMS_PROVIDER` (local/vault/aws_kms/gcp_kms), `KMS_LOCAL_KEY_FILE`, `KMS_VAULT_ADDR`, `KMS_VAULT_TOKEN`, `KMS_VAULT_KEY_PATH`, `KMS_AWS_KEY_ID`, `KMS_AWS_REGION`, `KMS_GCP_KEY_NAME`.
- Load Argon2id configuration variables: `ARGON2_TIME` (3), `ARGON2_MEMORY_KB` (65536), `ARGON2_THREADS` (4).
- Load rotation variables: `KEY_ROTATION_MAX_BATCH` (1000).

### 3. Dependency Injection (`wire.go`)
- Create Wire provider sets for injecting Domain, Usecase, Adapter, and Infrastructure components.
- Dynamically construct the specific KMS Provider based on the `KMS_PROVIDER` configuration variable.

### 4. Service Bootstrap (`main.go`)
- Implement the application entry point.
- Bootstrap configuration, dependency injection, and structured logger (`slog`) with `request_id`, `account_id`, `operation`.
- Set up OTel tracing (Jaeger) with spans for `ov-crypto.Encrypt`, `ov-crypto.Decrypt`, `ov-crypto.RotateKey` and KMS operations.
- Set up Prometheus metrics at `:9108/metrics`: `ov_crypto_encrypt_total`, `ov_crypto_decrypt_total`, `ov_crypto_encrypt_duration_seconds`, `ov_crypto_key_rotation_total`, `ov_crypto_kms_latency_seconds`, `ov_crypto_api_key_validation_total`.
- Configure and start the gRPC server on `GRPC_PORT`.
- Expose Health Check API on `HEALTH_PORT`: `/healthz` and `/readyz` (with `db`, `kms`, `nats` checks).
- Implement graceful shutdown via SIGTERM (30s timeout) to ensure active requests complete and jobs are paused.

## Dependencies
- TASK-CRY-003 (KMS Providers and Event Adapters)
- TASK-CRY-004 (gRPC Handler)

## Acceptance Criteria
- Database schema matches the specifications accurately.
- Application bootstraps successfully with Wire.
- Observability and graceful shutdown are fully functional.
- The service can successfully run and accept gRPC connections.
