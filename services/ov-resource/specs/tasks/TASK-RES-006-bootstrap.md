---
id: TASK-RES-006
service: ov-resource
status: Done
---

# TASK-RES-006: Service Bootstrap, Configuration & Observability

## Objective
Configure application startup, dependency injection, observability, and shutdown protocols defined in `configuration.md` and `runbook.md`.

## Requirements
1. **Configuration (`internal/infra/config/config.go`)**:
   - Bind all env vars: `GRPC_PORT` (9054), `HEALTH_PORT` (9107), `LOG_LEVEL`, `OTEL_ENDPOINT`, `NATS_URL`, `DB_DSN`, `DB_MAX_CONNECTIONS`, `OV_FS_ADDR`, `CHUNK_SIZE_TOKENS`, `CHUNK_OVERLAP_TOKENS`, `MAX_INGESTION_SIZE_MB`, `WATCH_ENABLED`, `WATCH_DEFAULT_POLL_MS`, `WATCH_MAX_TASKS`, `TREESITTER_ENABLED`.
2. **Dependency Injection (`internal/infra/wire/wire.go`)**:
   - Connect layers safely using Google Wire.
3. **Observability (`runbook.md`)**:
   - **Metrics**: Expose Prometheus at `:9107/metrics`. Must include: `ov_resource_ingest_total`, `ov_resource_ingest_duration_seconds`, `ov_resource_chunks_produced_total`, `ov_resource_watch_polls_total`, `ov_resource_parse_errors_total`.
   - **Tracing**: OTel spans to `OTEL_ENDPOINT` for the ingestion pipeline (`ov-resource.Ingest`).
   - **Logging**: `slog` JSON formatting including `request_id`, `account_id`, `filename`.
4. **Application Lifecycle (`cmd/server/main.go`)**:
   - HTTP Health endpoints at `:9107` (`/healthz` returning `{"status": "serving"}`, `/readyz` checking db, nats, ov-fs).
   - Graceful shutdown on SIGTERM with a 30-second timeout, allowing active ingest jobs to complete and stopping watch polling.

## Dependencies
- Wire, Viper, Prometheus SDK, OpenTelemetry, slog.
