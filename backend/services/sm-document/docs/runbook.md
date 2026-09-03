---
id: DOC-S06
service: sm-document
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
audience: SRE, DevOps, On-call engineers
---

# sm-document — Runbook

## Startup

```bash
make run-sm-document
# Or: docker compose up sm-document
```

**Prerequisites**: PostgreSQL+pgvector, NATS, Bifrost

## Shutdown

Graceful shutdown via SIGTERM (30s timeout). In-progress pipeline jobs marked `failed`.

## Health Check

```bash
curl http://localhost:9116/healthz
# Expected: {"status": "serving"}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| Documents stuck in `queued` | Pipeline workers idle | Check PIPELINE_WORKERS, restart |
| `extracting` failures | Bifrost unreachable | Verify BIFROST_URL |
| `RESOURCE_EXHAUSTED` | DB pool exhausted | Increase DB_MAX_CONNS |
| NATS publish failures | NATS down | Check NATS cluster health |

## Monitoring

- **Metrics**: `:9116/metrics` — create_total, pipeline_duration, chunks_total
- **Traces**: Jaeger via OTel
- **Logs**: JSON slog with request_id, org_id, document_id

## Escalation

1. On-call checks logs + metrics
2. Pipeline failures >10% in 5min → Supermemory team lead
3. Bifrost down >5min → Infrastructure team
