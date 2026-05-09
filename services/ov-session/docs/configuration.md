---
id: DOC-S05
service: ov-session
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-session — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9053` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9106` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_MAX_CONNECTIONS` | int | `20` | No | Connection pool max size |
| `OV_FS_ADDR` | string | `ov-fs:9051` | Yes | ov-fs gRPC address (archive/memory write) |
| `BIFROST_ADDR` | string | `bifrost:8443` | Yes | Bifrost LLM gateway |
| `COMPRESSION_VERSION` | string | `v2` | No | Default compressor: v1 (legacy) or v2 (template) |
| `MEMORY_EXTRACTION_ENABLED` | bool | `true` | No | Enable LLM memory extraction on commit |
| `MEMORY_EXTRACTION_MODE` | string | `async` | No | Extraction mode: async / sync |
| `DEDUP_SIMILARITY_THRESHOLD` | float | `0.85` | No | Semantic similarity threshold for dedup |
| `MAX_MESSAGES_PER_SESSION` | int | `1000` | No | Maximum messages per session |
| `WM_AUTO_UPDATE` | bool | `true` | No | Auto-update Working Memory after each message |
| `ARCHIVE_PATH_TEMPLATE` | string | `{account}/{user}/sessions/{id}/archive.md` | No | Archive path template |
| `MEMORY_PATH_TEMPLATE` | string | `{account}/{user}/memories/{category}/{id}.md` | No | Memory file path template |

## Example .env

```env
GRPC_PORT=9053
HEALTH_PORT=9106
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://vnp:secret@localhost:5432/ov_session?sslmode=disable
OV_FS_ADDR=ov-fs:9051
BIFROST_ADDR=bifrost:8443
COMPRESSION_VERSION=v2
MEMORY_EXTRACTION_ENABLED=true
MEMORY_EXTRACTION_MODE=async
DEDUP_SIMILARITY_THRESHOLD=0.85
```
