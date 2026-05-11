---
id: DOC-S06
service: cognee-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# cognee-pipeline — Runbook

## Startup / Shutdown

```bash
# Start
docker compose up cognee-pipeline

# Graceful shutdown (30s drain)
docker compose stop cognee-pipeline

# Force stop
docker compose kill cognee-pipeline
```

## Health Checks

| Endpoint | Port | Expected |
|----------|------|----------|
| `GET /healthz` | 9091 | `200 {"status":"ok","checks":{"postgres":"up","neo4j":"up","minio":"up","nats":"up"}}` |
| gRPC Health | 9011 | `SERVING` |

```bash
curl http://localhost:9091/healthz
grpcurl -plaintext localhost:9011 grpc.health.v1.Health/Check
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| Health check failing | Check dependency connections | Verify POSTGRES_DSN, NEO4J_URI, NATS_URL |
| Pipeline stuck RUNNING | Stage timeout or LLM failure | Check job metrics, cancel and retry |
| OOM during large file | Text extraction consuming memory | Increase container memory, reduce chunk size |
| Neo4j connection refused | Graph DB down | Restart Neo4j, check bolt port 7687 |
| NATS publish failure | JetStream stream missing | Create stream: `nats stream add cognee` |
| LLM rate limited | Too many concurrent calls | Reduce `PIPELINE_MAX_CONCURRENT_LLM` |

## Monitoring

- **Prometheus**: `:9091/metrics`
- **Key Metrics**: `pipeline_duration_seconds`, `pipeline_stage_duration_seconds`, `llm_calls_total`, `ingestion_items_total`
- **OTel Traces**: Pipeline stages as spans
- **Logs**: `docker compose logs cognee-pipeline -f`

## Deployment

```bash
# Build
docker build -t vnp-memory/cognee-pipeline:latest -f services/cognee-pipeline/Dockerfile .

# Deploy
docker compose up -d cognee-pipeline

# Rollback
docker compose up -d --no-deps cognee-pipeline  # with previous image tag
```

## Escalation

1. On-call engineer → check logs + health
2. If pipeline stuck → cancel job via gRPC
3. If data loss → restore from MinIO (files) + PostgreSQL backup (metadata)
4. Escalate to Cognee Engine Team lead
