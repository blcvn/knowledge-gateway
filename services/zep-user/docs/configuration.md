---
id: DOC-S05
service: zep-user
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-user — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9061` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `12061` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS JetStream server URL |
| `NATS_STREAM` | string | `zep` | No | NATS stream name |
| `PG_DSN` | string | — | Yes | PostgreSQL connection string |
| `PG_MAX_OPEN_CONNS` | int | `10` | No | Max open PostgreSQL connections |
| `PG_MAX_IDLE_CONNS` | int | `5` | No | Max idle PostgreSQL connections |
| `PG_CONN_MAX_LIFETIME` | duration | `30m` | No | Connection max lifetime |
| `SERVICE_NAME` | string | `zep-user` | No | OTel service name |

## YAML Configuration

```yaml
user:
  grpc:
    port: 9061
  health:
    port: 12061
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 10
    max_idle_connections: 5
    conn_max_lifetime: 30m
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  telemetry:
    service_name: "zep-user"
    otel_endpoint: "otel-collector:4317"
```

## Example .env

```env
GRPC_PORT=9061
HEALTH_PORT=12061
LOG_LEVEL=info
PG_DSN=postgres://postgres:postgres@db:5432/zep?sslmode=disable
PG_MAX_OPEN_CONNS=10
NATS_URL=nats://nats:4222
NATS_STREAM=zep
OTEL_ENDPOINT=otel-collector:4317
SERVICE_NAME=zep-user
```
