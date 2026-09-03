---
id: DOC-S06
service: cognee-search
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# cognee-search — Runbook

## Startup

```bash
make run-cognee-search
docker compose up cognee-search
```

**Prerequisites**: Neo4j, Qdrant, Redis, NATS, Bifrost running.

## Health Check

```bash
curl http://localhost:9093/healthz
curl http://localhost:9093/readyz
```

## Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| Empty search results | No cognified data | Ensure cognee-cognify pipeline completed |
| Slow GRAPH_COMPLETION | LLM latency | Check Bifrost. Consider CHUNKS for faster results |
| Redis cache miss rate high | TTL too short | Increase REDIS_CACHE_TTL |
| Qdrant timeout | Vector collection too large | Scale Qdrant. Add tenant-scoped filtering |

## Monitoring

- `cognee_search_queries_total` — by search_type
- `cognee_search_latency_seconds` — per strategy
- `cognee_search_cache_hit_ratio` — Redis cache effectiveness

## Escalation

1. On-call checks logs + metrics
2. If LLM → AI platform team; If DB → data infra team
