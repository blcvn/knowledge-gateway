---
id: DOC-S05
service: sm-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-memory — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9072` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9117` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_MAX_CONNS` | int | `25` | No | Max DB pool size |
| `BIFROST_URL` | string | `http://bifrost:8443` | Yes | LLM gateway |
| `EMBEDDING_MODEL` | string | `text-embedding-3-small` | No | Embedding model |
| `EXTRACTION_MODEL` | string | `gpt-4o-mini` | No | LLM for fact extraction |
| `FORGETTING_EVAL_INTERVAL` | duration | `1h` | No | Forgetting curve evaluation interval |
| `FORGETTING_DECAY_BASE_HOURS` | int | `24` | No | Base decay threshold (hours) |
| `MAX_VERSION_CHAIN_DEPTH` | int | `10` | No | Max version chain depth |

## Example .env

```env
GRPC_PORT=9072
HEALTH_PORT=9117
NATS_URL=nats://nats:4222
DB_DSN=postgres://user:pass@postgresql:5432/sm_memory?sslmode=disable
BIFROST_URL=http://bifrost:8443
EXTRACTION_MODEL=gpt-4o-mini
FORGETTING_EVAL_INTERVAL=1h
```
