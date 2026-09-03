---
id: DOC-S06
service: ov-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# ov-search — Runbook

## Startup

```bash
make run-ov-search
docker compose up ov-search postgresql qdrant nats bifrost
```

**Prerequisites**: PostgreSQL + Qdrant + NATS + Bifrost must be running.

## Shutdown

Graceful shutdown via SIGTERM (30s timeout). In-flight searches complete; hotness recompute stops.

## Health Check

```bash
curl http://localhost:9105/healthz
# Expected: {"status": "serving"}

curl http://localhost:9105/readyz
# Expected: {"status": "ready", "checks": {"db": "ok", "qdrant": "ok", "nats": "ok", "bifrost": "ok"}}
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| gRPC UNAVAILABLE | Service not started | Check logs, verify port 9052 |
| Slow search (>500ms) | Qdrant overloaded or cold cache | Check Qdrant metrics, increase replicas |
| No results returned | Embeddings not indexed | Verify NATS subscriber processing `ov.content.written` |
| Hotness scores all zero | Recompute job failed | Check PostgreSQL connectivity, trigger manual recompute |
| Embedding failures | Bifrost unavailable | Check Bifrost health, verify `EMBEDDING_MODEL` |
| Score propagation wrong | Directory tree stale | Re-index parent directories |
| OOM on large queries | Too many results before reranking | Reduce `SEARCH_MAX_RESULTS` |

## Monitoring

- **Metrics**: Prometheus at `:9105/metrics`
  - `ov_search_query_total` — search count by strategy
  - `ov_search_query_duration_seconds` — search latency histogram
  - `ov_search_embedding_upsert_total` — embedding operations
  - `ov_search_hotness_recompute_duration_seconds` — hotness job
  - `ov_search_propagation_depth` — max score propagation depth
- **Traces**: Jaeger via OTel — trace per search with span per pipeline step
- **Logs**: Structured JSON via slog with `request_id`, `account_id`, `query_hash`

## Deployment Checklist

- [ ] PostgreSQL migrations applied (`make migrate-ov-search`)
- [ ] Qdrant collection `ov_embeddings` created with correct config
- [ ] NATS JetStream stream `openviking` exists with subjects `ov.content.*`
- [ ] Bifrost reachable and embedding model loaded

## Escalation

1. On-call checks logs + metrics + Qdrant dashboard
2. If Qdrant issues → check disk, memory, HNSW rebuild status
3. If embedding failures → escalate to Bifrost/LLM team
4. If unresolved in 15min → escalate to OpenViking team lead
