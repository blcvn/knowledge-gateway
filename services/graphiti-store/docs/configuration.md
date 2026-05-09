---
id: DOC-S05
service: graphiti-store
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-store — Configuration Reference

## Environment Variables

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `GRPC_PORT` | gRPC server listen port | int | `9024` | ✅ |
| `HEALTH_PORT` | Health/metrics HTTP port | int | `9097` | ✅ |
| `LOG_LEVEL` | Logging level | string | `info` | ❌ |
| `LOG_FORMAT` | Log output format | string | `json` | ❌ |
| `OTEL_EXPORTER_ENDPOINT` | OTel collector endpoint | string | `localhost:4317` | ❌ |
| `OTEL_SERVICE_NAME` | OTel service name | string | `graphiti-store` | ❌ |
| `NATS_URL` | NATS JetStream server URL | string | `nats://localhost:4222` | ✅ |

### Service-Specific Variables

See [Architecture](./architecture.md) for downstream service addresses and database connection strings.

## Example `.env`

```env
GRPC_PORT=9024
HEALTH_PORT=9097
NATS_URL=nats://localhost:4222
LOG_LEVEL=debug
OTEL_EXPORTER_ENDPOINT=localhost:4317
```
