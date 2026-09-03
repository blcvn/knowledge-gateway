---
id: TASK-FS-007
title: Implement Configuration and Runbook Compliance
service: ov-fs
status: Done
---

# TASK-FS-007: Implement Configuration and Runbook Compliance

## Objective
Ensure 100% compliance with the specific operational and configuration requirements defined in `docs/configuration.md` and `docs/runbook.md`, enabling the service to be production-ready and fully observable by SRE/DevOps teams.

## Requirements

1. **Configuration Management**:
   - Extend Viper configuration (`internal/infra/config/config.go`) to explicitly map all required environment variables: `GRPC_PORT`, `HEALTH_PORT`, `LOG_LEVEL`, `OTEL_ENDPOINT`, `NATS_URL`, `DB_DSN`, `DB_BACKEND`, `DB_MAX_CONNECTIONS`, `CRYPTO_SERVICE_ADDR`, `CRYPTO_ENABLED`, `PATHLOCK_TIMEOUT_MS`, `MAX_FILE_SIZE_MB`, `ABSTRACT_GENERATION`, and `LLM_ENDPOINT`.
   - Ensure support for both PostgreSQL (`PG_*`) and SurrealDB (`SURREAL_*`) specific configuration parameters.

2. **Graceful Shutdown**:
   - Implement SIGTERM listener with a 30-second timeout.
   - Ensure pending writes complete and all `PathLock` mechanisms release held locks before the service terminates.

3. **Advanced Health Checks**:
   - Ensure `/healthz` HTTP endpoint on port `9104` returns `{"status": "serving"}`.
   - Implement `/readyz` endpoint checking and returning the status of external dependencies: `{"status": "ready", "checks": {"db": "ok", "nats": "ok", "crypto": "ok"}}`.

4. **Specific Observability Metrics**:
   - Expose the exact Prometheus metrics required by the Runbook at `:9104/metrics`:
     - `ov_fs_read_total` and `ov_fs_write_total` (Counters).
     - `ov_fs_read_duration_seconds` (Histogram).
     - `ov_fs_pathlock_wait_seconds` (Histogram).
     - `ov_fs_encryption_duration_seconds` (Histogram).
   - Ensure OTel traces integrate with Jaeger as specified.

## Acceptance Criteria
- All environment variables defined in `docs/configuration.md` are actively parsed and utilized.
- Sending `SIGTERM` gracefully shuts down the gRPC server, flushing writes and locks within 30 seconds.
- Curl to `/readyz` returns the correct JSON format and validates dependency health.
- Prometheus exposes the exact metric names specified in the runbook.
