---
id: DOC-S06
service: memobase-engine
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-engine — Runbook

## Startup / Shutdown

```bash
make run-memobase-engine
# Or: docker compose up memobase-engine

# Graceful shutdown: SIGTERM → drain NATS consumers → finish in-flight LLM calls
kill -SIGTERM <pid>
```

### Startup Checks

1. LLM sanity check: sends test request to Bifrost (max_tokens=16)
2. Embedding dimension validation: verifies vector dim matches `EMBEDDING_DIM`
3. PostgreSQL pgvector extension: `CREATE EXTENSION IF NOT EXISTS vector`

## Health Check

| Endpoint | Port | Expected |
|----------|------|----------|
| gRPC health | 9032 | `SERVING` |
| HTTP /healthz | 9099 | `200 OK {"status":"healthy"}` |
| HTTP /readyz | 9099 | `200 OK` (DB + NATS + Bifrost connected) |

## Common Issues

### LLM call failures

**Symptom**: Buffer entries move to `failed` status; no profiles updated.

**Diagnosis**: Check logs for `LLM_INVOCATION_FAILED` errors. Verify Bifrost connectivity.

**Resolution**:
```bash
# Test Bifrost connectivity
curl http://bifrost:8443/health

# Retry failed buffers (they remain in memobase-ingestion as "failed")
# Reset via memobase-ingestion FlushBuffer RPC
```

### Profile merge conflicts

**Symptom**: Duplicate or inconsistent profile entries.

**Diagnosis**:
```sql
SELECT topic, sub_topic, count(*) FROM user_profiles
WHERE user_id = 'xxx' AND project_id = 'yyy'
GROUP BY topic, sub_topic HAVING count(*) > 1;
```

**Resolution**: YOLO merge should handle dedup. If persistent, check LLM model quality.

### High embedding latency

**Symptom**: Event processing slow (> 5s per flush).

**Diagnosis**: Check `memobase_engine_embedding_latency` Prometheus metric.

**Resolution**: Consider switching to faster embedding provider or reducing batch size.

## Deployment

```bash
kubectl rollout restart deployment/memobase-engine -n vnp-memory
kubectl rollout undo deployment/memobase-engine -n vnp-memory
```

## Monitoring

- **Key Metrics**: LLM invocations/s, LLM latency p95, profile updates/s, flush success rate
- **Alerts**: LLM failure rate > 10%, flush queue depth > 100, embedding latency > 10s

## Escalation

- **L1**: VNP Memory — Memobase team
- **L2**: Platform team (if Bifrost/NATS infrastructure issue)
