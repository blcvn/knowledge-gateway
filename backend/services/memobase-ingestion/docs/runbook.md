---
id: DOC-S06
service: memobase-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-ingestion — Runbook

## Startup / Shutdown

```bash
# Startup
make run-memobase-ingestion
# Or: docker compose up memobase-ingestion

# Graceful shutdown (SIGTERM)
kill -SIGTERM <pid>
# gRPC server drains in-flight requests (30s grace period)
```

## Health Check

| Endpoint | Port | Expected |
|----------|------|----------|
| gRPC health | 9031 | `SERVING` |
| HTTP /healthz | 9098 | `200 OK {"status":"healthy"}` |
| HTTP /readyz | 9098 | `200 OK` (DB + NATS connected) |

```bash
grpcurl -plaintext localhost:9031 grpc.health.v1.Health/Check
curl http://localhost:9098/healthz
```

## Common Issues

### Buffer entries stuck in "processing"

**Symptom**: Buffer entries remain in `processing` state indefinitely.

**Diagnosis**:
```sql
SELECT count(*), status FROM buffer_zones
WHERE project_id = 'xxx' GROUP BY status;
```

**Resolution**:
```sql
-- Reset stuck entries (after confirming memobase-engine is not processing)
UPDATE buffer_zones SET status = 'idle', updated_at = NOW()
WHERE status = 'processing' AND updated_at < NOW() - INTERVAL '10 minutes';
```

### Buffer not flushing

**Symptom**: Blobs accumulate but no flush events published.

**Diagnosis**:
```sql
SELECT SUM(token_size) FROM buffer_zones
WHERE user_id = 'xxx' AND status = 'idle';
```

**Resolution**: Check `MAX_BUFFER_TOKEN_SIZE` env var. Force flush via gRPC `FlushBuffer` RPC.

### NATS connectivity

**Symptom**: Flush events not reaching memobase-engine.

**Diagnosis**: Check `nats sub 'memobase.buffer.ready'`

**Resolution**: Verify NATS_URL config. Check NATS JetStream stream `memobase` exists.

## Deployment

```bash
# Rolling update
kubectl rollout restart deployment/memobase-ingestion -n vnp-memory

# Rollback
kubectl rollout undo deployment/memobase-ingestion -n vnp-memory
```

## Monitoring

- **Grafana**: `vnp-memory / memobase-ingestion` dashboard
- **Alerts**: Buffer queue depth > 1000, flush failure rate > 5%

## Escalation

- **L1**: VNP Memory — Memobase team
- **L2**: Platform team (if NATS/DB infrastructure issue)
