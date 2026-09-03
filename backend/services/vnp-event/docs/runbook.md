---
id: DOC-S06
service: vnp-event
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-event — Runbook

## Startup / Shutdown

```bash
make run-vnp-event
docker compose up vnp-event postgresql redis nats
# Graceful shutdown: SIGTERM → drain NATS consumers → finish in-flight RPCs (30s) → exit
```

## Health Check

| Endpoint | Port | Expected |
|----------|------|----------|
| gRPC health | 9041 | `SERVING` |
| HTTP /healthz | 9101 | `200 OK {"status":"healthy"}` |
| HTTP /readyz | 9101 | `200 OK` (DB + Redis + NATS connected) |
| HTTP /metrics | 9101 | Prometheus metrics |

```bash
grpcurl -plaintext localhost:9041 grpc.health.v1.Health/Check
curl http://localhost:9101/healthz
```

## Common Issues

### Events not ingested from NATS

**Symptom**: Timeline is empty despite engine activity.

**Diagnosis**:
```bash
# Check NATS consumer status
nats consumer info vnp-events vnp-event-svc
# Check pending/redelivered counts
```

**Resolution**:
- Verify NATS_URL config and JetStream stream `vnp-events` exists
- Check consumer group `vnp-event-svc` is active
- Verify upstream engines are publishing events (e.g. `memobase.memory.flushed`)

### Semantic search returning no results

**Symptom**: `SearchEvents` returns empty results for valid queries.

**Diagnosis**:
```sql
SELECT count(*) FROM timeline_events WHERE embedding IS NOT NULL AND tenant_id = 'xxx';
```

**Resolution**:
- Check `EMBEDDING_PROVIDER` connectivity
- Verify `EMBEDDING_DIM` matches database column dimension
- Ensure pgvector HNSW index exists: `SELECT * FROM pg_indexes WHERE indexname LIKE '%embedding%';`

### Memobase events missing from timeline

**Symptom**: Memobase `memory.flushed` events not appearing.

**Diagnosis**: Check NATS subscription for `memobase.memory.flushed` and `memobase.event.created` subjects.

**Resolution**: Verify memobase-engine is publishing `memobase.event.created` events after pipeline completion.

## Deployment

```bash
kubectl rollout restart deployment/vnp-event -n vnp-memory
kubectl rollout undo deployment/vnp-event -n vnp-memory
```

## Monitoring

- **Key Metrics**: events_created_total (by source_engine), timeline_query_latency_ms, semantic_search_latency_ms, nats_messages_processed_total
- **Alerts**: NATS consumer lag > 1000, event ingestion rate drops > 50%, search latency p95 > 500ms

## Escalation

1. **L1**: Check health endpoints, verify NATS/DB connectivity
2. **L2**: Check OTel traces for NATS consumer, review embedding errors
3. **L3**: Contact VNP Memory — Platform team
