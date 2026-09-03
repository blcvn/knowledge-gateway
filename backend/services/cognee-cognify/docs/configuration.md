---
id: DOC-S05
service: cognee-cognify
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-cognify — Configuration Reference

## Environment Variables

### Server

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9012` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9092` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | No | Graceful shutdown timeout |

### PostgreSQL (Job State)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `PG_HOST` | string | `localhost` | Yes | PostgreSQL host |
| `PG_PORT` | int | `5432` | Yes | PostgreSQL port |
| `PG_DATABASE` | string | `cognee_cognify` | Yes | Database name |
| `PG_USER` | string | `cognee` | Yes | Database user |
| `PG_PASSWORD` | string | — | Yes | Database password |

### Neo4j (Knowledge Graph)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NEO4J_URI` | string | `bolt://neo4j:7687` | Yes | Neo4j Bolt URI |
| `NEO4J_USER` | string | `neo4j` | Yes | Neo4j username |
| `NEO4J_PASSWORD` | string | — | Yes | Neo4j password |
| `NEO4J_DATABASE` | string | `neo4j` | No | Neo4j database name |

### Qdrant (Vector Store)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `QDRANT_HOST` | string | `qdrant` | Yes | Qdrant host |
| `QDRANT_PORT` | int | `6334` | Yes | Qdrant gRPC port |
| `QDRANT_API_KEY` | string | — | No | Qdrant API key (if secured) |

### LLM (Bifrost Gateway)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `BIFROST_URL` | string | `http://bifrost:8443` | Yes | Bifrost LLM gateway URL |
| `LLM_MODEL` | string | `gpt-4o` | No | Default LLM model for extraction |
| `LLM_FAST_MODEL` | string | `gpt-4o-mini` | No | Fast model for classification/dedup |
| `EMBEDDING_MODEL` | string | `text-embedding-3-large` | No | Embedding model name |
| `LLM_CONCURRENCY` | int | `5` | No | Max concurrent LLM requests |

### NATS (Messaging)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |

### Pipeline

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DEFAULT_CHUNK_SIZE` | int | `512` | No | Default chunk size in tokens |
| `DEFAULT_CHUNK_OVERLAP` | int | `50` | No | Default chunk overlap in tokens |
| `PIPELINE_TIMEOUT` | duration | `30m` | No | Max pipeline execution time |

### Observability

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector endpoint |
| `OTEL_SERVICE_NAME` | string | `cognee-cognify` | No | Service name for traces |

## Example .env

```env
GRPC_PORT=9012
HEALTH_PORT=9092
LOG_LEVEL=info

PG_HOST=localhost
PG_PORT=5432
PG_DATABASE=cognee_cognify
PG_USER=cognee
PG_PASSWORD=secret

NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=neo4j_secret

QDRANT_HOST=qdrant
QDRANT_PORT=6334

BIFROST_URL=http://bifrost:8443
LLM_MODEL=gpt-4o
LLM_FAST_MODEL=gpt-4o-mini
EMBEDDING_MODEL=text-embedding-3-large
LLM_CONCURRENCY=5

NATS_URL=nats://nats:4222
OTEL_ENDPOINT=otel-collector:4317
```
