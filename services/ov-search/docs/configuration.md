---
id: DOC-S05
service: ov-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-search — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9052` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9105` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string (hotness/metadata) |
| `QDRANT_URL` | string | `qdrant:6334` | Yes | Qdrant gRPC endpoint |
| `QDRANT_COLLECTION` | string | `ov_embeddings` | No | Qdrant collection name |
| `BIFROST_ADDR` | string | `bifrost:8443` | Yes | Bifrost LLM gateway for embeddings |
| `EMBEDDING_MODEL` | string | `text-embedding-3-large` | No | Embedding model name |
| `EMBEDDING_DIM` | int | `1536` | No | Embedding vector dimension |
| `OV_FS_ADDR` | string | `ov-fs:9051` | Yes | ov-fs gRPC address (tiered loading) |
| `HOTNESS_DECAY_HALF_LIFE_H` | int | `24` | No | Hotness decay half-life in hours |
| `HOTNESS_SESSION_BOOST` | float | `0.3` | No | Boost factor per session reference |
| `HOTNESS_RECOMPUTE_INTERVAL_M` | int | `5` | No | Hotness recompute interval in minutes |
| `RERANK_DEFAULT_STRATEGY` | string | `rrf` | No | Default reranking: rrf / mmr / cross_encoder |
| `SEARCH_MAX_RESULTS` | int | `50` | No | Maximum results per search |
| `PROPAGATION_FACTOR` | float | `0.7` | No | Score propagation factor (child → parent) |
| `CONVERGENCE_THRESHOLD` | float | `0.05` | No | Stop when quality gain < threshold |

## Example .env

```env
GRPC_PORT=9052
HEALTH_PORT=9105
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://vnp:secret@localhost:5432/ov_search?sslmode=disable
QDRANT_URL=qdrant:6334
BIFROST_ADDR=bifrost:8443
OV_FS_ADDR=ov-fs:9051
HOTNESS_DECAY_HALF_LIFE_H=24
RERANK_DEFAULT_STRATEGY=rrf
```
