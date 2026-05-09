---
id: DOC-S06
service: cognee-cognify
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# cognee-cognify — Runbook

## Startup

```bash
make run-cognee-cognify
docker compose up cognee-cognify
```

**Prerequisites**: PostgreSQL, Neo4j, Qdrant, NATS, Bifrost must be running.

## Shutdown

Graceful shutdown via SIGTERM (30s timeout).

## Health Check

```bash
curl http://localhost:9092/healthz
curl http://localhost:9092/readyz
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| Pipeline stuck at extract_entities | Bifrost LLM timeout | Check Bifrost health, verify API keys |
| Neo4j connection refused | Neo4j not started | Verify NEO4J_URI, check container logs |
| Job status FAILED | Pipeline error | Check error_message in job record |
| High memory usage | Large batch extraction | Reduce LLM_CONCURRENCY |

## Monitoring

- **Metrics**: Prometheus at `:9092/metrics`
- **Traces**: Jaeger via OTel collector
- **Logs**: Structured JSON via slog

## Escalation

1. On-call checks logs + job status
2. LLM issues → AI platform team
3. Graph DB issues → data infrastructure team
