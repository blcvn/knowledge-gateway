---
id: DOC-S06
service: graphiti-search
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-search — Runbook (Operations Guide)

## Startup / Shutdown

### Startup Sequence

```bash
# 1. Verify dependencies
grpcurl -plaintext graphiti-store:9024 grpc.health.v1.Health/Check
redis-cli -u redis://localhost:6379 ping
nats server check connection --server nats://localhost:4222

# 2. Start service
go run cmd/server/main.go
# Or: docker compose up graphiti-search
```

**Startup checks:**
1. graphiti-store gRPC → fail-fast if unavailable
2. Redis connection → warn (graceful degradation, search still works uncached)
3. NATS subscription → warn (cache invalidation disabled)

### Shutdown
1. Drain gRPC requests → close Redis → close NATS → flush OTel

## Health Check Endpoints

| Endpoint | Port | Checks |
|----------|------|--------|
| `grpc.health.v1.Health/Check` | 9022 | gRPC alive |
| `GET /healthz` | 9095 | Liveness |
| `GET /readyz` | 9095 | Readiness (store + redis connected) |
| `GET /metrics` | 9095 | Prometheus metrics |

## Common Errors & Resolution

### Error: `graphiti-store unavailable`
- **Cause**: Store service down or circuit breaker open
- **Resolution**: Check graphiti-store health. CB auto-recovers after timeout.

### Error: `Redis connection refused`
- **Impact**: Search works but uncached (every query hits graphiti-store)
- **Resolution**: Check Redis container. Search degrades gracefully.

### Error: `cross-encoder rerank timeout`
- **Cause**: graphiti-pipeline/Rerank RPC too slow (LLM bottleneck)
- **Resolution**: Increase `PIPELINE_TIMEOUT` or fall back to RRF reranking

### Error: `search returned 0 results`
- **Possible causes**: Empty graph, wrong group_id, indexes not built
- **Resolution**: Verify data exists via Neo4j browser. Check BuildIndices was called.

## Monitoring

### Key Metrics

| Metric | Alert Threshold | Action |
|--------|----------------|--------|
| `graphiti_search_duration_seconds` (p95) | >2s | Check reranker latency |
| `graphiti_search_cache_hit_ratio` | <50% | Check NATS invalidation frequency |
| `graphiti_search_results_count{method}` | avg <1 | Verify graph data + indexes |
| `graphiti_search_rerank_duration_seconds{strategy="cross_encoder"}` | >5s | Switch to RRF |

## Deployment & Rollback

```bash
# Deploy
kubectl set image deployment/graphiti-search \
  search=ghcr.io/vnp/graphiti-search:v1.0.0 -n graphiti-system

# Rollback
kubectl rollout undo deployment/graphiti-search -n graphiti-system
```

## Escalation

1. **L1**: Check health, verify Redis + store connectivity
2. **L2**: Check reranker performance, review search quality metrics
3. **L3**: Cross-service search quality issues, Neo4j index optimization
