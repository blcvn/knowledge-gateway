---
id: DOC-S06
service: graphiti-knowledge
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-knowledge — Runbook (Operations Guide)

## Startup / Shutdown

### Startup Sequence

```bash
# 1. Verify dependencies
curl -s http://bifrost:8080/health  # LLM gateway
grpcurl -plaintext graphiti-store:9024 grpc.health.v1.Health/Check

# 2. Start service
go run cmd/server/main.go
```

**Startup checks:**
1. Bifrost connectivity → fail-fast if LLM gateway unavailable
2. graphiti-store gRPC → warn (resolution will fail but extraction works)
3. Prompt templates loaded → fail-fast if any template missing

### Shutdown
1. Wait for in-flight LLM calls to complete (up to `SHUTDOWN_TIMEOUT`)
2. Close HTTP clients → close gRPC connections → flush OTel

## Health Check Endpoints

| Endpoint | Port | Checks |
|----------|------|--------|
| `grpc.health.v1.Health/Check` | 9023 | gRPC alive |
| `GET /healthz` | 9096 | Liveness |
| `GET /readyz` | 9096 | Readiness (Bifrost + store connected) |
| `GET /metrics` | 9096 | Prometheus metrics |

## Common Errors & Resolution

### Error: `LLM timeout (60s)`
- **Cause**: Complex content, large token count, or Bifrost overloaded
- **Resolution**: Increase `LLM_TIMEOUT`, reduce `LLM_MAX_TOKENS`, or check Bifrost health

### Error: `circuit breaker open`
- **Cause**: 5+ consecutive LLM failures
- **Resolution**: Wait 30s for auto-recovery. Check Bifrost → upstream LLM provider status.

### Error: `bulkhead full`
- **Cause**: LLM_MAX_CONCURRENT concurrent requests exceeded
- **Resolution**: Queue is full. Increase `LLM_MAX_CONCURRENT` or reduce ingestion throughput.

### Error: `malformed LLM response`
- **Cause**: LLM returned non-JSON or unexpected format
- **Resolution**: Check prompt template. Response parser retries extraction once. Log raw response for debugging.

### Error: `entity similarity search failed`
- **Cause**: graphiti-store unavailable during resolution
- **Impact**: Entity resolution skipped — all entities created as new (duplicates possible)
- **Resolution**: Check graphiti-store health. Re-run resolution later.

## Monitoring

### Key Metrics

| Metric | Alert Threshold | Action |
|--------|----------------|--------|
| `graphiti_knowledge_llm_duration_seconds{template}` (p95) | >30s | Check Bifrost/LLM provider |
| `graphiti_knowledge_tokens_total{model}` | >1M/hour | Monitor LLM cost |
| `graphiti_knowledge_extraction_entities_count` | avg <1 | Check prompt quality |
| `graphiti_knowledge_resolution_merge_ratio` | >80% | Verify threshold tuning |
| `graphiti_knowledge_bulkhead_active` | >80% of MAX | Scale or throttle input |

## Deployment & Rollback

```bash
kubectl set image deployment/graphiti-knowledge \
  knowledge=ghcr.io/vnp/graphiti-knowledge:v1.0.0 -n graphiti-system
kubectl rollout undo deployment/graphiti-knowledge -n graphiti-system
```

## Escalation

1. **L1**: Check health, verify Bifrost + store connectivity
2. **L2**: Review LLM token usage, check prompt template rendering
3. **L3**: LLM quality issues, entity resolution accuracy problems
