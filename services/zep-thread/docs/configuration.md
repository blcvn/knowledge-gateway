---
id: DOC-S05
service: zep-thread
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-thread — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9062` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `12062` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS JetStream URL |
| `NATS_STREAM` | string | `zep` | No | NATS stream name |
| `PG_DSN` | string | — | Yes | PostgreSQL connection string |
| `PG_MAX_OPEN_CONNS` | int | `10` | No | Max open connections |
| `ADVISORY_LOCK_RETRY_INITIAL` | duration | `200ms` | No | Initial lock retry interval |
| `ADVISORY_LOCK_RETRY_MAX` | duration | `30s` | No | Max lock retry interval |
| `ADVISORY_LOCK_MAX_ATTEMPTS` | int | `15` | No | Max lock retry attempts |
| `ADVISORY_LOCK_MULTIPLIER` | float | `2.0` | No | Retry backoff multiplier |

## YAML Configuration

```yaml
thread:
  grpc:
    port: 9062
  health:
    port: 12062
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 10
  advisory_lock:
    retry_initial: 200ms
    retry_max: 30s
    retry_max_attempts: 15
    retry_multiplier: 2.0
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  telemetry:
    service_name: "zep-thread"
    otel_endpoint: "otel-collector:4317"
```
