---
id: DOC-S06
service: memobase-context
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-context — Runbook

## Startup / Shutdown

```bash
make run-memobase-context
# Or: docker compose up memobase-context

# Graceful shutdown
kill -SIGTERM <pid>
```

## Health Check

| Endpoint | Port | Expected |
|----------|------|----------|
| gRPC health | 9033 | `SERVING` |
| HTTP /healthz | 9100 | `200 OK {"status":"healthy"}` |
| HTTP /readyz | 9100 | `200 OK` (DB + Redis + NATS connected) |

## Common Issues

### High context latency (> 100ms p95)

**Symptom**: GetContext exceeding SLA target.

**Diagnosis**:
1. Check Redis cache hit rate: `redis-cli info stats | grep keyspace_hits`
2. Check pgvector query latency: `memobase_context_db_query_latency` metric
3. Check event gist index: HNSW vs IVFFlat

**Resolution**:
- If low cache hit rate: verify NATS `profile.changed` events aren't causing excessive invalidation
- If high DB latency: check pgvector index, consider `REINDEX CONCURRENTLY`
- If event search slow: reduce `EVENT_SEARCH_WINDOW_DAYS` or `EVENT_SEARCH_TOPK`

### Redis cache inconsistency

**Symptom**: Stale profiles returned after engine update.

**Diagnosis**: Check NATS consumer lag for `memobase.profile.changed`.

**Resolution**:
```bash
# Manual cache invalidation
redis-cli DEL "user_profiles::project_id::user_id"
```

### Empty context returned

**Symptom**: GetContext returns empty string.

**Diagnosis**:
```sql
SELECT count(*) FROM user_profiles WHERE user_id = 'xxx' AND project_id = 'yyy';
SELECT count(*) FROM user_event_gists WHERE user_id = 'xxx' AND project_id = 'yyy';
```

**Resolution**: Verify memobase-engine has processed blobs for this user. Check buffer status in memobase-ingestion.

## Deployment

```bash
kubectl rollout restart deployment/memobase-context -n vnp-memory
kubectl rollout undo deployment/memobase-context -n vnp-memory
```

## Monitoring

- **Key Metrics**: Context latency p95, cache hit rate, profiles served/s, event gists searched/s
- **Alerts**: p95 latency > 200ms, cache hit rate < 80%, error rate > 1%

## Escalation

- **L1**: VNP Memory — Memobase team
- **L2**: Platform team (if Redis/DB infrastructure issue)
