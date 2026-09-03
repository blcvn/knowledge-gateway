---
id: TASK-ING-08
title: Implement Config and Telemetry Infrastructure
service: cognee-ingestion
feature: FEAT-ING-003
status: Done
---

## Objective
Implement configuration loading and OpenTelemetry/logging initialization.

## Files to Create/Update
- `internal/infra/config/config.go`: Viper config structs and loader logic.
- `internal/infra/telemetry/tracer.go`: OTel tracer provider.
- `internal/infra/telemetry/metrics.go`: Prometheus metrics setup.
- `internal/infra/telemetry/logger.go`: `slog` JSON logger setup.
- Related `*_test.go` files.

## Acceptance Criteria
- Config is parsed and validated correctly from env vars or YAML.
- Tracer, metrics, and logger initialize correctly adhering to OTel standards.
- Unit tests pass with >= 80% coverage.
