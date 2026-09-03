---
id: DOC-S05
service: sm-analytics
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-analytics — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9078` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9123` | Yes | Health check port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS for event consumption |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_MAX_CONNS` | int | `15` | No | Max DB pool |
| `AGGREGATION_INTERVAL` | duration | `1h` | No | Materialized view refresh interval |
| `EVENT_BATCH_SIZE` | int | `100` | No | NATS batch consumption size |
| `RETENTION_DAYS` | int | `90` | No | Raw event retention period |

## Example .env

```env
GRPC_PORT=9078
HEALTH_PORT=9123
NATS_URL=nats://nats:4222
DB_DSN=postgres://user:pass@postgresql:5432/sm_analytics?sslmode=disable
AGGREGATION_INTERVAL=1h
RETENTION_DAYS=90
```
