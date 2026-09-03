---
id: DOC-S05
service: cognee-search
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-search — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9013` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9093` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `NEO4J_URI` | string | `bolt://neo4j:7687` | Yes | Neo4j connection |
| `NEO4J_USER` | string | `neo4j` | Yes | Neo4j user |
| `NEO4J_PASSWORD` | string | — | Yes | Neo4j password |
| `QDRANT_HOST` | string | `qdrant` | Yes | Qdrant host |
| `QDRANT_PORT` | int | `6334` | Yes | Qdrant gRPC port |
| `REDIS_URL` | string | `redis://redis:6379/1` | Yes | Redis for result cache |
| `REDIS_CACHE_TTL` | duration | `5m` | No | Search result cache TTL |
| `BIFROST_URL` | string | `http://bifrost:8443` | Yes | LLM gateway |
| `LLM_MODEL` | string | `gpt-4o` | No | LLM model for completions |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector |
| `DEFAULT_TOP_K` | int | `10` | No | Default result count |
| `MAX_TOP_K` | int | `100` | No | Maximum allowed top_k |

## Example .env

```env
GRPC_PORT=9013
HEALTH_PORT=9093
NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=secret
QDRANT_HOST=qdrant
QDRANT_PORT=6334
REDIS_URL=redis://redis:6379/1
BIFROST_URL=http://bifrost:8443
NATS_URL=nats://nats:4222
```
