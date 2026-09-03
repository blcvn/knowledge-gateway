---
id: DOC-S05
service: sm-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-search — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9073` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9118` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `BIFROST_URL` | string | `http://bifrost:8443` | Yes | Embedding + reranking |
| `EMBEDDING_MODEL` | string | `text-embedding-3-small` | No | Query embedding model |
| `RERANK_MODEL` | string | `rerank-english-v3.0` | No | Cross-encoder model |
| `HNSW_EF_SEARCH` | int | `64` | No | HNSW search accuracy |
| `RRF_K_CONSTANT` | int | `60` | No | RRF fusion constant |
| `DEFAULT_CHUNK_CONTEXT_WINDOW` | int | `1` | No | ±N adjacent chunks |
| `MAX_SEARCH_RESULTS` | int | `100` | No | Hard result limit |
| `QUERY_REWRITE_TIMEOUT_MS` | int | `500` | No | LLM rewrite timeout |
| `RERANK_TIMEOUT_MS` | int | `1000` | No | Cross-encoder timeout |

## Example .env

```env
GRPC_PORT=9073
HEALTH_PORT=9118
DB_DSN=postgres://user:pass@postgresql:5432/sm_search?sslmode=disable
BIFROST_URL=http://bifrost:8443
HNSW_EF_SEARCH=64
RRF_K_CONSTANT=60
```
