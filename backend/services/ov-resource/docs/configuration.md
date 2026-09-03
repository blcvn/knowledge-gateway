---
id: DOC-S05
service: ov-resource
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-resource — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9054` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9107` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_MAX_CONNECTIONS` | int | `20` | No | Connection pool max size |
| `OV_FS_ADDR` | string | `ov-fs:9051` | Yes | ov-fs gRPC address |
| `CHUNK_SIZE_TOKENS` | int | `512` | No | Default chunk size in tokens |
| `CHUNK_OVERLAP_TOKENS` | int | `50` | No | Chunk overlap in tokens |
| `MAX_INGESTION_SIZE_MB` | int | `100` | No | Maximum single file ingestion size |
| `WATCH_ENABLED` | bool | `true` | No | Enable watch manager |
| `WATCH_DEFAULT_POLL_MS` | int | `30000` | No | Default watch polling interval |
| `WATCH_MAX_TASKS` | int | `100` | No | Maximum concurrent watch tasks |
| `TREESITTER_ENABLED` | bool | `true` | No | Enable tree-sitter for code parsing |

## Example .env

```env
GRPC_PORT=9054
HEALTH_PORT=9107
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://vnp:secret@localhost:5432/ov_resource?sslmode=disable
OV_FS_ADDR=ov-fs:9051
CHUNK_SIZE_TOKENS=512
CHUNK_OVERLAP_TOKENS=50
WATCH_ENABLED=true
```
