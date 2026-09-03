---
id: DOC-S05
service: graphiti-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-ingestion — Configuration Reference

## Environment Variables

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `GRPC_PORT` | gRPC server listen port | int | `9021` | ✅ |
| `HEALTH_PORT` | Health/metrics HTTP port | int | `9094` | ✅ |
| `POSTGRES_DSN` | PostgreSQL connection string | string | — | ✅ |
| `NATS_URL` | NATS JetStream server URL | string | `nats://localhost:4222` | ✅ |
| `GRAPHITI_KNOWLEDGE_ADDR` | graphiti-knowledge gRPC address | string | `graphiti-knowledge:9023` | ✅ |
| `GRAPHITI_STORE_ADDR` | graphiti-store gRPC address | string | `graphiti-store:9024` | ✅ |
| `LOG_LEVEL` | Logging level | string | `info` | ❌ |
| `LOG_FORMAT` | Log output format | string | `json` | ❌ |
| `OTEL_EXPORTER_ENDPOINT` | OTel collector endpoint | string | `localhost:4317` | ❌ |
| `OTEL_SERVICE_NAME` | OTel service name | string | `graphiti-ingestion` | ❌ |
| `MAX_CONCURRENT_INGESTIONS` | Max parallel saga pipelines | int | `10` | ❌ |
| `SAGA_RETRY_MAX` | Max retries per saga step | int | `3` | ❌ |
| `SAGA_RETRY_BACKOFF_MS` | Initial retry backoff in ms | int | `1000` | ❌ |
| `GROUP_SERIALIZATION_TIMEOUT` | Per-group lock timeout | duration | `60s` | ❌ |
| `GRPC_CLIENT_TIMEOUT` | Timeout for downstream gRPC calls | duration | `120s` | ❌ |
| `CIRCUIT_BREAKER_THRESHOLD` | Consecutive failures to open CB | int | `5` | ❌ |
| `CIRCUIT_BREAKER_TIMEOUT` | CB half-open timeout | duration | `30s` | ❌ |

## Example `.env`

```env
GRPC_PORT=9021
HEALTH_PORT=9094
POSTGRES_DSN=postgres://vnp:password@localhost:5432/vnp_memory?sslmode=disable
NATS_URL=nats://localhost:4222
GRAPHITI_KNOWLEDGE_ADDR=localhost:9023
GRAPHITI_STORE_ADDR=localhost:9024
LOG_LEVEL=debug
OTEL_EXPORTER_ENDPOINT=localhost:4317
MAX_CONCURRENT_INGESTIONS=10
SAGA_RETRY_MAX=3
```
