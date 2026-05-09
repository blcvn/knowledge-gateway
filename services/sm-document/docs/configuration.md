---
id: DOC-S05
service: sm-document
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-document — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9071` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9116` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_MAX_CONNS` | int | `25` | No | Max DB connection pool size |
| `DB_MIN_CONNS` | int | `5` | No | Min DB connection pool size |
| `BIFROST_URL` | string | `http://bifrost:8443` | Yes | LLM gateway for embeddings |
| `EMBEDDING_MODEL` | string | `text-embedding-3-small` | No | Default embedding model |
| `EMBEDDING_DIM` | int | `1536` | No | Embedding vector dimension |
| `MATRYOSHKA_DIM` | int | `256` | No | Compact embedding dimension |
| `MAX_DOCUMENT_SIZE_MB` | int | `50` | No | Max upload size |
| `CHUNK_SIZE` | int | `512` | No | Target chunk size in tokens |
| `CHUNK_OVERLAP` | int | `50` | No | Chunk overlap in tokens |
| `PIPELINE_WORKERS` | int | `4` | No | Concurrent pipeline workers |
| `PIPELINE_TIMEOUT_SEC` | int | `300` | No | Per-document processing timeout |

## Example .env

```env
GRPC_PORT=9071
HEALTH_PORT=9116
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://user:pass@postgresql:5432/sm_document?sslmode=disable
BIFROST_URL=http://bifrost:8443
EMBEDDING_MODEL=text-embedding-3-small
CHUNK_SIZE=512
CHUNK_OVERLAP=50
PIPELINE_WORKERS=4
```
