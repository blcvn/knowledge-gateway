---
id: DOC-S06
service: vnp-search-hub
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-search-hub — Runbook

## Startup / Shutdown

```bash
make run-vnp-search-hub
docker compose up vnp-search-hub redis
# Graceful shutdown: SIGTERM → drain in-flight searches → close engine connections → exit
```

## Health Check

| Endpoint | Port | Expected |
|----------|------|----------|
| gRPC health | 9042 | `SERVING` |
| HTTP /healthz | 9102 | `200 OK {"status":"healthy"}` |
| HTTP /readyz | 9102 | `200 OK` (Redis + at least 1 engine reachable) |
| HTTP /metrics | 9102 | Prometheus metrics |

```bash
grpcurl -plaintext localhost:9042 grpc.health.v1.Health/Check
curl http://localhost:9102/healthz
```

## Common Issues

### Recall returning partial results

**Symptom**: Some engines return 0 results or `engine_status.available = false`.

**Diagnosis**:
```bash
# Check engine health cache
redis-cli GET "engine_health::memobase"
# Check circuit breaker states
```

**Resolution**:
- This is **expected behavior** (degraded mode). vnp-search-hub returns results from available engines.
- Check failing engine's health directly (e.g. `grpcurl -plaintext memobase-context:9033 grpc.health.v1.Health/Check`)
- If circuit breaker is OPEN, wait for `CB_TIMEOUT` (15s default)

### Memobase results not appearing in recall

**Symptom**: Recall results missing `source_engine: memobase` entries.

**Diagnosis**:
```bash
# Test direct Memobase search
grpcurl -d '{"user_id":"test","project_id":"test"}' \
  -plaintext memobase-context:9033 \
  memobase.context.v1.MemobaseContextService/GetProfiles
```

**Resolution**:
- Verify `MEMOBASE_CONTEXT_ADDR` is correct
- Check if user has profiles (empty profiles = no results)
- Ensure memobase-engine has processed blobs for the user

### High recall latency (> 5s)

**Symptom**: `Recall` RPC exceeds SLA.

**Diagnosis**: Check per-engine latency in response `engine_status[].latency_ms`.

**Resolution**:
- Identify slowest engine and investigate separately
- Reduce `FAN_OUT_TIMEOUT` to fail fast on slow engines
- Consider using engine filter to exclude slow engines: `engines: ["graphiti","cognee"]`

### Stale cached results

**Symptom**: Recall returns outdated results after data ingestion.

**Resolution**: Reduce `CACHE_TTL` (default 60s) or clear cache:
```bash
redis-cli KEYS "recall::*" | xargs redis-cli DEL
```

## Deployment

```bash
kubectl rollout restart deployment/vnp-search-hub -n vnp-memory
kubectl rollout undo deployment/vnp-search-hub -n vnp-memory
```

## Monitoring

- **Key Metrics**: recall_total, recall_latency_ms, engine_latency_ms (by engine), engine_failures_total (by engine), cache_hit_ratio, rerank_latency_ms
- **Alerts**: All engines failing simultaneously, recall latency p95 > 10s, cache hit ratio < 50%

## Escalation

1. **L1**: Check health, verify Redis connectivity, review engine health cache
2. **L2**: Check OTel traces for fan-out, review per-engine failure patterns
3. **L3**: Contact VNP Memory — Platform team
